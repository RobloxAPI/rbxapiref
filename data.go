package main

import (
	"bytes"
	"fmt"
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"github.com/robloxapi/rbxapiref/fetch"
	"github.com/robloxapi/rbxfile"
	"html"
	"html/template"
	"io/ioutil"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	ClassPath           = "class"
	EnumPath            = "enum"
	TypePath            = "type"
	FileExt             = ".html"
	MemberAnchorPrefix  = "member-"
	SectionAnchorPrefix = "section-"
	MainTitle           = "Roblox API Reference"
	TitleSep            = "-"
)

type Data struct {
	Settings    Settings
	CurrentYear int

	Patches  []Patch
	Latest   *Build
	Metadata ReflectionMetadata

	EarliestYear  int
	LatestYear    int
	PatchesByYear []PatchSet
	LatestPatches PatchSet

	Entities  *Entities
	TreeRoots []*ClassEntity

	Templates *template.Template
}

type Build struct {
	Config string
	Info   BuildInfo
	API    *rbxapijson.Root
}

type Patch struct {
	Stale   bool       `json:"-"`
	Prev    *BuildInfo `json:",omitempty"`
	Info    BuildInfo
	Config  string
	Actions []Action
}

type PatchSet struct {
	Year    int
	Years   []int
	Patches []*Patch
}

// Escape once to escape the file name, then again to escape the URL.
func doubleEscape(s string) string {
	return url.PathEscape(url.PathEscape(s))
}

// FileLink generates a URL, relative to an arbitrary host.
func (data *Data) FileLink(linkType string, args ...string) (s string) {
retry:
	switch strings.ToLower(linkType) {
	case "index":
		s = "index" + FileExt
	case "resource":
		s = path.Join(data.Settings.Output.Resources, path.Join(args...))
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
		switch strings.ToLower(args[0]) {
		case "class", "enum":
			a := make([]string, 2)
			linkType, a[0] = args[0], args[1]
			args = a
			goto retry
		}
		s = path.Join(TypePath, doubleEscape(args[1])+FileExt)
	case "about":
		s = "about" + FileExt
	case "repository":
		return "https://github.com/robloxapi/rbxapiref"
	case "issues":
		return "https://github.com/robloxapi/rbxapiref/issues"
	case "search":
		s = "search.db"
	case "devhub":
		switch linkType = strings.ToLower(args[0]); linkType {
		case "class", "enumitem", "enum":
			return path.Join(DevHubURL, linkType, doubleEscape(args[1]))
		case "property", "function", "event", "callback":
			return path.Join(DevHubURL, linkType, doubleEscape(args[1]), doubleEscape(args[2]))
		case "type":
			return path.Join(DevHubURL, "datatype", doubleEscape(args[1]))
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
		return l
	}
	return p
}

// PathFromLink transforms a link into a path, if possible.
func (data *Data) PathFromLink(l string) string {
	l, _ = url.PathUnescape(l)
	return l
}

// AbsPathFromLink transforms a link into an absolute path, if possible.
func (data *Data) AbsPathFromLink(l string) string {
	l, _ = url.PathUnescape(l)
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
			class = "class-icon"
			title = "Class"
			meta, ok := data.Metadata.Classes[v[1].(string)]
			if !ok {
				goto finish
			}
			index = meta.ExplorerImageIndex
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
		if value.Element == nil {
			goto finish
		}
		v = []interface{}{value.Element}
		goto retry
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
	case *rbxapijson.Class:
		class = "class-icon"
		title = "Class"
		meta, ok := data.Metadata.Classes[value.Name]
		if !ok {
			goto finish
		}
		index = meta.ExplorerImageIndex
	case rbxapi.Member:
		class = "member-icon"
		title = value.GetMemberType()
		index = memberIconIndex[title]
		if len(v) > 1 && v[1].(bool) == false {
			goto finish
		}
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

func (data *Data) Title(sub string) string {
	if sub != "" {
		return sub + " " + TitleSep + " " + MainTitle
	}
	return MainTitle
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
		}
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

func (data *Data) GenerateHistoryElements(entity interface{}, button bool) (template.HTML, error) {
	var patches []Patch
	switch entity := entity.(type) {
	case *ClassEntity:
		patches = entity.Patches
	case *MemberEntity:
		patches = entity.Patches
	case *EnumEntity:
		patches = entity.Patches
	case *EnumItemEntity:
		patches = entity.Patches
	default:
		return "", nil
	}
	if len(patches) == 0 {
		return "", nil
	}
	if len(patches) == 1 && data.Patches[0].Info.Equal(patches[0].Info) {
		return "", nil
	}
	return data.ExecuteTemplate("history", struct {
		First   BuildInfo
		Patches []Patch
		Button  bool
	}{data.Patches[0].Info, patches, button})
}

func (data *Data) GenerateTree() {
	for id, eclass := range data.Entities.Classes {
		super := eclass.Element.Superclass
		if !eclass.Removed {
			if s := data.Entities.Classes[super]; s == nil || s.Removed {
				data.TreeRoots = append(data.TreeRoots, eclass)
			}
		}
		for class := data.Entities.Classes[super]; class != nil; class = data.Entities.Classes[super] {
			if !class.Removed {
				eclass.Superclasses = append(eclass.Superclasses, class)
			}
			super = class.Element.Superclass
		}
		for _, sub := range data.Entities.Classes {
			if sub.Element.Superclass == id && !sub.Removed {
				eclass.Subclasses = append(eclass.Subclasses, sub)
			}
		}
		sort.Slice(eclass.Subclasses, func(i, j int) bool {
			return eclass.Subclasses[i].ID < eclass.Subclasses[j].ID
		})
	}
	sort.Slice(data.TreeRoots, func(i, j int) bool {
		return data.TreeRoots[i].ID < data.TreeRoots[j].ID
	})
}

func (data *Data) GenerateUpdates() {
	if len(data.Patches) == 0 {
		return
	}

	// Patches will be displayed latest-first.
	patches := make([]*Patch, len(data.Patches))
	for i := len(data.Patches) / 2; i >= 0; i-- {
		j := len(data.Patches) - 1 - i
		patches[i], patches[j] = &data.Patches[j], &data.Patches[i]
	}
	// Exclude earliest patch.
	patches = patches[:len(patches)-1]

	data.LatestYear = patches[0].Info.Date.Year()
	data.EarliestYear = patches[len(patches)-1].Info.Date.Year()
	data.PatchesByYear = make([]PatchSet, data.LatestYear-data.EarliestYear+1)
	years := make([]int, len(data.PatchesByYear))
	for i := range years {
		years[i] = data.LatestYear - i
	}

	// Populate PatchesByYear.
	i := 0
	current := data.LatestYear
	for j, patch := range patches {
		year := patch.Info.Date.Year()
		if year < current {
			if j > i {
				data.PatchesByYear[data.LatestYear-current] = PatchSet{
					Year:    current,
					Years:   years,
					Patches: patches[i:j],
				}
			}
			current = year
			i = j
		}
	}
	if len(patches) > i {
		data.PatchesByYear[data.LatestYear-current] = PatchSet{
			Year:    current,
			Years:   years,
			Patches: patches[i:],
		}
	}

	// Populate LatestPatches.
	i = len(patches)
	epoch := patches[0].Info.Date.AddDate(0, -3, 0)
	for j, patch := range patches {
		if patch.Info.Date.Before(epoch) {
			i = j - 1
			break
		}
	}
	data.LatestPatches = PatchSet{
		Years:   years,
		Patches: patches[:i],
	}
}

type BuildInfo struct {
	Hash    string
	Date    time.Time
	Version fetch.Version
}

func (a BuildInfo) Equal(b BuildInfo) bool {
	if a.Hash != b.Hash {
		return false
	}
	if a.Version != b.Version {
		return false
	}
	if !a.Date.Equal(b.Date) {
		return false
	}
	return true
}

func (m BuildInfo) String() string {
	return fmt.Sprintf("%s; %s; %s", m.Hash, m.Date, m.Version)
}

type ReflectionMetadata struct {
	Classes map[string]ClassMetadata
	Enums   map[string]EnumMetadata
}

type ItemMetadata struct {
	Name string
	// Browsable       bool
	// ClassCategory   string
	// Constraint      string
	// Deprecated      bool
	// EditingDisabled bool
	// IsBackend       bool
	// ScriptContext   string
	// UIMaximum       float64
	// UIMinimum       float64
	// UINumTicks      float64
	// Summary         string
}

type ClassMetadata struct {
	ItemMetadata
	ExplorerImageIndex int
	// ExplorerOrder      int
	// Insertable         bool
	// PreferredParent    string
	// PreferredParents   string
}

type EnumMetadata struct {
	ItemMetadata
}

func getMetadataValue(p interface{}, v rbxfile.Value) {
	switch p := p.(type) {
	case *int:
		switch v := v.(type) {
		case rbxfile.ValueInt:
			*p = int(v)
		case rbxfile.ValueString:
			*p, _ = strconv.Atoi(string(v))
		}
	}
}

func (data *Data) GenerateMetadata(rmd *rbxfile.Root) {
	data.Metadata.Classes = make(map[string]ClassMetadata)
	data.Metadata.Enums = make(map[string]EnumMetadata)
	for _, list := range rmd.Instances {
		switch list.ClassName {
		case "ReflectionMetadataClasses":
			for _, class := range list.Children {
				if class.ClassName != "ReflectionMetadataClass" {
					continue
				}
				meta := ClassMetadata{ItemMetadata: ItemMetadata{Name: class.Name()}}
				getMetadataValue(&meta.ExplorerImageIndex, class.Properties["ExplorerImageIndex"])
				data.Metadata.Classes[meta.Name] = meta
			}
		case "ReflectionMetadataEnums":
		}
	}
}
