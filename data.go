package main

import (
	"bytes"
	"fmt"
	"github.com/alecthomas/chroma"
	chhtml "github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/gomarkdown/markdown/ast"
	mdhtml "github.com/gomarkdown/markdown/html"
	"github.com/pkg/errors"
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/patch"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"github.com/robloxapi/rbxapiref/documents"
	"github.com/robloxapi/rbxapiref/fetch"
	"html"
	"html/template"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

type Data struct {
	Settings      Settings
	Manifest      *Manifest
	CurrentYear   int
	Entities      *Entities
	Templates     *template.Template
	CodeFormatter *chhtml.Formatter
	ResOnly       bool
	Stamp         template.HTML
}

type Patch struct {
	Stale   bool       `json:"-"`
	Prev    *BuildInfo `json:",omitempty"`
	Info    BuildInfo
	Config  string
	Actions []Action
}

// Escape once to escape the file name, then again to escape the URL.
func doubleEscape(s string) string {
	return url.PathEscape(url.PathEscape(s))
}

func pathText(text string) string {
	var s []rune
	var dash bool
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			s = append(s, r)
			dash = false
		} else if !dash && unicode.IsSpace(r) {
			s = append(s, '-')
			dash = true
		}
	}
	return string(s)
}

func anchorText(text string) string {
	var s []rune
	var dash bool
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			s = append(s, unicode.ToLower(r))
			dash = false
		} else if !dash {
			s = append(s, '-')
			dash = true
		}
	}
	return string(s)
}

// FileLink generates a URL, relative to an arbitrary host.
func (data *Data) FileLink(linkType string, args ...string) (s string) {
retry:
	switch strings.ToLower(linkType) {
	case "index":
		s = "index" + FileExt
	case "resource":
		s = path.Join(data.Settings.Output.Resources, path.Join(args...))
	case "docres":
		s = path.Join(data.Settings.Output.DocResources, path.Join(args...))
	case "updates":
		if len(args) > 0 {
			s = path.Join("updates", doubleEscape(args[0])+FileExt)
		} else {
			s = "updates" + FileExt
		}
	case "class":
		s = path.Join(ClassPath, doubleEscape(args[0])+FileExt)
	case "member":
		if len(args) == 1 {
			return (&url.URL{Fragment: MemberAnchorPrefix + args[0]}).String()
		} else if len(args) == 2 {
			s = path.Join(ClassPath, doubleEscape(args[0])+FileExt) +
				(&url.URL{Fragment: MemberAnchorPrefix + args[1]}).String()
		}
	case "enum":
		s = path.Join(EnumPath, doubleEscape(args[0])+FileExt)
	case "enumitem":
		if len(args) == 1 {
			return (&url.URL{Fragment: MemberAnchorPrefix + args[0]}).String()
		} else if len(args) == 2 {
			s = path.Join(EnumPath, doubleEscape(args[0])+FileExt) +
				(&url.URL{Fragment: MemberAnchorPrefix + args[1]}).String()
		}
	case "type":
		if len(args) == 1 {
			s = path.Join(TypePath, doubleEscape(args[0])+FileExt)
		} else if len(args) == 2 {
			switch strings.ToLower(args[0]) {
			case "class", "enum":
				a := make([]string, 2)
				linkType, a[0] = args[0], args[1]
				args = a
				goto retry
			}
			s = path.Join(TypePath, doubleEscape(args[1])+FileExt)
		}
	case "about":
		s = "about" + FileExt
	case "repository":
		return "https://github.com/robloxapi/rbxapiref"
	case "issues":
		return "https://github.com/robloxapi/rbxapiref/issues"
	case "docmon":
		s = "docmon" + FileExt
	case "search":
		s = "search.db"
	case "manifest":
		s = data.Settings.Output.Manifest
	case "devhub":
		switch linkType = strings.ToLower(args[0]); linkType {
		case "class", "enum":
			return "https://" + path.Join(DevHubURL, linkType, pathText(args[1]))
		case "property", "function", "event", "callback":
			return "https://" + path.Join(DevHubURL, linkType, pathText(args[1]), pathText(args[2]))
		case "enumitem":
			return "https://" + path.Join(DevHubURL, "enum", pathText(args[1])) + "#" + anchorText(args[2])
		case "type":
			return "https://" + path.Join(DevHubURL, "datatype", pathText(args[1]))
		}
	}
	s = path.Join("/", data.Settings.Output.Sub, s)
	return s
}

// FilePath generates a file path relative to the output root directory. On a
// web server serving static files, the returned path is meant to point to the
// same file as the file pointed to by the URL generated by FileLink.
func (data *Data) FilePath(typ string, args ...string) string {
	return data.PathFromLink(data.FileLink(typ, args...))
}

// AbsFilePath generates an absolute path located in the Output. On a web
// server serving static files, the returned path is meant to point to the
// same file as the file pointed to by the URL generated by FileLink.
func (data *Data) AbsFilePath(typ string, args ...string) string {
	return data.AbsPathFromLink(data.FileLink(typ, args...))
}

// LinkFromPath transforms a path into a link, if possible.
func (data *Data) LinkFromPath(p string) string {
	if l, err := filepath.Rel(data.Settings.Output.Root, p); err == nil {
		return path.Clean(l)
	}
	return path.Clean(p)
}

// PathFromLink transforms a link into a path, if possible.
func (data *Data) PathFromLink(l string) string {
	l, _ = url.PathUnescape(l)
	l = strings.TrimPrefix(l, "/")
	return filepath.Clean(l)
}

// AbsPathFromLink transforms a link into an absolute path, if possible.
func (data *Data) AbsPathFromLink(l string) string {
	l, _ = url.PathUnescape(l)
	l = strings.TrimPrefix(l, "/")
	return data.AbsPath(l)
}

// AbsPath transforms a relative path into an absolute path.
func (data *Data) AbsPath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(data.Settings.Output.Root, p)
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
	case *ClassEntity:
		class = "class-icon"
		title = "Class"
		if value.Metadata.Instance == nil {
			goto finish
		}
		index = GetMetadataInt(value.Metadata, "ExplorerImageIndex")
	case *MemberEntity:
		if value.Element == nil {
			goto finish
		}
		v = []interface{}{value.Element}
		goto retry
	case *EnumEntity:
		if value.Element == nil {
			goto finish
		}
		v = []interface{}{value.Element}
		goto retry
	case *EnumItemEntity:
		if value.Element == nil {
			goto finish
		}
		v = []interface{}{value.Element}
		goto retry
	case TypeCategory:
		class = "member-icon"
		title = "TypeCategory"
		index = 0
	case *TypeEntity:
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
	var action *Action
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
			t = member.Element
		case "enum":
			enum, ok := data.Entities.Enums[v[1].(string)]
			if !ok {
				return ""
			}
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
			t = item.Element
		default:
			return ""
		}
	case *ClassEntity:
		t = value.Element
	case *MemberEntity:
		t = value.Element
	case *EnumEntity:
		t = value.Element
	case *EnumItemEntity:
		t = value.Element
	case Action:
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
			}
			goto finish
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

func (data *Data) EmbedResource(resource string) (interface{}, error) {
	b, err := ioutil.ReadFile(filepath.Join(data.Settings.Input.Resources, resource))
	switch filepath.Ext(resource) {
	case ".css":
		return template.CSS(b), err
	case ".js":
		return template.JS(b), err
	case ".html", ".svg":
		return template.HTML(b), err
	}
	return string(b), err
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
					Attr{Name: "href", Value: data.FileLink("resource", resource.Name)},
					Attr{Name: "rel", Value: "stylesheet"},
					Attr{Name: "type", Value: "text/css"},
				)
			case ".js":
				ResData.Resource.Attr = append(ResData.Resource.Attr,
					Attr{Name: "src", Value: data.FileLink("resource", resource.Name)},
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

func (data *Data) GenerateCardElements(pages ...*Page) (elements []template.HTML, err error) {
	getField := func(name string) (value string, ok bool) {
		for _, page := range pages {
			if v, k := page.Meta[name]; k {
				value = v
				ok = true
			}
		}
		return value, ok
	}

	elements = append(elements,
		generateMetaTag("property", "og:type", "website"),
		generateMetaTag("name", "twitter:card", "summary"),
	)
	if title, ok := getField("Title"); ok {
		elements = append(elements,
			generateMetaTag("property", "og:title", title),
			generateMetaTag("name", "twitter:title", title),
		)
	}
	if desc, ok := getField("Description"); ok {
		elements = append(elements,
			generateMetaTag("property", "og:description", desc),
			generateMetaTag("name", "twitter:description", desc),
		)
	}
	if image, ok := getField("Image"); ok {
		u := (&url.URL{Scheme: "https", Host: data.Settings.Output.Host, Path: data.FileLink("resource", image)}).String()
		elements = append(elements,
			generateMetaTag("property", "og:image", u),
			generateMetaTag("name", "twitter:image", u),
		)
	}

	return elements, nil
}

func (data *Data) GenerateHistoryElements(entity interface{}, button bool, ascending bool) (template.HTML, error) {
	var patches []Patch
	switch entity := entity.(type) {
	case *ClassEntity:
		patches = MergePatches(entity.Patches, nil, nil)
		for _, member := range entity.MemberList {
			patches = MergePatches(patches, member.Patches, func(action *Action) bool {
				// Filter actions where the parent entity is the cause.
				return action.GetMember() != nil
			})
		}
	case *MemberEntity:
		patches = MergePatches(entity.Patches, nil, nil)
	case *EnumEntity:
		patches = MergePatches(entity.Patches, nil, nil)
		for _, item := range entity.ItemList {
			patches = MergePatches(patches, item.Patches, func(action *Action) bool {
				return action.GetEnumItem() != nil
			})
		}
	case *EnumItemEntity:
		patches = MergePatches(entity.Patches, nil, nil)
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
		First   BuildInfo
		Patches []Patch
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
	files.Add(data.FilePath("manifest"))
	files.Add(data.FilePath("search"))
	for _, page := range pages {
		if page.File != "" {
			files.Add(page.File)
		}
		for _, res := range page.Styles {
			files.Add(data.FilePath("resource", res.Name))
		}
		for _, res := range page.Scripts {
			files.Add(data.FilePath("resource", res.Name))
		}
		for _, res := range page.Resources {
			files.Add(data.FilePath("resource", res.Name))
		}
		for _, res := range page.DocResources {
			files.Add(data.FilePath("docres", res.Name))
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
	root := filepath.Dir(data.AbsFilePath(""))
	err := filepath.Walk(data.AbsFilePath(""), func(path string, info os.FileInfo, err error) error {
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
		dstname := data.AbsFilePath(dstType, name)
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
			if err != nil {
				return errors.WithMessage(err, "create resource")
			}
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

func (data *Data) LatestPatch() Patch {
	return data.Manifest.Patches[len(data.Manifest.Patches)-1]
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
		link = data.FileLink("docres", path)
		return
	case "class":
		slash := strings.IndexByte(path, '/')
		if slash < 0 {
			link = data.FileLink("class", unescapeURLPath(path))
			return
		}
		link = data.FileLink("member", unescapeURLPath(path[:slash]), unescapeURLPath(path[slash+1:]))
		return
	case "enum":
		slash := strings.IndexByte(path, '/')
		if slash < 0 {
			link = data.FileLink("enum", unescapeURLPath(path))
			return
		}
		link = data.FileLink("enumitem", unescapeURLPath(path[:slash]), unescapeURLPath(path[slash+1:]))
		return
	case "type":
		link = data.FileLink("type", unescapeURLPath(path))
		return
	case "member":
		link = data.FileLink("member", unescapeURLPath(path))
		return
	}
	return "", "", ref
}

// Normalizes the references within a document according to ParseDocReference,
// and returns any resources that the document refers to.
func (data *Data) NormalizeDocReferences(document Document) []Resource {
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

	latest := data.LatestPatch()
	client := &fetch.Client{
		Config:    data.Settings.Configs[latest.Config],
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

	renderer := mdhtml.NewRenderer(mdhtml.RendererOptions{
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
				if err := data.CodeFormatter.Format(&buf, StyleRobloxLight, iterator); err != nil {
					return ast.GoToNext, false
				}
				io.Copy(w, &buf)

				return ast.SkipChildren, true
			}
			return ast.GoToNext, false
		},
	})

	docDir := documents.NewDirectorySection(
		data.Settings.Input.Documents,
		documents.MarkdownHandler{
			UseGit:        data.Settings.Input.UseGit,
			StripComments: true,
		}.FileHandler,
	)
	if apiDir := docDir.Query("api"); apiDir != nil {
		for _, entity := range data.Entities.ClassList {
			if entity.Document, _ = apiDir.Query("class", entity.ID).(Document); entity.Document != nil {
				entity.Document.SetRender(renderer)
				for _, member := range entity.MemberList {
					if member.Document, _ = entity.Document.Query("Members", member.ID[1]).(Document); member.Document != nil {
						member.Document.SetRender(renderer)
					}
				}
			}
		}
		for _, entity := range data.Entities.EnumList {
			if entity.Document, _ = apiDir.Query("enum", entity.ID).(Document); entity.Document != nil {
				entity.Document.SetRender(renderer)
				for _, item := range entity.ItemList {
					if item.Document, _ = entity.Document.Query("Members", item.ID[1]).(Document); item.Document != nil {
						item.Document.SetRender(renderer)
					}
				}
			}
		}
		for _, entity := range data.Entities.TypeList {
			if entity.Document, _ = apiDir.Query("type", entity.ID).(Document); entity.Document != nil {
				entity.Document.SetRender(renderer)
			}
		}
	}

	for _, entity := range data.Entities.ClassList {
		for _, member := range entity.MemberList {
			member.DocStatus = GenerateDocStatus(member)
		}
		entity.DocStatus = GenerateDocStatus(entity)
	}
	for _, entity := range data.Entities.EnumList {
		for _, item := range entity.ItemList {
			item.DocStatus = GenerateDocStatus(item)
		}
		entity.DocStatus = GenerateDocStatus(entity)
	}
	for _, entity := range data.Entities.TypeList {
		entity.DocStatus = GenerateDocStatus(entity)
	}
}

func GetDocStatus(entity interface{}) DocStatus {
	if entity, ok := entity.(Documentable); ok {
		return entity.GetDocStatus()
	}
	return GenerateDocStatus(entity)
}

func GenerateDocStatus(entity interface{}) (s DocStatus) {
	setStatus := func(status *int, hasDoc bool, section documents.Section) {
		if !hasDoc {
			*status = 0
		} else if section == nil {
			*status = 1
		} else if count, ok := section.(documents.Countable); ok && count.IsEmpty() {
			*status = 2
		} else {
			*status = 3
		}
	}

	var document Document
	if doc, ok := entity.(Documentable); ok {
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
		s.DetailsSections = count.BlockCount()
	}
	if count, ok := examples.(documents.Countable); ok {
		s.ExampleCount = count.CodeBlockCount()
	}

	var count int
	var total int
	switch entity := entity.(type) {
	case *ClassEntity:
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
		}
	case *MemberEntity:
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
	case *EnumEntity:
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
		for _, item := range entity.ItemList {
			total += 1
			if item.DocStatus.SummaryStatus >= 3 {
				count++
			}
		}
	case *EnumItemEntity:
		// Only include summary. In most cases, details and examples for every
		// single enum item is going overboard.
		total += 1
		if s.SummaryStatus >= 3 {
			count++
		}
	case TypeCategory:
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
	case *TypeEntity:
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
		s.AggregateProgress = float64(count) / float64(total) * 100
	} else {
		s.AggregateProgress = 100
	}
	return s
}
