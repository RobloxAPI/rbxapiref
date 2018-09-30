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

type ClassPageData struct {
	Name         string
	Entity       *ClassEntity
	Superclasses []string
	Subclasses   []string
	Members      []MemberSection
	Enums        []string
}

type MemberSection struct {
	Class   string
	Members []rbxapi.Member
}

type EnumPageData struct {
	Name    string
	Entity  *EnumEntity
	Members [][2]string
}

type TypePageData struct {
	Name    string
	Entity  rbxapijson.Type
	Members [][2]string
}

func getRelevantMembers(entities map[[2]string]*MemberEntity, typ rbxapijson.Type) (members [][2]string) {
	for id, member := range entities {
		switch member := member.Element.(type) {
		case *rbxapijson.Property:
			if member.ValueType == typ {
				goto addMember
			}
		case *rbxapijson.Function:
			if member.ReturnType == typ {
				goto addMember
			}
			for _, p := range member.Parameters {
				if p.Type == typ {
					goto addMember
				}
			}
		case *rbxapijson.Event:
			for _, p := range member.Parameters {
				if p.Type == typ {
					goto addMember
				}
			}
		case *rbxapijson.Callback:
			if member.ReturnType == typ {
				goto addMember
			}
			for _, p := range member.Parameters {
				if p.Type == typ {
					goto addMember
				}
			}
		}
		continue
	addMember:
		members = append(members, id)
	}
	sort.Slice(members, func(i, j int) bool {
		if members[i][0] == members[j][0] {
			return members[i][1] < members[j][1]
		}
		return members[i][0] < members[j][0]
	})
	return members
}

func buildPageData(data *Data, pageDataSet map[string]interface{}, typ string, args ...string) {
	link := data.FileLink(typ, args...)
	if _, ok := pageDataSet[link]; ok {
		return
	}
	switch typ {
	case "class":
		class := args[0]
		entity := data.Entities.Classes[class]
		if entity == nil || entity.Element == nil {
			return
		}
		pageData := ClassPageData{Name: class, Entity: entity}
		tree := data.Tree[class]
		if tree != nil {
			pageData.Superclasses = tree.Super
			pageData.Subclasses = tree.Sub
		}
		enums := map[string]struct{}{}
		for _, member := range entity.Element.Members {
			switch member := member.(type) {
			case *rbxapijson.Property:
				if member.ValueType.Category == "Enum" {
					enums[member.ValueType.Name] = struct{}{}
				}
			case *rbxapijson.Function:
				if member.ReturnType.Category == "Enum" {
					enums[member.ReturnType.Name] = struct{}{}
				}
				for _, p := range member.Parameters {
					if p.Type.Category == "Enum" {
						enums[p.Type.Name] = struct{}{}
					}
				}
			case *rbxapijson.Event:
				for _, p := range member.Parameters {
					if p.Type.Category == "Enum" {
						enums[p.Type.Name] = struct{}{}
					}
				}
			case *rbxapijson.Callback:
				if member.ReturnType.Category == "Enum" {
					enums[member.ReturnType.Name] = struct{}{}
				}
				for _, p := range member.Parameters {
					if p.Type.Category == "Enum" {
						enums[p.Type.Name] = struct{}{}
					}
				}
			}
		}
		pageData.Enums = make([]string, 0, len(enums))
		for enum := range enums {
			pageData.Enums = append(pageData.Enums, enum)
		}
		sort.Strings(pageData.Enums)
		for entity != nil && entity.Element != nil {
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
			pageData.Members = append(pageData.Members, MemberSection{class, members})
			class = entity.Element.Superclass
			entity = data.Entities.Classes[class]
		}
		pageDataSet[link] = &pageData
	case "enum":
		enum := args[0]
		entity := data.Entities.Enums[enum]
		if entity == nil {
			return
		}
		pageData := EnumPageData{Name: enum, Entity: entity}
		pageData.Members = getRelevantMembers(
			data.Entities.Members,
			rbxapijson.Type{Category: "Enum", Name: pageData.Name},
		)
		pageDataSet[link] = &pageData
	case "type":
		switch t := strings.ToLower(args[0]); t {
		case "class", "enum":
			buildPageData(data, pageDataSet, t, args[1])
			return
		}
		pageData := TypePageData{Name: args[1]}
		pageData.Entity = rbxapijson.Type{args[0], args[1]}
		pageData.Members = getRelevantMembers(data.Entities.Members, pageData.Entity)
		pageDataSet[link] = &pageData
	}
}

func buildTypePage(data *Data, pageDataSet map[string]interface{}, v interface{}) {
	switch v := v.(type) {
	case rbxapijson.Type:
		buildPageData(data, pageDataSet, "type", v.Category, v.Name)
	case []rbxapijson.Parameter:
		for _, p := range v {
			buildPageData(data, pageDataSet, "type", p.Type.Category, p.Type.Name)
		}
	}
}

func GenerateRefPage(data *Data) error {
	classPage := Page{
		CurrentYear: data.CurrentYear,
		Template:    "class",
		Styles:      []Resource{{Name: "class.css"}},
		Scripts:     []Resource{{Name: "class.js"}},
	}
	enumPage := Page{
		CurrentYear: data.CurrentYear,
		Template:    "enum",
		Styles:      []Resource{{Name: "enum.css"}},
		Scripts:     []Resource{},
	}
	typePage := Page{
		CurrentYear: data.CurrentYear,
		Template:    "type",
		Styles:      []Resource{},
		Scripts:     []Resource{},
	}

	pageDataSet := map[string]interface{}{}
	for _, patch := range data.Patches {
		// if !patch.Stale {
		// 	continue
		// }
		for _, action := range patch.Actions {
			switch {
			case action.Class != nil:
				buildPageData(data, pageDataSet, "class", action.Class.Name)
				buildPageData(data, pageDataSet, "class", action.Class.Superclass)
				for _, member := range action.Class.Members {
					switch member.GetMemberType() {
					case "Property":
						member := member.(*rbxapijson.Property)
						buildTypePage(data, pageDataSet, member.ValueType)
					case "Function":
						member := member.(*rbxapijson.Function)
						buildTypePage(data, pageDataSet, member.ReturnType)
						buildTypePage(data, pageDataSet, member.Parameters)
					case "Event":
						member := member.(*rbxapijson.Event)
						buildTypePage(data, pageDataSet, member.Parameters)
					case "Callback":
						member := member.(*rbxapijson.Callback)
						buildTypePage(data, pageDataSet, member.ReturnType)
						buildTypePage(data, pageDataSet, member.Parameters)
					}
				}
				member := action.GetMember()
				if member == nil {
					continue
				}
				switch member.GetMemberType() {
				case "Property":
					member := member.(*rbxapijson.Property)
					buildTypePage(data, pageDataSet, member.ValueType)
				case "Function":
					member := member.(*rbxapijson.Function)
					buildTypePage(data, pageDataSet, member.ReturnType)
					buildTypePage(data, pageDataSet, member.Parameters)
				case "Event":
					member := member.(*rbxapijson.Event)
					buildTypePage(data, pageDataSet, member.Parameters)
				case "Callback":
					member := member.(*rbxapijson.Callback)
					buildTypePage(data, pageDataSet, member.ReturnType)
					buildTypePage(data, pageDataSet, member.Parameters)
				}
			case action.Enum != nil:
				buildPageData(data, pageDataSet, "enum", action.Enum.Name)
			}
			buildTypePage(data, pageDataSet, action.GetPrev())
			buildTypePage(data, pageDataSet, action.GetNext())
		}
	}
	pageDataList := make([]string, 0, len(pageDataSet))
	for k := range pageDataSet {
		pageDataList = append(pageDataList, k)
	}
	sort.Strings(pageDataList)
	dirs := map[string]bool{}
	for _, link := range pageDataList {
		pageData := pageDataSet[link]
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
		var page *Page
		switch pageData := pageData.(type) {
		case *ClassPageData:
			page = &classPage
			page.Data = pageData
			page.Title = pageData.Name
		case *EnumPageData:
			page = &enumPage
			page.Data = pageData
			page.Title = pageData.Name
		case *TypePageData:
			page = &typePage
			page.Data = pageData
			page.Title = pageData.Name
		}
		if page != nil {
			err = GeneratePage(data, file, *page)
		}
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
