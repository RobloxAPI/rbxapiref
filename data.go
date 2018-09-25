package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/patch"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"github.com/robloxapi/rbxapiref/fetch"
	"github.com/robloxapi/rbxfile"
	"html/template"
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
)

type Data struct {
	Settings Settings

	Patches  []Patch
	Latest   *Build
	Metadata ReflectionMetadata

	Entities *Entities
	Types    []rbxapijson.Type
	TypeList []TypeCategory

	Tree      map[string]*TreeNode
	TreeRoots []string

	Templates *template.Template
}

type TypeCategory struct {
	Name  string
	Types []rbxapijson.Type
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
			s = path.Join("updates", args[0]+FileExt)
		} else {
			s = "updates" + FileExt
		}
	case "class":
		s = path.Join(ClassPath, url.PathEscape(args[0])+FileExt)
	case "member":
		if len(args) == 1 {
			return (&url.URL{Fragment: MemberAnchorPrefix + args[0]}).String()
		} else if len(args) == 2 {
			s = path.Join(ClassPath, url.PathEscape(args[0])+FileExt) +
				(&url.URL{Fragment: MemberAnchorPrefix + args[1]}).String()
		}
	case "enum":
		s = path.Join(EnumPath, url.PathEscape(args[0])+FileExt)
	case "enumitem":
		if len(args) == 1 {
			return (&url.URL{Fragment: MemberAnchorPrefix + args[0]}).String()
		} else if len(args) == 2 {
			s = path.Join(EnumPath, url.PathEscape(args[0])+FileExt) +
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
		s = path.Join(TypePath, url.PathEscape(args[1])+FileExt)
	}
	s = path.Join("/", data.Settings.Output.Sub, s)
	return s
}

// FilePath generates an absolute path located in the Output. On a web server
// serving static files, the returned path is meant to point to the same file
// as the file pointed to by the URL generated by FileLink.
func (data *Data) FilePath(typ string, args ...string) string {
	return data.PathFromLink(data.FileLink(typ, args...))
}

// LinkFromPath transforms a path into a link, if possible.
func (data *Data) LinkFromPath(p string) string {
	if l, err := filepath.Rel(data.Settings.Output.Root, p); err == nil {
		return l
	}
	return p
}

// PathFrom link transforms a link into a path, if possible.
func (data *Data) PathFromLink(l string) string {
	return filepath.Join(data.Settings.Output.Root, l)
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
		const body = `<span class="%s" style="background-position: %dpx"></span>`
		style = fmt.Sprintf(` style="background-position: %dpx"`, -index*16)
	}
	const body = `<span class="%s" title="%s"%s></span>`
	return template.HTML(fmt.Sprintf(body, template.HTMLEscapeString(class), template.HTMLEscapeString(title), style))
}

func (data *Data) ExecuteTemplate(name string, tdata interface{}) (template.HTML, error) {
	var buf bytes.Buffer
	err := data.Templates.ExecuteTemplate(&buf, name, tdata)
	return template.HTML(buf.String()), err
}

func addType(types map[string]rbxapijson.Type, t rbxapijson.Type) {
	switch t.Category {
	case "Class", "Enum":
		return
	}
	if _, ok := types[t.Name]; ok {
		return
	}
	types[t.Name] = t
}

func (data *Data) GenerateEntities() {
	data.Entities = &Entities{
		Classes:   make(map[string]*ClassEntity),
		Members:   make(map[[2]string]*MemberEntity),
		Enums:     make(map[string]*EnumEntity),
		EnumItems: make(map[[2]string]*EnumItemEntity),
	}
	types := map[string]rbxapijson.Type{}

	for _, class := range data.Latest.API.Classes {
		if data.Entities.Classes[class.Name] != nil {
			continue
		}
		data.Entities.Classes[class.Name] = &ClassEntity{
			ID:      class.Name,
			Element: class,
		}
		for _, member := range class.Members {
			id := [2]string{class.Name, member.GetName()}
			if data.Entities.Members[id] != nil {
				continue
			}
			data.Entities.Members[id] = &MemberEntity{
				ID:      id,
				Element: member,
			}
			switch member.GetMemberType() {
			case "Property":
				member := member.(*rbxapijson.Property)
				addType(types, member.ValueType)
			case "Function":
				member := member.(*rbxapijson.Function)
				addType(types, member.ReturnType)
				for _, p := range member.Parameters {
					addType(types, p.Type)
				}
			case "Event":
				member := member.(*rbxapijson.Event)
				for _, p := range member.Parameters {
					addType(types, p.Type)
				}
			case "Callback":
				member := member.(*rbxapijson.Callback)
				addType(types, member.ReturnType)
				for _, p := range member.Parameters {
					addType(types, p.Type)
				}
			}
		}
	}
	for _, enum := range data.Latest.API.Enums {
		if data.Entities.Enums[enum.Name] != nil {
			continue
		}
		data.Entities.Enums[enum.Name] = &EnumEntity{
			ID:      enum.Name,
			Element: enum,
		}
		for _, item := range enum.Items {
			id := [2]string{enum.Name, item.Name}
			if data.Entities.EnumItems[id] != nil {
				continue
			}
			data.Entities.EnumItems[id] = &EnumItemEntity{
				ID:      id,
				Element: item,
			}
		}
	}

	data.Types = make([]rbxapijson.Type, 0, len(types))
	data.TypeList = []TypeCategory{}
loop:
	for _, t := range types {
		data.Types = append(data.Types, t)
		for i, cat := range data.TypeList {
			if cat.Name == t.Category {
				data.TypeList[i].Types = append(data.TypeList[i].Types, t)
				continue loop
			}
		}
		data.TypeList = append(data.TypeList, TypeCategory{
			Name:  t.Category,
			Types: []rbxapijson.Type{t},
		})
	}
	sort.Slice(data.Types, func(i, j int) bool {
		return data.Types[i].Name < data.Types[j].Name
	})
	sort.Slice(data.TypeList, func(i, j int) bool {
		return data.TypeList[i].Name < data.TypeList[j].Name
	})
	for _, cat := range data.TypeList {
		sort.Slice(cat.Types, func(i, j int) bool {
			return cat.Types[i].Name < cat.Types[j].Name
		})
	}
}

func (data *Data) GenerateTree() {
	data.Tree = make(map[string]*TreeNode, len(data.Entities.Classes))
	for id, class := range data.Entities.Classes {
		node := TreeNode{}
		super := class.Element.Superclass
		if data.Entities.Classes[super] == nil {
			data.TreeRoots = append(data.TreeRoots, id)
		}
		for class := data.Entities.Classes[super]; class != nil; class = data.Entities.Classes[super] {
			node.Super = append(node.Super, super)
			super = class.Element.Superclass
		}
		for subid, sub := range data.Entities.Classes {
			if sub.Element.Superclass == id {
				node.Sub = append(node.Sub, subid)
			}
		}
		sort.Strings(node.Sub)
		data.Tree[id] = &node
	}
	sort.Strings(data.TreeRoots)
}

type TreeNode struct {
	Super []string
	Sub   []string
}

type Settings struct {
	// Input specifies input settings.
	Input SettingsInput
	// Output specifies output settings.
	Output SettingsOutput
	// Configs maps an identifying name to a fetch configuration.
	Configs map[string]fetch.Config
	// UseConfigs specifies the logical concatenation of the fetch configs
	// defined in the Configs setting. Builds from these configs are read
	// sequentially.
	UseConfigs []string
}

type SettingsInput struct {
	// Resources is the location of resource files.
	Resources string
	// Templates is the location of template files.
	Templates string
}

type SettingsOutput struct {
	// Root is the directory to which generated files will be written.
	Root string
	// Sub is a path that follows the output directory and precedes a
	// generated file path.
	Sub string
	// Resources is the path relative to the Base where generated resource
	// files will be written.
	Resources string
	// Manifest is the path relative to the base that points to the manifest
	// file.
	Manifest string
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

type Entities struct {
	Classes   map[string]*ClassEntity
	Members   map[[2]string]*MemberEntity
	Enums     map[string]*EnumEntity
	EnumItems map[[2]string]*EnumItemEntity
}

type ClassEntity struct {
	ID      string
	Element *rbxapijson.Class
	Patches []Patch
	Removed bool
}

type MemberEntity struct {
	ID      [2]string
	Element rbxapi.Member
	Patches []Patch
	Removed bool
}

type EnumEntity struct {
	ID      string
	Element *rbxapijson.Enum
	Patches []Patch
	Removed bool
}

type EnumItemEntity struct {
	ID      [2]string
	Element *rbxapijson.EnumItem
	Patches []Patch
	Removed bool
}

type Action struct {
	Type     patch.Type
	Class    *rbxapijson.Class    `json:",omitempty"`
	Property *rbxapijson.Property `json:",omitempty"`
	Function *rbxapijson.Function `json:",omitempty"`
	Event    *rbxapijson.Event    `json:",omitempty"`
	Callback *rbxapijson.Callback `json:",omitempty"`
	Enum     *rbxapijson.Enum     `json:",omitempty"`
	EnumItem *rbxapijson.EnumItem `json:",omitempty"`
	Field    string               `json:",omitempty"`
	Prev     *Value               `json:",omitempty"`
	Next     *Value               `json:",omitempty"`
}

func WrapActions(actions []patch.Action) []Action {
	c := make([]Action, len(actions))
	for i, action := range actions {
		c[i] = Action{
			Type:  action.GetType(),
			Field: action.GetField(),
		}
		if p := action.GetPrev(); p != nil {
			c[i].Prev = WrapValue(p)
		}
		if n := action.GetNext(); n != nil {
			c[i].Next = WrapValue(n)
		}
		switch action := action.(type) {
		case patch.Member:
			class := action.GetClass().(*rbxapijson.Class)
			members := class.Members
			class.Members = nil
			c[i].Class = class.Copy().(*rbxapijson.Class)
			class.Members = members

			c[i].SetMember(action.GetMember().Copy())
		case patch.Class:
			if action.GetType() == patch.Change {
				class := action.GetClass().(*rbxapijson.Class)
				members := class.Members
				class.Members = nil
				c[i].Class = class.Copy().(*rbxapijson.Class)
				class.Members = members
			} else {
				c[i].Class = action.GetClass().Copy().(*rbxapijson.Class)
			}
		case patch.EnumItem:
			enum := action.GetEnum().(*rbxapijson.Enum)
			items := enum.Items
			enum.Items = nil
			c[i].Enum = enum.Copy().(*rbxapijson.Enum)
			enum.Items = items

			c[i].EnumItem = action.GetEnumItem().Copy().(*rbxapijson.EnumItem)
		case patch.Enum:
			if action.GetType() == patch.Change {
				enum := action.GetEnum().(*rbxapijson.Enum)
				items := enum.Items
				enum.Items = nil
				c[i].Enum = enum.Copy().(*rbxapijson.Enum)
				enum.Items = items

			} else {
				c[i].Enum = action.GetEnum().Copy().(*rbxapijson.Enum)
			}
		}
	}
	return c
}
func (a *Action) GetClass() rbxapi.Class { return a.Class }
func (a *Action) GetMember() rbxapi.Member {
	switch {
	case a.Property != nil:
		return a.Property
	case a.Function != nil:
		return a.Function
	case a.Event != nil:
		return a.Event
	case a.Callback != nil:
		return a.Callback
	}
	return nil
}
func (a *Action) SetMember(member rbxapi.Member) {
	switch member := member.(type) {
	case *rbxapijson.Property:
		a.Property = member
		a.Function = nil
		a.Event = nil
		a.Callback = nil
	case *rbxapijson.Function:
		a.Property = nil
		a.Function = member
		a.Event = nil
		a.Callback = nil
	case *rbxapijson.Event:
		a.Property = nil
		a.Function = nil
		a.Event = member
		a.Callback = nil
	case *rbxapijson.Callback:
		a.Property = nil
		a.Function = nil
		a.Event = nil
		a.Callback = member
	}
}
func (a *Action) GetEnum() rbxapi.Enum         { return a.Enum }
func (a *Action) GetEnumItem() rbxapi.EnumItem { return a.EnumItem }
func (a *Action) GetType() patch.Type          { return a.Type }
func (a *Action) GetField() string             { return a.Field }
func (a *Action) GetPrev() interface{} {
	if a.Prev != nil {
		return a.Prev.V
	}
	return nil
}
func (a *Action) GetNext() interface{} {
	if a.Next != nil {
		return a.Next.V
	}
	return nil
}
func (a *Action) String() string { return "Action" }

type Value struct {
	V interface{}
}

func WrapValue(v interface{}) *Value {
	w := Value{}
	switch v := v.(type) {
	case rbxapi.Type:
		w.V = rbxapijson.Type{
			Category: v.GetCategory(),
			Name:     v.GetName(),
		}
	case []rbxapi.Parameter:
		params := make([]rbxapijson.Parameter, len(v))
		for i, p := range v {
			params[i] = rbxapijson.Parameter{
				Type: rbxapijson.Type{
					Category: p.GetType().GetCategory(),
					Name:     p.GetType().GetName(),
				},
				Name: p.GetName(),
			}
			if d, ok := p.GetDefault(); ok {
				params[i].Default = &d
			}
		}
		w.V = params
	default:
		w.V = v
	}
	return &w
}

func (v *Value) MarshalJSON() (b []byte, err error) {
	var w struct {
		Type  string
		Value interface{}
	}
	switch v := v.V.(type) {
	case bool:
		w.Type = "bool"
		w.Value = v
	case int:
		w.Type = "int"
		w.Value = v
	case string:
		w.Type = "string"
		w.Value = v
	case rbxapijson.Type:
		w.Type = "Type"
		w.Value = v
	case []string:
		w.Type = "strings"
		w.Value = v
	case []rbxapijson.Parameter:
		w.Type = "Parameters"
		w.Value = v
	}
	return json.Marshal(&w)
}

func (v *Value) UnmarshalJSON(b []byte) (err error) {
	var w struct{ Type string }
	if err = json.Unmarshal(b, &w); err != nil {
		return err
	}
	switch w.Type {
	case "bool":
		var value struct{ Value bool }
		if err = json.Unmarshal(b, &value); err != nil {
			return err
		}
		v.V = value.Value
	case "int":
		var value struct{ Value int }
		if err = json.Unmarshal(b, &value); err != nil {
			return err
		}
		v.V = value.Value
	case "string":
		var value struct{ Value string }
		if err = json.Unmarshal(b, &value); err != nil {
			return err
		}
		v.V = value.Value
	case "Type":
		var value struct{ Value rbxapijson.Type }
		if err = json.Unmarshal(b, &value); err != nil {
			return err
		}
		v.V = value.Value
	case "strings":
		var value struct{ Value []string }
		if err = json.Unmarshal(b, &value); err != nil {
			return err
		}
		v.V = value.Value
	case "Parameters":
		var value struct{ Value []rbxapijson.Parameter }
		if err = json.Unmarshal(b, &value); err != nil {
			return err
		}
		v.V = value.Value
	}
	return nil
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
