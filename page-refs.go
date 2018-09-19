package main

import (
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	Name    string
	Entity  *EnumEntity
	Members [][2]string
}

type TypePage struct {
	Name   string
	Entity rbxapijson.Type
}

func buildPageData(data *Data, pageSet map[string]interface{}, typ string, args ...string) {
	link := data.FileLink(typ, args...)
	if _, ok := pageSet[link]; ok {
		return
	}
	switch typ {
	case "class":
		class := args[0]
		entity := data.Entities.Classes[class]
		if entity == nil {
			return
		}
		page := ClassPage{Name: class, Entity: entity}
		tree := data.Tree[class]
		if tree != nil {
			page.Superclasses = tree.Super
			page.Subclasses = tree.Sub
		}
		for {
			entity = data.Entities.Classes[class]
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
		enum := args[0]
		entity := data.Entities.Enums[enum]
		if entity == nil {
			return
		}
		page := EnumPage{Name: enum, Entity: entity}
		enumType := rbxapijson.Type{Category: "Enum", Name: page.Name}
		for id, member := range data.Entities.Members {
			switch member := member.Element.(type) {
			case *rbxapijson.Property:
				if member.ValueType == enumType {
					goto add
				}
			case *rbxapijson.Function:
				if member.ReturnType == enumType {
					goto add
				}
				for _, p := range member.Parameters {
					if p.Type == enumType {
						goto add
					}
				}
			case *rbxapijson.Event:
				for _, p := range member.Parameters {
					if p.Type == enumType {
						goto add
					}
				}
			case *rbxapijson.Callback:
				if member.ReturnType == enumType {
					goto add
				}
				for _, p := range member.Parameters {
					if p.Type == enumType {
						goto add
					}
				}
			}
			continue
		add:
			page.Members = append(page.Members, id)
		}
		sort.Slice(page.Members, func(i, j int) bool {
			if page.Members[i][0] == page.Members[j][0] {
				return page.Members[i][1] < page.Members[j][1]
			}
			return page.Members[i][0] < page.Members[j][0]
		})
		pageSet[link] = &page
	case "type":
		switch t := strings.ToLower(args[0]); t {
		case "class", "enum":
			buildPageData(data, pageSet, t, args[1])
			return
		}
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
