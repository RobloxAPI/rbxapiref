package main

import (
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/patch"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"sort"
)

type Entities struct {
	Classes   map[string]*ClassEntity
	ClassList []*ClassEntity
	Members   map[[2]string]*MemberEntity

	Enums     map[string]*EnumEntity
	EnumList  []*EnumEntity
	EnumItems map[[2]string]*EnumItemEntity

	Types    map[string]*TypeEntity
	TypeList []*TypeEntity
	TypeCats []TypeCategory
}

type Entity interface {
	IsRemoved() bool
}

// ElementTyper is implemented by an entity that can be referred to by an
// rbxapijson.Type.
type ElementTyper interface {
	Entity
	ElementType() rbxapijson.Type
}

type ClassEntity struct {
	ID      string
	Element *rbxapijson.Class
	Patches []Patch
	Removed bool

	Members    map[string]*MemberEntity
	MemberList []*MemberEntity

	References    map[rbxapijson.Type]ElementTyper
	ReferenceList []ElementTyper
	Referrers     map[[2]string]*MemberEntity
	ReferrerList  []*MemberEntity
}

func (e *ClassEntity) IsRemoved() bool { return e.Removed }
func (e *ClassEntity) ElementType() rbxapijson.Type {
	return rbxapijson.Type{Category: "Class", Name: e.Element.Name}
}

type MemberEntity struct {
	ID      [2]string
	Element rbxapi.Member
	Patches []Patch
	Removed bool

	Parent *ClassEntity

	References    map[rbxapijson.Type]ElementTyper
	ReferenceList []ElementTyper
}

func (e *MemberEntity) IsRemoved() bool { return e.Removed }

type EnumEntity struct {
	ID      string
	Element *rbxapijson.Enum
	Patches []Patch
	Removed bool

	Items    map[string]*EnumItemEntity
	ItemList []*EnumItemEntity

	Referrers    map[[2]string]*MemberEntity
	ReferrerList []*MemberEntity
}

func (e *EnumEntity) IsRemoved() bool { return e.Removed }
func (e *EnumEntity) ElementType() rbxapijson.Type {
	return rbxapijson.Type{Category: "Enum", Name: e.Element.Name}
}

type EnumItemEntity struct {
	ID      [2]string
	Element *rbxapijson.EnumItem
	Patches []Patch
	Removed bool

	Parent *EnumEntity
}

func (e *EnumItemEntity) IsRemoved() bool { return e.Removed }

type TypeEntity struct {
	ID      string
	Element rbxapijson.Type
	Removed bool

	Referrers    map[[2]string]*MemberEntity
	ReferrerList []*MemberEntity
}

func (e *TypeEntity) IsRemoved() bool { return e.Removed }
func (e *TypeEntity) ElementType() rbxapijson.Type {
	return e.Element
}

type TypeCategory struct {
	Name  string
	Types []*TypeEntity
}

func addPatch(patches *[]Patch, action *Action, info BuildInfo) {
	for i := len(*patches) - 1; i >= 0; i-- {
		patch := (*patches)[i]
		if patch.Info.Equal(info) {
			patch.Actions = append(patch.Actions, *action)
			return
		}
	}
	*patches = append(*patches, Patch{
		Info:    info,
		Actions: []Action{*action},
	})
}

func (entities *Entities) AddClass(action *Action, info BuildInfo) {
	class := action.Class
	id := class.Name
	eclass := entities.Classes[id]
	if eclass == nil {
		eclass = &ClassEntity{
			ID:         id,
			Element:    class.Copy().(*rbxapijson.Class),
			Members:    map[string]*MemberEntity{},
			References: map[rbxapijson.Type]ElementTyper{},
			Referrers:  map[[2]string]*MemberEntity{},
		}
		entities.Classes[id] = eclass
	}
	addPatch(&eclass.Patches, action, info)
	switch action.Type {
	case patch.Add:
		eclass.Element = class.Copy().(*rbxapijson.Class)
		eclass.Removed = false
		for _, member := range class.Members {
			id := [2]string{class.Name, member.GetName()}
			emember := eclass.Members[id[1]]
			if emember == nil {
				emember = &MemberEntity{
					ID:         id,
					Element:    member.Copy(),
					References: map[rbxapijson.Type]ElementTyper{},
					Parent:     eclass,
				}
				eclass.Members[id[1]] = emember
				entities.Members[id] = emember
			}
			emember.Element = member.Copy()
			emember.Removed = false
		}
	case patch.Remove:
		eclass.Removed = true
	case patch.Change:
		eclass.Element.Patch([]patch.Action{action})
	}
}

func (entities *Entities) AddMember(action *Action, info BuildInfo) {
	class := action.Class
	member := action.GetMember()
	id := [2]string{class.Name, member.GetName()}
	eclass := entities.Classes[id[0]]
	if eclass == nil {
		panic("missing class of member entity")
	}
	emember := eclass.Members[id[1]]
	if emember == nil {
		emember = &MemberEntity{
			ID:         id,
			Element:    member.Copy(),
			References: map[rbxapijson.Type]ElementTyper{},
			Parent:     eclass,
		}
		eclass.Members[id[1]] = emember
		entities.Members[id] = emember
	}
	addPatch(&emember.Patches, action, info)
	eclass.Element.Patch([]patch.Action{patch.Member(action)})
	switch action.Type {
	case patch.Add:
		emember.Element = member.Copy()
		emember.Removed = false
	case patch.Remove:
		emember.Removed = true
	case patch.Change:
		emember.Element.(patch.Patcher).Patch([]patch.Action{action})
	}
}

func (entities *Entities) AddEnum(action *Action, info BuildInfo) {
	enum := action.Enum
	id := enum.Name
	eenum := entities.Enums[id]
	if eenum == nil {
		eenum = &EnumEntity{
			ID:        id,
			Element:   enum.Copy().(*rbxapijson.Enum),
			Items:     map[string]*EnumItemEntity{},
			Referrers: map[[2]string]*MemberEntity{},
		}
		entities.Enums[id] = eenum
	}
	addPatch(&eenum.Patches, action, info)
	switch action.Type {
	case patch.Add:
		eenum.Element = enum.Copy().(*rbxapijson.Enum)
		eenum.Removed = false
		for _, item := range enum.Items {
			id := [2]string{enum.Name, item.Name}
			eitem := eenum.Items[id[1]]
			if eitem == nil {
				eitem = &EnumItemEntity{
					ID:      id,
					Element: item.Copy().(*rbxapijson.EnumItem),
					Parent:  eenum,
				}
				eenum.Items[id[1]] = eitem
				entities.EnumItems[id] = eitem
			}
			eitem.Element = item.Copy().(*rbxapijson.EnumItem)
			eitem.Removed = false
		}
	case patch.Remove:
		eenum.Removed = true
	case patch.Change:
		eenum.Element.Patch([]patch.Action{action})
	}
}

func (entities *Entities) AddEnumItem(action *Action, info BuildInfo) {
	enum := action.Enum
	item := action.EnumItem
	id := [2]string{enum.Name, item.Name}
	eenum := entities.Enums[id[0]]
	if eenum == nil {
		panic("missing enum of enumitem entity")
	}
	eitem := eenum.Items[id[1]]
	if eitem == nil {
		eitem = &EnumItemEntity{
			ID:      id,
			Element: item.Copy().(*rbxapijson.EnumItem),
			Parent:  eenum,
		}
		eenum.Items[id[1]] = eitem
		entities.EnumItems[id] = eitem
	}
	addPatch(&eitem.Patches, action, info)
	eenum.Element.Patch([]patch.Action{action})
	switch action.Type {
	case patch.Add:
		eitem.Element = item.Copy().(*rbxapijson.EnumItem)
		eitem.Removed = false
	case patch.Remove:
		eitem.Removed = true
	case patch.Change:
		eitem.Element.Patch([]patch.Action{action})
	}
}

var memberTypeOrder = map[string]int{
	"Property": 0,
	"Function": 1,
	"Event":    2,
	"Callback": 3,
}

func GenerateEntities(data *Data) (entities *Entities) {
	entities = &Entities{
		Classes:   make(map[string]*ClassEntity),
		Members:   make(map[[2]string]*MemberEntity),
		Enums:     make(map[string]*EnumEntity),
		EnumItems: make(map[[2]string]*EnumItemEntity),
		Types:     make(map[string]*TypeEntity),
	}

	for _, patch := range data.Patches {
		for _, action := range patch.Actions {
			switch {
			case action.EnumItem != nil:
				entities.AddEnumItem(&action, patch.Info)
			case action.Enum != nil:
				entities.AddEnum(&action, patch.Info)
			case action.GetMember() != nil:
				entities.AddMember(&action, patch.Info)
			case action.Class != nil:
				entities.AddClass(&action, patch.Info)
			}
		}
	}

	referType := func(emember *MemberEntity, typ rbxapijson.Type) {
		if emember.Removed {
			return
		}
		var et ElementTyper
		switch typ.Category {
		case "Class":
			eclass := entities.Classes[typ.Name]
			if eclass == nil {
				return
			}
			if _, ok := eclass.Referrers[emember.ID]; !ok {
				eclass.Referrers[emember.ID] = emember
				eclass.ReferrerList = append(eclass.ReferrerList, emember)
			}
			et = eclass
		case "Enum":
			eenum := entities.Enums[typ.Name]
			if eenum == nil {
				return
			}
			if _, ok := eenum.Referrers[emember.ID]; !ok {
				eenum.Referrers[emember.ID] = emember
				eenum.ReferrerList = append(eenum.ReferrerList, emember)
			}
			et = eenum
		default:
			etype := entities.Types[typ.Name]
			if etype == nil {
				etype = &TypeEntity{
					ID:        typ.Name,
					Element:   typ,
					Removed:   true,
					Referrers: map[[2]string]*MemberEntity{},
				}
				entities.Types[typ.Name] = etype
			}
			if !emember.Removed && !emember.Parent.Removed {
				etype.Removed = false
			}
			if _, ok := etype.Referrers[emember.ID]; !ok {
				etype.Referrers[emember.ID] = emember
				etype.ReferrerList = append(etype.ReferrerList, emember)
			}
			et = etype
		}
		if emember.References[typ] == nil {
			emember.References[typ] = et
			emember.ReferenceList = append(emember.ReferenceList, et)
		}
	}

	for _, entity := range entities.Members {
		switch element := entity.Element.(type) {
		case *rbxapijson.Property:
			referType(entity, element.ValueType)
		case *rbxapijson.Function:
			referType(entity, element.ReturnType)
			for _, param := range element.Parameters {
				referType(entity, param.Type)
			}
		case *rbxapijson.Event:
			for _, param := range element.Parameters {
				referType(entity, param.Type)
			}
		case *rbxapijson.Callback:
			referType(entity, element.ReturnType)
			for _, param := range element.Parameters {
				referType(entity, param.Type)
			}
		}
	}

	for _, eclass := range entities.Classes {
		for _, emember := range eclass.Members {
			sort.Slice(emember.ReferenceList, func(i, j int) bool {
				it := emember.ReferenceList[i].ElementType()
				jt := emember.ReferenceList[j].ElementType()
				if it.Category == jt.Category {
					return it.Name < jt.Name
				}
				return it.Category < jt.Category
			})
			for _, et := range emember.ReferenceList {
				if typ := et.ElementType(); eclass.References[typ] == nil {
					eclass.References[typ] = et
					eclass.ReferenceList = append(eclass.ReferenceList, et)
				}
			}
		}
		sort.Slice(eclass.ReferenceList, func(i, j int) bool {
			it := eclass.ReferenceList[i].ElementType()
			jt := eclass.ReferenceList[j].ElementType()
			if it.Category == jt.Category {
				return it.Name < jt.Name
			}
			return it.Category < jt.Category
		})
		sort.Slice(eclass.ReferrerList, func(i, j int) bool {
			if eclass.ReferrerList[i].ID[0] == eclass.ReferrerList[j].ID[0] {
				return eclass.ReferrerList[i].ID[1] < eclass.ReferrerList[j].ID[1]
			}
			return eclass.ReferrerList[i].ID[0] < eclass.ReferrerList[j].ID[0]
		})
	}

	for _, eenum := range entities.Enums {
		sort.Slice(eenum.ReferrerList, func(i, j int) bool {
			if eenum.ReferrerList[i].ID[0] == eenum.ReferrerList[j].ID[0] {
				return eenum.ReferrerList[i].ID[1] < eenum.ReferrerList[j].ID[1]
			}
			return eenum.ReferrerList[i].ID[0] < eenum.ReferrerList[j].ID[0]
		})
	}

	for _, etype := range entities.Types {
		sort.Slice(etype.ReferrerList, func(i, j int) bool {
			if etype.ReferrerList[i].ID[0] == etype.ReferrerList[j].ID[0] {
				return etype.ReferrerList[i].ID[1] < etype.ReferrerList[j].ID[1]
			}
			return etype.ReferrerList[i].ID[0] < etype.ReferrerList[j].ID[0]
		})
	}

	{
		entities.ClassList = make([]*ClassEntity, len(entities.Classes))
		i := 0
		for _, eclass := range entities.Classes {
			entities.ClassList[i] = eclass
			i++

			eclass.MemberList = make([]*MemberEntity, len(eclass.Members))
			j := 0
			for _, emember := range eclass.Members {
				eclass.MemberList[j] = emember
				j++
			}
			sort.Slice(eclass.MemberList, func(i, j int) bool {
				it := memberTypeOrder[eclass.MemberList[i].Element.GetMemberType()]
				jt := memberTypeOrder[eclass.MemberList[j].Element.GetMemberType()]
				if it == jt {
					return eclass.MemberList[i].ID[1] < eclass.MemberList[j].ID[1]
				}
				return it < jt
			})
		}
		sort.Slice(entities.ClassList, func(i, j int) bool {
			return entities.ClassList[i].ID < entities.ClassList[j].ID
		})
	}

	{
		entities.EnumList = make([]*EnumEntity, len(entities.Enums))
		i := 0
		for _, eenum := range entities.Enums {
			entities.EnumList[i] = eenum
			i++

			eenum.ItemList = make([]*EnumItemEntity, len(eenum.Items))
			j := 0
			for _, emember := range eenum.Items {
				eenum.ItemList[j] = emember
				j++
			}
			sort.Slice(eenum.ItemList, func(i, j int) bool {
				return eenum.ItemList[i].Element.Value < eenum.ItemList[j].Element.Value
			})
		}
		sort.Slice(entities.EnumList, func(i, j int) bool {
			return entities.EnumList[i].ID < entities.EnumList[j].ID
		})
	}

	entities.TypeList = make([]*TypeEntity, 0, len(entities.Types))
loop:
	for _, etype := range entities.Types {
		entities.TypeList = append(entities.TypeList, etype)
		for i, cat := range entities.TypeCats {
			if cat.Name == etype.Element.Category {
				entities.TypeCats[i].Types = append(entities.TypeCats[i].Types, etype)
				continue loop
			}
		}
		entities.TypeCats = append(entities.TypeCats, TypeCategory{
			Name:  etype.Element.Category,
			Types: []*TypeEntity{etype},
		})
	}
	sort.Slice(entities.TypeList, func(i, j int) bool {
		return entities.TypeList[i].ID < entities.TypeList[j].ID
	})
	sort.Slice(entities.TypeCats, func(i, j int) bool {
		return entities.TypeCats[i].Name < entities.TypeCats[j].Name
	})
	for _, cat := range entities.TypeCats {
		sort.Slice(cat.Types, func(i, j int) bool {
			return cat.Types[i].ID < cat.Types[j].ID
		})
	}

	return entities
}
