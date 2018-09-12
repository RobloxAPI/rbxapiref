package main

import (
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/patch"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"github.com/robloxapi/rbxapi/rbxapijson/diff"
	"github.com/robloxapi/rbxapiref/fetch"
	"html/template"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type API struct {
	Entities  *Entities
	Patches   []Patch
	Latest    *Build
	Templates *template.Template
}

type Metadata struct {
	Hash    string
	Date    time.Time
	Version fetch.Version
}

func (a Metadata) Equal(b Metadata) bool {
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

func (m Metadata) String() string {
	return fmt.Sprintf("%s; %s; %s", m.Hash, m.Date, m.Version)
}

type Build struct {
	Config   string
	Metadata Metadata
	API      *rbxapijson.Root
}

type Patch struct {
	Prev     *Metadata `json:",omitempty"`
	Metadata Metadata
	Config   string
	Actions  []Action
}

type Action struct {
	Type     patch.Type
	Class    *rbxapijson.Class    `json:",omitempty"`
	Property *rbxapijson.Property `json:",omitempty"`
	Function *rbxapijson.Function `json:",omitempty"`
	Event    *rbxapijson.Event    `json:",omitempty"`
	Callback *rbxapijson.Callback `json:",omitempty"`
	Enum     *rbxapijson.Enum     `json:",omitempty"`
	Item     *rbxapijson.EnumItem `json:",omitempty"`
	Field    string               `json:",omitempty"`
	Prev     *Value               `json:",omitempty"`
	Next     *Value               `json:",omitempty"`
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

type Value struct {
	V interface{}
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

func WrapActions(actions []patch.Action) []Action {
	c := make([]Action, len(actions))
	for i, action := range actions {
		c[i] = Action{
			Type:  action.GetType(),
			Field: action.GetField(),
		}
		if p := action.GetPrev(); p != nil {
			c[i].Prev = &Value{p}
		}
		if n := action.GetNext(); n != nil {
			c[i].Next = &Value{n}
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

			c[i].Item = action.GetItem().Copy().(*rbxapijson.EnumItem)
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
func (a *Action) GetEnum() rbxapi.Enum     { return a.Enum }
func (a *Action) GetItem() rbxapi.EnumItem { return a.Item }
func (a *Action) GetType() patch.Type      { return a.Type }
func (a *Action) GetField() string         { return a.Field }
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

type ClassEntity struct {
	Latest  *rbxapijson.Class
	Patches []Patch
}

type MemberEntity struct {
	Latest  rbxapi.Member
	Patches []Patch
}

type EnumEntity struct {
	Latest  *rbxapijson.Enum
	Patches []Patch
}

type EnumItemEntity struct {
	Latest  *rbxapijson.EnumItem
	Patches []Patch
}

type TypeEntity struct {
	Type rbxapijson.Type
}

type Entities struct {
	Classes   map[string]*ClassEntity
	Members   map[[2]string]*MemberEntity
	Enums     map[string]*EnumEntity
	EnumItems map[[2]string]*EnumItemEntity
	Types     map[string]*TypeEntity
}

const (
	RootPath            = "ref"
	ClassPath           = "class"
	EnumPath            = "enum"
	TypePath            = "type"
	FileExt             = ".html"
	MemberAnchorPrefix  = "member-"
	SectionAnchorPrefix = "section-"
)

func generateLink(typ string, iname, imember interface{}) (s string) {
	name := toString(iname)
	member := toString(imember)
retry:
	for {
		switch strings.ToLower(typ) {
		case "index":
			s = "index" + FileExt
		case "res":
			s = path.Join("res", name)
		case "updates":
			if name == "" {
				s = "updates" + FileExt
			} else {
				s = path.Join("updates", name+FileExt)
			}
		case "class":
			s = path.Join(ClassPath, url.PathEscape(name)+FileExt)
		case "member":
			s = path.Join(ClassPath, url.PathEscape(name)+FileExt) + (&url.URL{Fragment: MemberAnchorPrefix + member}).String()
		case "enum":
			s = path.Join(EnumPath, url.PathEscape(name)+FileExt)
		case "enumitem":
			s = path.Join(EnumPath, url.PathEscape(name)+FileExt) + (&url.URL{Fragment: MemberAnchorPrefix + member}).String()
		case "type":
			switch strings.ToLower(name) {
			case "class", "enum":
				typ, name, member = name, member, ""
				goto retry
			}
			s = path.Join(TypePath, url.PathEscape(member)+FileExt)
		}
		break
	}
	s = path.Join("/", RootPath, s)
	return s
}

func toString(v interface{}) string {
	switch v := v.(type) {
	case Value:
		return toString(v.V)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(v)
	case string:
		return v
	case rbxapijson.Type:
		return v.String()
	case []string:
		return "[" + strings.Join(v, ", ") + "]"
	case []rbxapijson.Parameter:
		ss := make([]string, len(v))
		for i, param := range v {
			ss[i] = param.Type.String() + " " + param.Name
			if param.Default != nil {
				ss[i] += " = " + *param.Default
			}
		}
		return "(" + strings.Join(ss, ", ") + ")"
	case []rbxapi.Parameter:
		ss := make([]string, len(v))
		for i, param := range v {
			ss[i] = param.GetType().String() + " " + param.GetName()
			if d, ok := param.GetDefault(); ok {
				ss[i] += " = " + d
			}
		}
		return "(" + strings.Join(ss, ", ") + ")"
	}
	return "<unknown value>"
}

func makeSubactions(action Action) []Action {
	if class := action.Class; class != nil {
		actions := make([]Action, len(class.Members))
		for i, member := range class.Members {
			actions[i] = Action{
				Type:  action.GetType(),
				Class: class,
			}
			actions[i].SetMember(member)
		}
		return actions
	} else if enum := action.Enum; enum != nil {
		actions := make([]Action, len(enum.Items))
		for i, item := range enum.Items {
			actions[i] = Action{
				Type: action.GetType(),
				Enum: enum,
				Item: item,
			}
		}
		return actions
	}
	return nil
}

func makeTemplates(dir string, funcs template.FuncMap) (tmpl *template.Template, err error) {
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	tmpl = template.New("")
	tmpl.Funcs(funcs)
	for _, fi := range fis {
		base := filepath.Base(fi.Name())
		name := base[:len(base)-len(filepath.Ext(base))]
		b, err := ioutil.ReadFile(filepath.Join(dir, fi.Name()))
		if err != nil {
			return nil, err
		}
		t := tmpl.New(name)
		if _, err = t.Parse(string(b)); err != nil {
			return nil, err
		}
		t.Funcs(funcs)
	}
	return
}

const BaseDir = RootPath

var pages = []func(api *API) error{
	func(api *API) error {
		f, err := os.Create(filepath.Join(BaseDir, "index.html"))
		if err != nil {
			return err
		}
		err = api.Templates.ExecuteTemplate(f, "index", api.Latest.Metadata.Hash)
		f.Close()
		return err
	},
	func(api *API) error {
		type args struct {
			Patches []Patch
			Year    int
			Years   []int
		}

		src := api.Patches
		if len(src) == 0 {
			f, err := os.Create(filepath.Join(BaseDir, "updates.html"))
			if err != nil {
				return err
			}
			err = api.Templates.ExecuteTemplate(f, "updates", args{})
			f.Close()
			return err
		}
		src = src[1:]
		patches := make([]Patch, len(src))
		for i := len(src) / 2; i >= 0; i-- {
			j := len(src) - 1 - i
			patches[i], patches[j] = src[j], src[i]
		}

		maxYear := patches[0].Metadata.Date.Year()
		minYear := patches[len(patches)-1].Metadata.Date.Year()
		patchesByYear := make([][]Patch, maxYear-minYear+1)
		years := make([]int, maxYear-minYear+1)
		for i := range years {
			years[i] = maxYear - i
		}
		{
			i := 0
			current := maxYear
			for j, patch := range patches {
				year := patch.Metadata.Date.Year()
				if year < current {
					if j > i {
						patchesByYear[maxYear-current] = patches[i:j]
					}
					current = year
					i = j
				}
			}
			if len(patches) > i {
				patchesByYear[maxYear-current] = patches[i:]
			}
		}
		{
			i := len(patches)
			epoch := patches[0].Metadata.Date.AddDate(0, -3, 0)
			for j, patch := range patches {
				if patch.Metadata.Date.Before(epoch) {
					i = j - 1
					break
				}
			}
			f, err := os.Create(filepath.Join(BaseDir, "updates.html"))
			if err != nil {
				return err
			}
			err = api.Templates.ExecuteTemplate(f, "updates", args{patches[:i], 0, years})
			f.Close()
			if err != nil {
				return err
			}
		}
		if err := os.MkdirAll(filepath.Join(BaseDir, "updates"), 0666); err != nil {
			return err
		}
		for i, patches := range patchesByYear {
			year := maxYear - i
			f, err := os.Create(filepath.Join(BaseDir, "updates", strconv.Itoa(year)+".html"))
			if err != nil {
				return err
			}
			err = api.Templates.ExecuteTemplate(f, "updates", args{patches, year, years})
			f.Close()
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func main() {
	spew.Config.DisableMethods = true
	spew.Config.DisablePointerMethods = true
	spew.Config.DisablePointerAddresses = true
	spew.Config.Indent = "\t"

	settings := make(map[string]fetch.Config)
	if f, err := os.Open("settings.json"); err == nil {
		err := json.NewDecoder(f).Decode(&settings)
		f.Close()
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	api := &API{
		Entities: &Entities{
			Classes:   make(map[string]*ClassEntity),
			Members:   make(map[[2]string]*MemberEntity),
			Enums:     make(map[string]*EnumEntity),
			EnumItems: make(map[[2]string]*EnumItemEntity),
			Types:     make(map[string]*TypeEntity),
		},
	}

	configs := []string{
		"LocalArchive",
		"Production",
	}
	client := &fetch.Client{}
	prevPatches := []Patch{}
	{
		f, err := os.Open(filepath.Join(BaseDir, "patches.json"))
		if err == nil {
			err = json.NewDecoder(f).Decode(&prevPatches)
			f.Close()
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}
	// fmt.Println("== PATCHES ================================")
	// spew.Dump(prevPatches)

	client.CacheMode = fetch.CacheNone
	builds := []Build{}
	for _, cfg := range configs {
		client.Config = settings[cfg]
		bs, err := client.Builds()
		if err != nil {
			fmt.Println(cfg, "error fetching builds:", err)
			return
		}
		for _, b := range bs {
			builds = append(builds, Build{Config: cfg, Metadata: Metadata(b)})
		}
	}
	client.CacheMode = fetch.CacheTemp
	// fmt.Println("== BUILDS ================================")
	// spew.Dump(builds)

loop:
	for _, build := range builds {
		for _, patch := range prevPatches {
			if !build.Metadata.Equal(patch.Metadata) {
				// Not relevant; skip.
				continue
			}
			// Current build has a cached version.
			if api.Latest == nil {
				if patch.Prev != nil {
					// Cached build is now the first, but was not originally;
					// actions are stale.
					fmt.Println("== STALE ", patch.Metadata)
					break
				}
			} else {
				if patch.Prev == nil {
					// Cached build was not originally the first, but now is;
					// actions are stale.
					fmt.Println("== STALE ", patch.Metadata)
					break
				}
				if !api.Latest.Metadata.Equal(*patch.Prev) {
					// Latest build does not match previous build; actions are
					// stale.
					fmt.Println("== STALE ", patch.Metadata)
					break
				}
			}
			// Cached actions are still fresh; set them directly.
			api.Patches = append(api.Patches, patch)
			api.Latest = &Build{Metadata: patch.Metadata, Config: patch.Config}
			fmt.Println("== CACHED ", patch.Metadata)
			continue loop
		}
		fmt.Println("== NEW ", build.Metadata)
		client.Config = settings[build.Config]
		root, err := client.APIDump(build.Metadata.Hash)
		if err != nil {
			fmt.Println(build.Config, "failed to get build ", build.Metadata.Hash, err)
			continue
		}
		build.API = root
		var actions []Action
		if api.Latest == nil {
			// First build; compare with nothing.
			actions = WrapActions((&diff.Diff{Prev: nil, Next: build.API}).Diff())
		} else {
			if api.Latest.API == nil {
				// Previous build was cached; fetch its data to compare with
				// current build.
				client.Config = settings[api.Latest.Config]
				root, err := client.APIDump(api.Latest.Metadata.Hash)
				if err != nil {
					fmt.Println(api.Latest.Config, "failed to get build ", api.Latest.Metadata.Hash, err)
					continue
				}
				api.Latest.API = root
			}
			actions = WrapActions((&diff.Diff{Prev: api.Latest.API, Next: build.API}).Diff())
		}
		patch := Patch{Metadata: build.Metadata, Config: build.Config, Actions: actions}
		if api.Latest != nil {
			prev := api.Latest.Metadata
			patch.Prev = &prev
		}
		api.Patches = append(api.Patches, patch)
		b := build
		api.Latest = &b
	}
	// Ensure that the latest API is present.
	if api.Latest.API == nil {
		client.Config = settings[api.Latest.Config]
		root, err := client.APIDump(api.Latest.Metadata.Hash)
		if err != nil {
			fmt.Println(api.Latest.Config, "failed to get build ", api.Latest.Metadata.Hash, err)
			return
		}
		api.Latest.API = root
	}

	var err error
	api.Templates, err = makeTemplates("templates", template.FuncMap{
		"link":       generateLink,
		"tostring":   toString,
		"subactions": makeSubactions,
		"type": func(v interface{}) string {
			return reflect.TypeOf(v).String()
		},
	})
	if err != nil {
		fmt.Println("failed to open template", err)
		return
	}

	for _, page := range pages {
		if err := page(api); err != nil {
			fmt.Println(err)
			return
		}
	}

	{
		f, err := os.Create(filepath.Join(BaseDir, "patches.json"))
		if err != nil {
			fmt.Println(err)
			return
		}
		je := json.NewEncoder(f)
		je.SetEscapeHTML(false)
		je.SetIndent("", "\t")
		err = je.Encode(api.Patches)
		f.Close()
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}
