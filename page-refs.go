package main

import (
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"os"
	"path/filepath"
	"sort"
)

var memberTypeOrder = map[string]int{
	"Property": 0,
	"Function": 1,
	"Event":    2,
	"Callback": 3,
}

type ClassPage struct {
	Name         string
	Entity       *ClassEntity
	Superclasses []string
	Subclasses   []string
	Members      []MemberSection
}

type MemberSection struct {
	Class   string
	Members []rbxapi.Member
}

type EnumPage struct {
	Name   string
	Entity *EnumEntity
}

type TypePage struct {
	Name   string
	Entity rbxapijson.Type
}

func buildPageData(data *Data, pageSet map[string]interface{}, typ string, args ...string) {
	link := data.FileLink(typ, args...)
	if pageSet[link] != nil {
		return
	}
	switch typ {
	case "class":
		class := args[0]
		if data.Entities.Classes[class] == nil {
			return
		}
		page := ClassPage{Name: class}
		tree := data.Tree[class]
		if tree != nil {
			page.Superclasses = tree.Super
			page.Subclasses = tree.Sub
		}
		page.Entity = data.Entities.Classes[class]
		for {
			entity := data.Entities.Classes[class]
			if entity == nil || entity.Element == nil {
				break
			}
			members := make([]rbxapi.Member, len(entity.Element.Members))
			copy(members, entity.Element.Members)
			sort.Slice(members, func(i, j int) bool {
				it := memberTypeOrder[members[i].GetMemberType()]
				jt := memberTypeOrder[members[j].GetMemberType()]
				if it == jt {
					return members[i].GetName() < members[j].GetName()
				}
				return it < jt
			})
			page.Members = append(page.Members, MemberSection{class, members})
			class = entity.Element.Superclass
		}
		pageSet[link] = &page
	case "enum":
		page := EnumPage{Name: args[0]}
		page.Entity = data.Entities.Enums[args[0]]
		pageSet[link] = &page
	case "type":
		page := TypePage{Name: args[1]}
		page.Entity = rbxapijson.Type{args[0], args[1]}
		pageSet[link] = &page
	}
}

func buildTypePage(data *Data, pageSet map[string]interface{}, v interface{}) {
	switch v := v.(type) {
	case rbxapijson.Type:
		buildPageData(data, pageSet, "type", v.Category, v.Name)
	case []rbxapijson.Parameter:
		for _, p := range v {
			buildPageData(data, pageSet, "type", p.Type.Category, p.Type.Name)
		}
	}
}

func GenerateRefPage(data *Data) error {
	pageSet := map[string]interface{}{}
	for _, patch := range data.Patches {
		// if !patch.Stale {
		// 	continue
		// }
		for _, action := range patch.Actions {
			switch {
			case action.Class != nil:
				buildPageData(data, pageSet, "class", action.Class.Name)
				buildPageData(data, pageSet, "class", action.Class.Superclass)
				for _, member := range action.Class.Members {
					switch member.GetMemberType() {
					case "Property":
						member := member.(*rbxapijson.Property)
						buildTypePage(data, pageSet, member.ValueType)
					case "Function":
						member := member.(*rbxapijson.Function)
						buildTypePage(data, pageSet, member.ReturnType)
						buildTypePage(data, pageSet, member.Parameters)
					case "Event":
						member := member.(*rbxapijson.Event)
						buildTypePage(data, pageSet, member.Parameters)
					case "Callback":
						member := member.(*rbxapijson.Callback)
						buildTypePage(data, pageSet, member.ReturnType)
						buildTypePage(data, pageSet, member.Parameters)
					}
				}
			case action.Enum != nil:
				buildPageData(data, pageSet, "enum", action.Enum.Name)
			}
			buildTypePage(data, pageSet, action.GetPrev())
			buildTypePage(data, pageSet, action.GetNext())
		}
	}
	pages := make([]string, 0, len(pageSet))
	for k := range pageSet {
		pages = append(pages, k)
	}
	sort.Strings(pages)
	dirs := map[string]bool{}
	for _, link := range pages {
		page := pageSet[link]
		path := data.PathFromLink(link)
		if dir := filepath.Dir(path); !dirs[dir] {
			if err := os.MkdirAll(dir, 0666); err != nil {
				return err
			}
			dirs[dir] = true
		}
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		switch page := page.(type) {
		case *ClassPage:
			err = data.Templates.ExecuteTemplate(file, "class", page)
		case *EnumPage:
			err = data.Templates.ExecuteTemplate(file, "enum", page)
		case *TypePage:
			err = data.Templates.ExecuteTemplate(file, "type", page)
		}
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
