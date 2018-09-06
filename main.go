package main

import (
	"encoding/json"
	"fmt"
	"github.com/robloxapi/rbxapi"
	rbxapidiff "github.com/robloxapi/rbxapi/diff"
	"github.com/robloxapi/rbxapi/patch"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"github.com/robloxapi/rbxapi/rbxapijson/diff"
	"github.com/robloxapi/rbxapiref/fetch"
	"html/template"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

type API struct {
	Entities  *Entities
	Patches   []Patch
	Latest    *Build
	Templates *template.Template
}

type Metadata = fetch.Build

type Build struct {
	Metadata Metadata
	API      *rbxapijson.Root
}

type Patch struct {
	Metadata Metadata
	Actions  []Action
}

type Action struct {
	patch.Action
}

func WrapActions(actions []patch.Action) []Action {
	c := make([]Action, len(actions))
	for i, action := range actions {
		c[i] = Action{Action: action}
	}
	return c
}

func (a *Action) GetClass() rbxapi.Class {
	if a, ok := a.Action.(patch.Class); ok {
		return a.GetClass()
	}
	return nil
}

func (a *Action) GetMember() rbxapi.Member {
	if a, ok := a.Action.(patch.Member); ok {
		return a.GetMember()
	}
	return nil
}

func (a *Action) GetEnum() rbxapi.Enum {
	if a, ok := a.Action.(patch.Enum); ok {
		return a.GetEnum()
	}
	return nil
}

func (a *Action) GetItem() rbxapi.EnumItem {
	if a, ok := a.Action.(patch.EnumItem); ok {
		return a.GetItem()
	}
	return nil
}

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

func generateLink(typ, name, member string) (s string) {
retry:
	for {
		switch strings.ToLower(typ) {
		case "index":
			s = "index" + FileExt
		case "updates":
			s = "updates" + FileExt
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
			s = path.Join(TypePath, url.PathEscape(name)+FileExt)
		}
		break
	}
	s = path.Join("/", RootPath, s)
	return s
}

func toString(v interface{}) string {
	switch v := v.(type) {
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(v)
	case string:
		return v
	case rbxapi.Type:
		return v.String()
	case []string:
		return "[" + strings.Join(v, ", ") + "]"
	case []rbxapi.Parameter:
		ss := make([]string, len(v))
		for i, param := range v {
			ss[i] = param.GetType().String() + " " + param.GetName()
			if def, ok := param.GetDefault(); ok {
				ss[i] += " = " + def
			}
		}
		return "(" + strings.Join(ss, ", ") + ")"
	}
	return "<unknown value>"
}

func makeSubactions(action Action) []Action {
	if class, _ := action.GetClass().(*rbxapijson.Class); class != nil {
		actions := make([]Action, len(class.Members))
		for i, member := range class.Members {
			actions[i] = Action{Action: &rbxapidiff.MemberAction{
				Type:   action.GetType(),
				Class:  class,
				Member: member,
			}}
		}
		return actions
	} else if enum, _ := action.GetEnum().(*rbxapijson.Enum); enum != nil {
		actions := make([]Action, len(enum.Items))
		for i, item := range enum.Items {
			actions[i] = Action{Action: &rbxapidiff.EnumItemAction{
				Type: action.GetType(),
				Enum: enum,
				Item: item,
			}}
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

type Page struct {
	Filename string
	Func     func(w io.Writer, api *API) error
}

const baseDir = RootPath

var pages = []Page{
	{"updates.html", func(w io.Writer, api *API) error {
		src := api.Patches[1:]
		patches := make([]Patch, len(src))
		for i := len(src)/2 - 1; i >= 0; i-- {
			j := len(src) - 1 - i
			patches[i], patches[j] = src[j], src[i]
		}
		if err := api.Templates.ExecuteTemplate(w, "updates", patches); err != nil {
			return err
		}
		return nil
	}},
}

func main() {
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

	client := &fetch.Client{CacheMode: fetch.CacheTemp}
	for c, cfg := range []fetch.Config{settings["LocalArchive"], settings["Production"]} {
		client.Config = cfg
		builds, err := client.Builds()
		if err != nil {
			fmt.Println(c, "error fetching builds:", err)
			return
		}
		if len(builds) < 1 {
			fmt.Println(c, "not enough builds")
			return
		}
		for i := 0; i < len(builds); i++ {
			root, err := client.APIDump(builds[i].Hash)
			if err != nil {
				fmt.Println(c, "failed to get build ", builds[i].Hash, err)
				continue
			}
			next := &Build{Metadata: builds[i], API: root}
			var actions []Action
			if api.Latest == nil {
				actions = WrapActions((&diff.Diff{Prev: nil, Next: next.API}).Diff())
			} else {
				actions = WrapActions((&diff.Diff{Prev: api.Latest.API, Next: next.API}).Diff())
			}

			api.Patches = append(api.Patches, Patch{Metadata: builds[i], Actions: actions})
			api.Latest = next
		}
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
		f, err := os.Create(filepath.Join(baseDir, page.Filename))
		if err != nil {
			fmt.Println(err)
			continue
		}
		err = page.Func(f, api)
		f.Close()
		if err != nil {
			fmt.Println(err)
			continue
		}
	}
}
