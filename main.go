package main

import (
	"encoding/json"
	"fmt"
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/links"
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
	"sort"
	"strconv"
	"strings"
)

type API struct {
	Entities  *Entities
	Latest    *Build
	Templates *template.Template
}

type Action struct {
	patch.Action
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

type Patch struct {
	// Build indicates which build caused the patch.
	Build   fetch.Build
	Actions []Action
}

type ClassEntity struct {
	Base, Latest rbxapi.Class
	Patches      []Patch
	Members      []*MemberEntity
}

type MemberEntity struct {
	Base, Latest rbxapi.Member
	Patches      []Patch
}

type EnumEntity struct {
	Base, Latest rbxapi.Enum
	Patches      []Patch
	Items        []*EnumItemEntity
}

type EnumItemEntity struct {
	Base, Latest rbxapi.EnumItem
	Patches      []Patch
}

type Entities struct {
	Classes []*ClassEntity
	Enums   []*EnumEntity
	Latest  map[interface{}]interface{}
	Links   *links.Links
}

type Build struct {
	Metadata fetch.Build
	API      *rbxapijson.Root
}

func wrapActions(actions []patch.Action) []Action {
	c := make([]Action, len(actions))
	for i, action := range actions {
		c[i] = Action{Action: action}
	}
	return c
}

func memberActions(class *rbxapijson.Class, prev, next rbxapi.Member) []Action {
	var actions []patch.Action
	if prev != nil && next != nil {
		switch prev.GetMemberType() {
		case "Property":
			actions = (&diff.DiffProperty{
				Class: class,
				Prev:  prev.(*rbxapijson.Property),
				Next:  next.(*rbxapijson.Property),
			}).Diff()
		case "Function":
			actions = (&diff.DiffFunction{
				Class: class,
				Prev:  prev.(*rbxapijson.Function),
				Next:  next.(*rbxapijson.Function),
			}).Diff()
		case "Event":
			actions = (&diff.DiffEvent{
				Class: class,
				Prev:  prev.(*rbxapijson.Event),
				Next:  next.(*rbxapijson.Event),
			}).Diff()
		case "Callback":
			actions = (&diff.DiffCallback{
				Class: class,
				Prev:  prev.(*rbxapijson.Callback),
				Next:  next.(*rbxapijson.Callback),
			}).Diff()
		}
	} else if prev != nil {
		switch prev.GetMemberType() {
		case "Property":
			actions = (&diff.DiffProperty{
				Class: class,
				Prev:  prev.(*rbxapijson.Property),
			}).Diff()
		case "Function":
			actions = (&diff.DiffFunction{
				Class: class,
				Prev:  prev.(*rbxapijson.Function),
			}).Diff()
		case "Event":
			actions = (&diff.DiffEvent{
				Class: class,
				Prev:  prev.(*rbxapijson.Event),
			}).Diff()
		case "Callback":
			actions = (&diff.DiffCallback{
				Class: class,
				Prev:  prev.(*rbxapijson.Callback),
			}).Diff()
		}
	} else if next != nil {
		switch next.GetMemberType() {
		case "Property":
			actions = (&diff.DiffProperty{
				Class: class,
				Next:  next.(*rbxapijson.Property),
			}).Diff()
		case "Function":
			actions = (&diff.DiffFunction{
				Class: class,
				Next:  next.(*rbxapijson.Function),
			}).Diff()
		case "Event":
			actions = (&diff.DiffEvent{
				Class: class,
				Next:  next.(*rbxapijson.Event),
			}).Diff()
		case "Callback":
			actions = (&diff.DiffCallback{
				Class: class,
				Next:  next.(*rbxapijson.Callback),
			}).Diff()
		}
	}
	return wrapActions(actions)
}

func (es *Entities) compareClasses(prev, next *Build) {
	if prev != nil {
		// Compare existing classes.
		for _, pclass := range prev.API.Classes {
			nclass, _ := es.Links.Next(pclass).(*rbxapijson.Class)
			actions := wrapActions((&diff.DiffClass{
				Prev:           pclass,
				Next:           nclass,
				ExcludeMembers: true,
			}).Diff())
			switch {
			case pclass != nil && nclass != nil:
				// Entity persists.
				eclass, _ := es.Latest[pclass].(*ClassEntity)
				if eclass == nil {
					panic(fmt.Errorf("no entity of class %s", pclass.Name))
				}

				// Point the latest item to the entity.
				delete(es.Latest, pclass)
				es.Latest[nclass] = eclass
				eclass.Latest = nclass

				// Include changes to entity.
				if len(actions) > 0 {
					eclass.Patches = append(eclass.Patches, Patch{
						Build:   next.Metadata,
						Actions: actions,
					})
				}

				// Compare existing members.
				for _, pmember := range pclass.Members {
					nmember, _ := es.Links.Next(pmember).(rbxapi.Member)
					actions := memberActions(pclass, pmember, nmember)
					switch {
					case pmember != nil && nmember != nil:
						// Member persists.
						emember, _ := es.Latest[pmember].(*MemberEntity)
						if emember == nil {
							panic(fmt.Errorf("no entity of member %s.%s", pclass.Name, pmember.GetName()))
						}

						// Point the latest item to the entity.
						delete(es.Latest, pmember)
						es.Latest[nmember] = emember
						emember.Latest = nmember

						if len(actions) > 0 {
							// Include changes to entity.
							emember.Patches = append(emember.Patches, Patch{
								Build:   next.Metadata,
								Actions: actions,
							})
						}
					case pmember != nil:
						// Member was removed.
						emember, _ := es.Latest[pmember].(*MemberEntity)
						if emember == nil {
							panic(fmt.Errorf("no entity of member %s.%s", pclass.Name, pmember.GetName()))
						}
						emember.Patches = append(emember.Patches, Patch{
							Build:   next.Metadata,
							Actions: actions,
						})
					}
				}
				// Check for new members.
				for _, nmember := range nclass.Members {
					if pmember, _ := es.Links.Prev(nmember).(rbxapi.Member); pmember != nil {
						// Only count members that did not previously exist.
						continue
					}
					emember := &MemberEntity{
						Base:   nmember,
						Latest: nmember,
						Patches: []Patch{{
							Build:   next.Metadata,
							Actions: memberActions(nclass, nil, nmember),
						}},
					}
					eclass.Members = append(eclass.Members, emember)
					es.Latest[nmember] = emember
				}
			case pclass != nil:
				// Entity was removed.
				eclass, _ := es.Latest[pclass].(*ClassEntity)
				if eclass == nil {
					panic(fmt.Errorf("no entity of class %s", pclass.Name))
				}
				eclass.Patches = append(eclass.Patches, Patch{
					Build:   next.Metadata,
					Actions: actions,
				})

				// Also remove members.
				for _, pmember := range pclass.Members {
					emember, _ := es.Latest[pmember].(*MemberEntity)
					if emember == nil {
						panic(fmt.Errorf("no entity of member %s.%s", pclass.Name, pmember.GetName()))
					}
					emember.Patches = append(emember.Patches, Patch{
						Build:   next.Metadata,
						Actions: memberActions(pclass, pmember, nil),
					})
				}
			}
		}
	}
	// Check for new entities.
	for _, nclass := range next.API.Classes {
		if pclass, _ := es.Links.Prev(nclass).(*rbxapijson.Class); pclass != nil {
			// Only count entities that did not previously exist.
			continue
		}
		eclass := &ClassEntity{
			Base:   nclass,
			Latest: nclass,
			Patches: []Patch{{
				Build: next.Metadata,
				Actions: wrapActions((&diff.DiffClass{
					Prev:           nil,
					Next:           nclass,
					ExcludeMembers: true,
				}).Diff()),
			}},
		}
		es.Classes = append(es.Classes, eclass)
		es.Latest[nclass] = eclass

		// Also add members.
		for _, nmember := range nclass.Members {
			emember := &MemberEntity{
				Base: nmember, Latest: nmember,
				Patches: []Patch{{
					Build:   next.Metadata,
					Actions: memberActions(nclass, nil, nmember),
				}},
			}
			eclass.Members = append(eclass.Members, emember)
			es.Latest[nmember] = emember
		}
	}
}

func (es *Entities) compareEnums(prev, next *Build) {
	if prev != nil {
		for _, penum := range prev.API.Enums {
			nenum, _ := es.Links.Next(penum).(*rbxapijson.Enum)
			actions := wrapActions((&diff.DiffEnum{
				Prev:         penum,
				Next:         nenum,
				ExcludeItems: true,
			}).Diff())
			switch {
			case penum != nil && nenum != nil:
				eenum, _ := es.Latest[penum].(*EnumEntity)
				if eenum == nil {
					panic(fmt.Errorf("no entity of enum %s", penum.Name))
				}

				delete(es.Latest, penum)
				es.Latest[nenum] = eenum
				eenum.Latest = nenum

				if len(actions) > 0 {
					eenum.Patches = append(eenum.Patches, Patch{
						Build:   next.Metadata,
						Actions: actions,
					})
				}

				for _, pitem := range penum.Items {
					nitem, _ := es.Links.Next(pitem).(*rbxapijson.EnumItem)
					actions := wrapActions((&diff.DiffEnumItem{
						Enum: penum,
						Prev: pitem,
						Next: nitem,
					}).Diff())
					switch {
					case pitem != nil && nitem != nil:
						eitem, _ := es.Latest[pitem].(*EnumItemEntity)
						if eitem == nil {
							panic(fmt.Errorf("no entity of enum item %s.%s", penum.Name, pitem.Name))
						}

						delete(es.Latest, pitem)
						es.Latest[nitem] = eitem
						eitem.Latest = nitem

						if len(actions) > 0 {
							eitem.Patches = append(eitem.Patches, Patch{
								Build:   next.Metadata,
								Actions: actions,
							})
						}
					case pitem != nil:
						eitem, _ := es.Latest[pitem].(*EnumItemEntity)
						if eitem == nil {
							panic(fmt.Errorf("no entity of enum item %s.%s", penum.Name, pitem.Name))
							continue
						}
						eitem.Patches = append(eitem.Patches, Patch{
							Build:   next.Metadata,
							Actions: actions,
						})
					}
				}
				for _, nitem := range nenum.Items {
					if pitem, _ := es.Links.Prev(nitem).(*rbxapijson.EnumItem); pitem != nil {
						continue
					}
					eitem := &EnumItemEntity{
						Base:   nitem,
						Latest: nitem,
						Patches: []Patch{{
							Build: next.Metadata,
							Actions: wrapActions((&diff.DiffEnumItem{
								Enum: nenum,
								Prev: nil,
								Next: nitem,
							}).Diff()),
						}},
					}
					eenum.Items = append(eenum.Items, eitem)
					es.Latest[nitem] = eitem
				}
			case penum != nil:
				eenum, _ := es.Latest[penum].(*EnumEntity)
				if eenum == nil {
					panic(fmt.Errorf("no entity of enum %s", penum.Name))
					continue
				}
				eenum.Patches = append(eenum.Patches, Patch{
					Build:   next.Metadata,
					Actions: actions,
				})
				for _, pitem := range penum.Items {
					eitem, _ := es.Latest[pitem].(*EnumItemEntity)
					if eitem == nil {
						continue
					}
					eitem.Patches = append(eitem.Patches, Patch{
						Build: next.Metadata,
						Actions: wrapActions((&diff.DiffEnumItem{
							Enum: penum,
							Prev: pitem,
							Next: nil,
						}).Diff()),
					})
				}
			}
		}
	}
	for _, nenum := range next.API.Enums {
		if penum, _ := es.Links.Prev(nenum).(*rbxapijson.Enum); penum != nil {
			continue
		}
		actions := wrapActions((&diff.DiffEnum{
			Prev:         nil,
			Next:         nenum,
			ExcludeItems: true,
		}).Diff())
		eenum := &EnumEntity{
			Base:   nenum,
			Latest: nenum,
			Patches: []Patch{{
				Build:   next.Metadata,
				Actions: actions,
			}},
		}
		es.Enums = append(es.Enums, eenum)
		es.Latest[nenum] = eenum
		for _, nitem := range nenum.Items {
			eitem := &EnumItemEntity{
				Base:   nitem,
				Latest: nitem,
				Patches: []Patch{{
					Build: next.Metadata,
					Actions: wrapActions((&diff.DiffEnumItem{
						Enum: nenum,
						Prev: nil,
						Next: nitem,
					}).Diff()),
				}},
			}
			eenum.Items = append(eenum.Items, eitem)
			es.Latest[nitem] = eitem
		}
	}
}

func (es *Entities) Compare(prev, next *Build) {
	if prev != nil {
		es.Links.Append(prev.API, next.API)
	}
	es.compareClasses(prev, next)
	es.compareEnums(prev, next)
}

func (es *Entities) Patches() []Patch {
	builds := map[string]*Patch{}
	for _, class := range es.Classes {
		for _, p := range class.Patches {
			patch := builds[p.Build.Hash]
			if patch == nil {
				patch = &Patch{Build: p.Build}
			}
			patch.Actions = append(patch.Actions, p.Actions...)
			builds[p.Build.Hash] = patch
		}
		for _, member := range class.Members {
			for _, p := range member.Patches {
				patch := builds[p.Build.Hash]
				if patch == nil {
					patch = &Patch{Build: p.Build}
				}
				patch.Actions = append(patch.Actions, p.Actions...)
				builds[p.Build.Hash] = patch
			}
		}
	}
	for _, enum := range es.Enums {
		for _, p := range enum.Patches {
			patch := builds[p.Build.Hash]
			if patch == nil {
				patch = &Patch{Build: p.Build}
			}
			patch.Actions = append(patch.Actions, p.Actions...)
			builds[p.Build.Hash] = patch
		}
		for _, item := range enum.Items {
			for _, p := range item.Patches {
				patch := builds[p.Build.Hash]
				if patch == nil {
					patch = &Patch{Build: p.Build}
				}
				patch.Actions = append(patch.Actions, p.Actions...)
				builds[p.Build.Hash] = patch
			}
		}
	}
	patches := make([]Patch, 0, len(builds))
	for _, p := range builds {
		patches = append(patches, *p)
	}
	sort.Slice(patches, func(i, j int) bool {
		return patches[i].Build.Date.Before(patches[j].Build.Date)
	})
	return patches
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
		s = path.Join(TypePath, url.PathEscape(name)+FileExt)
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
		patches := api.Entities.Patches()[1:]
		for i := len(patches)/2 - 1; i >= 0; i-- {
			j := len(patches) - 1 - i
			patches[i], patches[j] = patches[j], patches[i]
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
			Latest: map[interface{}]interface{}{},
			Links:  links.Make(nil, nil),
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
			next := &Build{builds[i], root}
			api.Entities.Compare(api.Latest, next)
			api.Latest = next
		}
	}

	var err error
	api.Templates, err = makeTemplates("templates", template.FuncMap{
		"tostring": toString,
		"link":     generateLink,
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
