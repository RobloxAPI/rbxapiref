package main

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alecthomas/chroma"
	chhtml "github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/gomarkdown/markdown/ast"
	mdhtml "github.com/gomarkdown/markdown/html"
	"github.com/pkg/errors"
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/patch"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"github.com/robloxapi/rbxapiref/builds"
	"github.com/robloxapi/rbxapiref/documents"
	"github.com/robloxapi/rbxapiref/entities"
	"github.com/robloxapi/rbxapiref/fetch"
	"github.com/robloxapi/rbxapiref/manifest"
	"github.com/robloxapi/rbxapiref/settings"
)

type Data struct {
	Settings      settings.Settings
	Manifest      *manifest.Manifest
	Time          time.Time
	Entities      *entities.Entities
	Templates     *template.Template
	CodeFormatter *chhtml.Formatter
	ResOnly       bool
}

const IconSize = 16

var memberIconIndex = map[string]int{
	"Property": 6,
	"Function": 4,
	"Event":    11,
	"Callback": 16,
}

func (data *Data) Icon(v ...interface{}) template.HTML {
	if len(v) == 0 {
		return ""
	}
	var class string
	var title string
	var index int
retry:
	switch value := v[0].(type) {
	case string:
		switch strings.ToLower(value) {
		case "class":
			entity, ok := data.Entities.Classes[v[1].(string)]
			if !ok {
				goto finish
			}
			v = []interface{}{entity}
			goto retry
		case "member":
			entity := data.Entities.Members[[2]string{v[1].(string), v[2].(string)}]
			if entity == nil {
				goto finish
			}
			v = []interface{}{entity.Element}
			goto retry
		case "enum":
			class = "enum-icon"
			title = "Enum"
			index = -1
		case "enumitem":
			class = "enum-item-icon"
			title = "EnumItem"
			index = -1
		}
	case *entities.ClassEntity:
		class = "class-icon"
		title = "Class"
		if value.Metadata.Instance == nil {
			goto finish
		}
		index = value.Metadata.GetInt("ExplorerImageIndex")
	case *entities.MemberEntity:
		if value.Element == nil {
			goto finish
		}
		v = []interface{}{value.Element}
		goto retry
	case *entities.EnumEntity:
		if value.Element == nil {
			goto finish
		}
		v = []interface{}{value.Element}
		goto retry
	case *entities.EnumItemEntity:
		if value.Element == nil {
			goto finish
		}
		v = []interface{}{value.Element}
		goto retry
	case entities.TypeCategory:
		class = "member-icon"
		title = "TypeCategory"
		index = 0
	case *entities.TypeEntity:
		class = "member-icon"
		title = "Type"
		index = 3
	case *rbxapijson.Class:
		entity, ok := data.Entities.Classes[value.Name]
		if !ok {
			goto finish
		}
		v = []interface{}{entity}
		goto retry
	case rbxapi.Member:
		class = "member-icon"
		title = value.GetMemberType()
		index = memberIconIndex[title]
		switch v := value.(type) {
		case interface{ GetSecurity() (string, string) }:
			r, w := v.GetSecurity()
			if r == "None" {
				r = ""
			}
			if w == "None" {
				w = ""
			}
			switch {
			case r != "" && w != "":
				title = "Protected " + title
				if r == w {
					title += " (Read/Write: " + r + ")"
				} else {
					title += " (Read: " + r + " / Write: " + w + ")"
				}
				index++
			case r != "":
				title = "Protected " + title + " (Read: " + r + ")"
				index++
			case w != "":
				title = "Protected " + title + " (Write: " + w + ")"
				index++
			default:
			}
		case interface{ GetSecurity() string }:
			s := v.GetSecurity()
			if s != "" && s != "None" {
				title = "Protected " + title + " (" + s + ")"
				index++
			}
		}
	case *rbxapijson.Enum:
		class = "enum-icon"
		title = "Enum"
		index = -1
	case *rbxapijson.EnumItem:
		class = "enum-item-icon"
		title = "EnumItem"
		index = -1
	}
finish:
	var style string
	if index >= 0 {
		style = fmt.Sprintf(` style="--icon-index: %d"`, index)
	}
	const body = `<span class="icon %s" title="%s"%s></span>`
	return template.HTML(fmt.Sprintf(body, template.HTMLEscapeString(class), template.HTMLEscapeString(title), style))
}

// Sorted by most to least permissive.
var securityContexts = map[string]int{
	"None":                  0,
	"RobloxPlaceSecurity":   1,
	"PluginSecurity":        2,
	"LocalUserSecurity":     3,
	"RobloxScriptSecurity":  4,
	"RobloxSecurity":        5,
	"NotAccessibleSecurity": 6,
}

func (data *Data) ElementStatusClasses(suffix bool, v ...interface{}) string {
	var t rbxapi.Taggable
	var action *builds.Action
	var removed bool
	switch value := v[0].(type) {
	case rbxapi.Taggable:
		t = value
	case string:
		switch value {
		case "class":
			class, ok := data.Entities.Classes[v[1].(string)]
			if !ok {
				return ""
			}
			removed = class.Removed
			t = class.Element
		case "member":
			class, ok := data.Entities.Classes[v[1].(string)]
			if !ok {
				return ""
			}
			member, ok := class.Members[v[2].(string)]
			if !ok {
				return ""
			}
			removed = member.Removed
			t = member.Element
		case "enum":
			enum, ok := data.Entities.Enums[v[1].(string)]
			if !ok {
				return ""
			}
			removed = enum.Removed
			t = enum.Element
		case "enumitem":
			enum, ok := data.Entities.Enums[v[1].(string)]
			if !ok {
				return ""
			}
			item, ok := enum.Items[v[2].(string)]
			if !ok {
				return ""
			}
			removed = item.Removed
			t = item.Element
		default:
			return ""
		}
	case *entities.ClassEntity:
		removed = value.Removed
		t = value.Element
	case *entities.MemberEntity:
		removed = value.Removed
		t = value.Element
	case *entities.EnumEntity:
		removed = value.Removed
		t = value.Element
	case *entities.EnumItemEntity:
		removed = value.Removed
		t = value.Element
	case builds.Action:
		action = &value
		t, _ = value.GetElement().(rbxapi.Taggable)
	default:
		return ""
	}

	var s []string
	if action != nil {
		switch action.Type {
		case patch.Change:
			switch action.Field {
			case "Security", "ReadSecurity", "WriteSecurity":
				// Select the most permissive context. This will cause changes
				// to visible contexts to always be displayed.
				p, _ := action.GetPrev().(string)
				n, _ := action.GetNext().(string)
				if securityContexts[p] < securityContexts[n] {
					if p != "" && p != "None" {
						s = append(s, "api-sec-"+p)
					}
				} else {
					if n != "" && n != "None" {
						s = append(s, "api-sec-"+n)
					}
				}
				goto finish
			case "Tags":
				// Include tag unless it is being changed.
				p := rbxapijson.Tags(action.GetPrev().([]string))
				n := rbxapijson.Tags(action.GetNext().([]string))
				if p.GetTag("Deprecated") && n.GetTag("Deprecated") {
					s = append(s, "api-deprecated")
				}
				if p.GetTag("NotBrowsable") && n.GetTag("NotBrowsable") {
					s = append(s, "api-not-browsable")
				}
				if p.GetTag("Hidden") && n.GetTag("Hidden") {
					s = append(s, "api-hidden")
				}
				goto finish
			}
		}
	}

	for _, tag := range t.GetTags() {
		switch tag {
		case "Deprecated":
			s = append(s, "api-deprecated")
		case "NotBrowsable":
			s = append(s, "api-not-browsable")
		case "Hidden":
			s = append(s, "api-hidden")
		}
	}
	if removed {
		s = append(s, "api-removed")
	}
	switch m := t.(type) {
	case interface{ GetSecurity() (string, string) }:
		r, w := m.GetSecurity()
		if r == w {
			if r != "" && r != "None" {
				s = append(s, "api-sec-"+r)
			}
		} else {
			s = append(s, "api-sec-"+r)
			s = append(s, "api-sec-"+w)
		}
	case interface{ GetSecurity() string }:
		if v := m.GetSecurity(); v != "" && v != "None" {
			s = append(s, "api-sec-"+v)
		}
	}

finish:
	if len(s) == 0 {
		return ""
	}

	sort.Strings(s)
	j := 0
	for i := 1; i < len(s); i++ {
		if s[j] != s[i] {
			j++
			s[j] = s[i]
		}
	}
	s = s[:j+1]

	classes := strings.Join(s, " ")
	if suffix {
		classes = " " + classes
	}
	return classes
}

func (data *Data) ExecuteTemplate(name string, tdata interface{}) (template.HTML, error) {
	var buf bytes.Buffer
	err := data.Templates.ExecuteTemplate(&buf, name, tdata)
	return template.HTML(buf.String()), err
}

func (data *Data) GenerateResourceElements(resources []Resource) (v []interface{}, err error) {
	for _, resource := range resources {
		var ResData struct {
			Type     string
			Resource Resource
			Content  interface{}
		}
		ResData.Type = filepath.Ext(resource.Name)
		ResData.Resource = resource
		ResData.Resource.Attr = nil
		if resource.Embed {
			var content []byte
			if resource.Content != nil {
				content = resource.Content
			} else {
				filename := filepath.Join(data.Settings.Input.Resources, resource.Name)
				if content, err = ioutil.ReadFile(filename); err != nil {
					return nil, err
				}
			}
			switch ResData.Type {
			case ".css":
				ResData.Content = template.CSS(content)
			case ".js":
				ResData.Content = template.JS(content)
			case ".html", ".svg":
				ResData.Content = template.HTML(content)
			default:
				ResData.Content = string(content)
			}
		} else {
			switch ResData.Type {
			case ".css":
				ResData.Resource.Attr = append(ResData.Resource.Attr,
					Attr{Name: "href", Value: data.Settings.Output.FileLink("resource", resource.Name)},
					Attr{Name: "rel", Value: "stylesheet"},
					Attr{Name: "type", Value: "text/css"},
				)
			case ".js":
				ResData.Resource.Attr = append(ResData.Resource.Attr,
					Attr{Name: "src", Value: data.Settings.Output.FileLink("resource", resource.Name)},
					Attr{Name: "charset", Value: "utf-8"},
				)
			}
		}
		ResData.Resource.Attr.Merge(resource.Attr)
		r, err := data.ExecuteTemplate("resource", ResData)
		if err != nil {
			return nil, err
		}
		v = append(v, r)
	}
	return v, nil
}

func generateMetaTag(a, b, c string) template.HTML {
	return template.HTML("<meta " + html.EscapeString(a) + "=\"" + html.EscapeString(b) + "\" content=\"" + html.EscapeString(c) + "\" />")
}

func (data *Data) GenerateHistoryElements(entity interface{}, button bool, ascending bool) (template.HTML, error) {
	var patches []builds.Patch
	switch entity := entity.(type) {
	case *entities.ClassEntity:
		patches = builds.MergePatches(entity.Patches, nil, nil)
		for _, member := range entity.MemberList {
			patches = builds.MergePatches(patches, member.Patches, func(action *builds.Action) bool {
				// Filter actions where the parent entity is the cause.
				return action.GetMember() != nil
			})
		}
	case *entities.MemberEntity:
		patches = builds.MergePatches(entity.Patches, nil, nil)
	case *entities.EnumEntity:
		patches = builds.MergePatches(entity.Patches, nil, nil)
		for _, item := range entity.ItemList {
			patches = builds.MergePatches(patches, item.Patches, func(action *builds.Action) bool {
				return action.GetEnumItem() != nil
			})
		}
	case *entities.EnumItemEntity:
		patches = builds.MergePatches(entity.Patches, nil, nil)
	default:
		return "", nil
	}
	if len(patches) == 0 {
		return "", nil
	}
	if len(patches) == 1 && data.Manifest.Patches[0].Info.Equal(patches[0].Info) {
		return "", nil
	}
	if ascending {
		sort.Slice(patches, func(i, j int) bool {
			return patches[i].Info.Date.Before(patches[j].Info.Date)
		})
	} else {
		sort.Slice(patches, func(i, j int) bool {
			return patches[i].Info.Date.After(patches[j].Info.Date)
		})
	}
	return data.ExecuteTemplate("history", struct {
		First   builds.Info
		Patches []builds.Patch
		Button  bool
	}{data.Manifest.Patches[0].Info, patches, button})
}

func (data *Data) GeneratePages(generators []PageGenerator) (pages []Page) {
	for _, generator := range generators {
		pages = append(pages, generator(data)...)
	}
	return pages
}

type FileSet struct {
	root string
	set  map[string]struct{}
}

func NewFileSet(root string, file ...string) *FileSet {
	files := &FileSet{root: root, set: map[string]struct{}{}}
	for _, file := range file {
		files.Add(file)
	}
	return files
}

func (files *FileSet) Add(file string) {
	files.set[file] = struct{}{}
}

func (files *FileSet) Has(file string) bool {
	_, ok := files.set[file]
	return ok
}

func (files *FileSet) Files() []string {
	fs := make([]string, 0, len(files.set))
	for file := range files.set {
		fs = append(fs, file)
	}
	sort.Strings(fs)
	return fs
}

func (data *Data) ComparePages(pages []Page) error {
	// Accumulate generated files.
	files := NewFileSet("")
	files.Add(data.Settings.Output.FilePath("manifest"))
	files.Add(data.Settings.Output.FilePath("search"))
	for _, page := range pages {
		if page.File != "" {
			files.Add(page.File)
		}
		for _, res := range page.Styles {
			files.Add(data.Settings.Output.FilePath("resource", res.Name))
		}
		for _, res := range page.Scripts {
			files.Add(data.Settings.Output.FilePath("resource", res.Name))
		}
		for _, res := range page.Resources {
			files.Add(data.Settings.Output.FilePath("resource", res.Name))
		}
		for _, res := range page.DocResources {
			files.Add(data.Settings.Output.FilePath("docres", res.Name))
		}
	}
	// Include directories.
	for _, file := range files.Files() {
		for {
			dir := filepath.Dir(file)
			if dir == file || dir == string(filepath.Separator) || dir == "." {
				break
			}
			file = dir
			files.Add(dir)
		}
	}

	// Walk the output tree.
	dirs := []string{}
	root := filepath.Dir(data.Settings.Output.AbsFilePath(""))
	err := filepath.Walk(data.Settings.Output.AbsFilePath(""), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if path, err = filepath.Rel(root, path); err != nil {
			return nil
		}
		// Skip "hidden" files that start with a dot.
		if info.Name()[0] == '.' {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip generated files.
		if files.Has(path) {
			return nil
		}
		if info.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		return os.Remove(filepath.Join(root, path))
	})
	if err != nil {
		return err
	}
	for _, path := range dirs {
		if err := os.Remove(filepath.Join(root, path)); err != nil {
			return err
		}
	}
	return nil
}

func (data *Data) RenderPageDirs(pages []Page) error {
	dirs := map[string]struct{}{}
	for _, page := range pages {
		dir := filepath.Join(data.Settings.Output.Root, filepath.Dir(page.File))
		if _, ok := dirs[dir]; ok {
			continue
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errors.WithMessage(err, "make directory")
		}
		dirs[dir] = struct{}{}
	}
	return nil
}

func (data *Data) copyResources(srcPath, dstType string, resources map[string]*Resource) error {
	dirs := map[string]struct{}{}
	for name, resource := range resources {
		var src io.ReadCloser
		if resource.Content != nil {
			src = ioutil.NopCloser(bytes.NewReader(resource.Content))
		} else {
			var err error
			if src, err = os.Open(filepath.Join(srcPath, name)); err != nil {
				return errors.WithMessage(err, "open resource")
			}
		}
		dstname := data.Settings.Output.AbsFilePath(dstType, name)
		dir := filepath.Dir(dstname)
		if _, ok := dirs[dir]; !ok {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return errors.WithMessage(err, "make directory")
			}
			dirs[dir] = struct{}{}
		}
		dst, err := os.Create(dstname)
		if err != nil {
			src.Close()
			return errors.WithMessage(err, "create resource")
		}
		_, err = io.Copy(dst, src)
		dst.Close()
		src.Close()
		if err != nil {
			return errors.WithMessage(err, "write resource")
		}
	}
	return nil
}

type Resources map[string]*Resource

func (r Resources) Add(resource *Resource) {
	// Avoid empty, embedded, or ignored resources.
	if resource.Name == "" || resource.Embed || resource.Ignore {
		return
	}
	if res, ok := r[resource.Name]; ok {
		// Prioritize resources with internal content.
		if res.Content != nil {
			return
		}
	}
	r[resource.Name] = resource
}

func (data *Data) RenderResources(pages []Page) error {
	resources := Resources{}
	docres := Resources{}
	for _, page := range pages {
		for i := range page.Styles {
			resources.Add(&page.Styles[i])
		}
		for i := range page.Scripts {
			resources.Add(&page.Scripts[i])
		}
		for i := range page.Resources {
			resources.Add(&page.Resources[i])
		}
		for i := range page.DocResources {
			docres.Add(&page.DocResources[i])
		}
	}
	if err := data.copyResources(data.Settings.Input.Resources, "resource", resources); err != nil {
		return err
	}
	return data.copyResources(data.Settings.Input.DocResources, "docres", docres)
}

func (data *Data) RenderPages(pages []Page) error {
	var rootData struct {
		Data     *Data
		MainPage *Page
		Page     *Page
	}
	rootData.Data = data
	// Treat first page with unspecified filename as main page.
	for _, page := range pages {
		if page.File == "" {
			rootData.MainPage = &page
			break
		}
	}
	if rootData.MainPage == nil {
		return errors.New("no main template")
	}
	for _, page := range pages {
		if page.File == "" {
			continue
		}
		file, err := os.Create(filepath.Join(data.Settings.Output.Root, page.File))
		if err != nil {
			return errors.WithMessage(err, "create file")
		}
		if page.Data == nil {
			page.Data = data
		}
		rootData.Page = &page
		err = data.Templates.ExecuteTemplate(file, rootData.MainPage.Template, rootData)
		file.Close()
		if err != nil {
			return errors.WithMessage(err, "generate page")
		}
	}
	return nil
}

func unescapeURLPath(path string) string {
	p, err := url.PathUnescape(path)
	if err != nil {
		return path
	}
	return p
}

func (data *Data) ParseDocReference(ref string) (scheme, path, link string) {
	colon := strings.IndexByte(ref, ':')
	if colon < 0 {
		return "", "", ref
	}
	switch scheme, path = ref[:colon], ref[colon+1:]; scheme {
	case "res":
		link = data.Settings.Output.FileLink("docres", path)
		return
	case "class":
		slash := strings.IndexByte(path, '/')
		if slash < 0 {
			link = data.Settings.Output.FileLink("class", unescapeURLPath(path))
			return
		}
		link = data.Settings.Output.FileLink("member", unescapeURLPath(path[:slash]), unescapeURLPath(path[slash+1:]))
		return
	case "enum":
		slash := strings.IndexByte(path, '/')
		if slash < 0 {
			link = data.Settings.Output.FileLink("enum", unescapeURLPath(path))
			return
		}
		link = data.Settings.Output.FileLink("enumitem", unescapeURLPath(path[:slash]), unescapeURLPath(path[slash+1:]))
		return
	case "type":
		link = data.Settings.Output.FileLink("type", unescapeURLPath(path))
		return
	case "member":
		link = data.Settings.Output.FileLink("member", unescapeURLPath(path))
		return
	}
	return "", "", ref
}

// Normalizes the references within a document according to ParseDocReference,
// and returns any resources that the document refers to.
func (data *Data) NormalizeDocReferences(document entities.Document) []Resource {
	doc, ok := document.(documents.Linkable)
	if !ok {
		return nil
	}
	resources := map[string]*Resource{}
	doc.SetLinks(func(link string) string {
		scheme, path, link := data.ParseDocReference(link)
		if scheme == "res" {
			if _, ok := resources[path]; !ok {
				resources[path] = &Resource{Name: path}
			}
		}
		return link
	})
	docres := make([]Resource, 0, len(resources))
	for _, resource := range resources {
		docres = append(docres, *resource)
	}
	sort.Slice(docres, func(i, j int) bool {
		return docres[i].Name < docres[j].Name
	})
	return docres
}

func (data *Data) GenerateMetadata() error {
	if data.ResOnly {
		return nil
	}

	latest := data.Manifest.Patches[len(data.Manifest.Patches)-1]
	client := &fetch.Client{
		Config:    data.Settings.Build.Configs[latest.Config],
		CacheMode: fetch.CacheTemp,
	}
	rmd, err := client.ReflectionMetadata(latest.Info.Hash)
	if err != nil {
		return errors.WithMessagef(err, "fetch metadata %s:", latest.Info.Hash)
	}

	for _, list := range rmd.Instances {
		switch list.ClassName {
		case "ReflectionMetadataClasses":
			for _, class := range list.Children {
				if class.ClassName != "ReflectionMetadataClass" {
					continue
				}
				entity := data.Entities.Classes[class.Name()]
				if entity == nil {
					continue
				}
				entity.Metadata.Instance = class

				for _, memberTypeList := range class.Children {
					for _, member := range memberTypeList.Children {
						if member.ClassName != "ReflectionMetadataMember" {
							continue
						}
						entity := entity.Members[member.Name()]
						if entity == nil {
							continue
						}
						entity.Metadata.Instance = member
					}
				}
			}
		case "ReflectionMetadataEnums":
			for _, enum := range list.Children {
				if enum.ClassName != "ReflectionMetadataEnum" {
					continue
				}
				entity := data.Entities.Enums[enum.Name()]
				if entity == nil {
					continue
				}
				entity.Metadata.Instance = enum

				for _, item := range enum.Children {
					if item.ClassName != "ReflectionMetadataEnumItem" {
						continue
					}
					entity := entity.Items[item.Name()]
					if entity == nil {
						continue
					}
					entity.Metadata.Instance = item
				}
			}
		}
	}
	return nil
}

func (data *Data) GenerateDocuments() {
	if data.Settings.Input.Documents == "" {
		return
	}

	data.CodeFormatter = chhtml.New(
		chhtml.WithClasses(),
		chhtml.TabWidth(4),
		chhtml.WithLineNumbers(),
		chhtml.LineNumbersInTable(),
	)

	if data.ResOnly {
		return
	}

	renderer := func() *mdhtml.Renderer {
		return mdhtml.NewRenderer(mdhtml.RendererOptions{
			HeadingIDPrefix: "doc-",
			RenderNodeHook: func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
				switch node := node.(type) {
				case *ast.CodeBlock:
					// io.WriteString
					i := bytes.IndexAny(node.Info, "\t ")
					if i < 0 {
						i = len(node.Info)
					}
					lang := string(node.Info[:i])
					lexer := lexers.Get(lang)
					if lexer == nil {
						return ast.GoToNext, false
					}
					lexer = chroma.Coalesce(lexer)
					iterator, err := lexer.Tokenise(nil, string(node.Literal))
					if err != nil {
						return ast.GoToNext, false
					}
					var buf bytes.Buffer
					if err := data.CodeFormatter.Format(&buf, settings.StyleRobloxLight, iterator); err != nil {
						return ast.GoToNext, false
					}
					io.Copy(w, &buf)

					return ast.SkipChildren, true
				}
				return ast.GoToNext, false
			},
		})
	}

	docDir := documents.NewDirectorySection(
		data.Settings.Input.Documents,
		documents.MarkdownHandler{
			UseGit:        data.Settings.Input.UseGit,
			StripComments: true,
		}.FileHandler,
	)
	if apiDir := docDir.Query("api"); apiDir != nil {
		for _, entity := range data.Entities.ClassList {
			if entity.Document, _ = apiDir.Query("class", entity.ID).(entities.Document); entity.Document != nil {
				entity.Document.SetRender(renderer())
				GenerateDocumentTypeIDs(entity.Document)
				for _, member := range entity.MemberList {
					if member.Document, _ = entity.Document.Query("Members", member.ID[1]).(entities.Document); member.Document != nil {
						member.Document.SetRender(renderer())
					}
				}
			}
		}
		for _, entity := range data.Entities.EnumList {
			if entity.Document, _ = apiDir.Query("enum", entity.ID).(entities.Document); entity.Document != nil {
				entity.Document.SetRender(renderer())
				for _, item := range entity.ItemList {
					if item.Document, _ = entity.Document.Query("Members", item.ID[1]).(entities.Document); item.Document != nil {
						item.Document.SetRender(renderer())
					}
				}
			}
		}
		for _, entity := range data.Entities.TypeList {
			if entity.Document, _ = apiDir.Query("type", entity.ID).(entities.Document); entity.Document != nil {
				entity.Document.SetRender(renderer())
				GenerateDocumentTypeIDs(entity.Document)
			}
		}
	}

	total := float64(len(data.Entities.ClassList) +
		len(data.Entities.EnumList) +
		len(data.Entities.TypeList))
	var count float64
	for _, entity := range data.Entities.ClassList {
		for _, member := range entity.MemberList {
			member.DocStatus = GenerateDocStatus(member)
		}
		entity.DocStatus = GenerateDocStatus(entity)
		count += entity.DocStatus.AggregateProgress
	}
	for _, entity := range data.Entities.EnumList {
		for _, item := range entity.ItemList {
			item.DocStatus = GenerateDocStatus(item)
		}
		entity.DocStatus = GenerateDocStatus(entity)
		count += entity.DocStatus.AggregateProgress
	}
	for _, entity := range data.Entities.TypeList {
		entity.DocStatus = GenerateDocStatus(entity)
		count += entity.DocStatus.AggregateProgress
	}
	data.Entities.Coverage = float32(count / total)
}

func GenerateDocStatus(entity interface{}) (s entities.DocStatus) {
	setStatus := func(status *int, hasDoc bool, section documents.Section) {
		if !hasDoc {
			*status = 0
		} else if section == nil {
			*status = 1
		} else if count, ok := section.(documents.Countable); ok && count.Count() == 0 {
			*status = 2
		} else {
			*status = 3
		}
	}

	var document entities.Document
	if doc, ok := entity.(entities.Documentable); ok {
		document = doc.GetDocument()
	}
	var summary documents.Section
	var details documents.Section
	var examples documents.Section
	if document != nil {
		s.HasDocument = true
		if summary = document.Query("Summary"); summary == nil {
			if summary = document.Query(""); summary != nil {
				s.SummaryOrphaned = true
			}
		}
		details = document.Query("Details")
		examples = document.Query("Examples")
	}
	setStatus(&s.SummaryStatus, s.HasDocument, summary)
	setStatus(&s.DetailsStatus, s.HasDocument, details)
	setStatus(&s.ExamplesStatus, s.HasDocument, examples)
	if count, ok := details.(documents.Countable); ok {
		s.DetailsSections = count.Count()
	}
	if count, ok := examples.(documents.Countable); ok {
		s.ExampleCount = count.Count()
	}

	var count int
	var total int
	switch entity := entity.(type) {
	case *entities.ClassEntity:
		total += 3
		if s.SummaryStatus >= 3 {
			count++
		}
		if s.DetailsStatus >= 3 {
			count++
		}
		if s.ExamplesStatus >= 3 {
			count++
		}
		for _, member := range entity.MemberList {
			total += 3
			if member.DocStatus.SummaryStatus >= 3 {
				count++
			}
			if member.DocStatus.DetailsStatus >= 3 {
				count++
			}
			if member.DocStatus.ExamplesStatus >= 3 {
				count++
			}
			if s.HasDocument {
				// Entity has a document, which means all members have a
				// document.
				if member.DocStatus.SummaryStatus == 0 {
					member.DocStatus.SummaryStatus = 1
				}
				if member.DocStatus.DetailsStatus == 0 {
					member.DocStatus.DetailsStatus = 1
				}
				if member.DocStatus.ExamplesStatus == 0 {
					member.DocStatus.ExamplesStatus = 1
				}
				if member.DocStatus.AggregateStatus == 0 {
					member.DocStatus.AggregateStatus = 1
				}
			}
		}
	case *entities.MemberEntity:
		total += 3
		if s.SummaryStatus >= 3 {
			count++
		}
		if s.DetailsStatus >= 3 {
			count++
		}
		if s.ExamplesStatus >= 3 {
			count++
		}
	case *entities.EnumEntity:
		// Examples not required for enums.
		total += 2
		if s.SummaryStatus >= 3 {
			count++
		}
		if s.DetailsStatus >= 3 {
			count++
		}
		// Show no status unless section is provided.
		if s.ExamplesStatus < 2 {
			s.ExamplesStatus = 0
		}
		for _, item := range entity.ItemList {
			total += 1
			if item.DocStatus.SummaryStatus >= 3 {
				count++
			}
			if s.HasDocument {
				if item.DocStatus.SummaryStatus == 0 {
					item.DocStatus.SummaryStatus = 1
				}
				if item.DocStatus.AggregateStatus == 0 {
					item.DocStatus.AggregateStatus = 1
				}
			}
		}
	case *entities.EnumItemEntity:
		// Only include summary. In most cases, details and examples for every
		// single enum item is going overboard.
		total += 1
		if s.SummaryStatus >= 3 {
			count++
		}
		if s.DetailsStatus < 2 {
			s.DetailsStatus = 0
		}
		if s.ExamplesStatus < 2 {
			s.ExamplesStatus = 0
		}
	case entities.TypeCategory:
		for _, typ := range entity.Types {
			total += 3
			if typ.DocStatus.SummaryStatus >= 3 {
				count++
			}
			if typ.DocStatus.DetailsStatus >= 3 {
				count++
			}
			if typ.DocStatus.ExamplesStatus >= 3 {
				count++
			}
		}
	case *entities.TypeEntity:
		total += 3
		if s.SummaryStatus >= 3 {
			count++
		}
		if s.DetailsStatus >= 3 {
			count++
		}
		if s.ExamplesStatus >= 3 {
			count++
		}
	}

	switch {
	case count == total:
		s.AggregateStatus = 3
	case count > 0:
		s.AggregateStatus = 2
	case count == 0 && s.HasDocument:
		s.AggregateStatus = 1
	}
	if total > 0 {
		s.AggregateProgress = float64(count) / float64(total)
	} else {
		s.AggregateProgress = 1
	}
	return s
}
