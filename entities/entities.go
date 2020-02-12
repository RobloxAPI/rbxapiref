package entities

import (
	"fmt"
	"html/template"
	"sort"
	"strconv"

	"github.com/gomarkdown/markdown"
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/patch"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"github.com/robloxapi/rbxapiref/builds"
	"github.com/robloxapi/rbxapiref/documents"
	"github.com/robloxapi/rbxfile"
)

type Entities struct {
	Classes   map[string]*ClassEntity
	ClassList []*ClassEntity
	Members   map[[2]string]*MemberEntity
	TreeRoots []*ClassEntity

	Enums     map[string]*EnumEntity
	EnumList  []*EnumEntity
	EnumItems map[[2]string]*EnumItemEntity

	Types    map[string]*TypeEntity
	TypeList []*TypeEntity
	TypeCats []TypeCategory

	Coverage float32
}

func (e *Entities) CoverageString() string {
	return fmt.Sprintf("%.2f%%", e.Coverage*100)
}

type DocStatus struct {
	HasDocument     bool
	SummaryStatus   int // 0:nofile; 1:nosection; 2:empty; 3:filled
	SummaryOrphaned bool

	DetailsStatus   int // 0:nofile; 1:nosection; 2:empty; 3:filled
	DetailsSections int

	ExamplesStatus int // 0:nofile; 1:nosection; 2:empty; 3:filled
	ExampleCount   int

	AggregateStatus   int // 0:nofile; 1:none; 2:some; 3:all
	AggregateProgress float64
}

func (s DocStatus) StatusString(status int) string {
	switch status {
	case 0:
		return "d"
	case 1:
		return "c"
	case 2:
		return "b"
	case 3:
		return "a"
	}
	return ""
}
func (s DocStatus) ProgressString() string {
	return fmt.Sprintf("%.2f%%", s.AggregateProgress*100)
}

type Entity interface {
	IsRemoved() bool
}

type Document interface {
	Query(name ...string) documents.Section
	SetRender(renderer markdown.Renderer)
	Render() template.HTML
}

type Documentable interface {
	GetDocument() Document
	GetDocStatus() DocStatus
}

func QueryDocument(d Documentable, name ...string) documents.Section {
	doc := d.GetDocument()
	if doc == nil {
		return nil
	}
	return doc.Query(name...)
}

func RenderDocument(s documents.Section, level int) template.HTML {
	if h, ok := s.(documents.Headingable); ok {
		root := h.RootLevel()
		h.AdjustLevels(level)
		render := s.Render()
		h.AdjustLevels(root)
		return render
	}
	return s.Render()
}

// ElementTyper is implemented by an entity that can be referred to by an
// rbxapijson.Type.
type ElementTyper interface {
	Entity
	Identifier() string
	ElementType() rbxapijson.Type
}

type Metadata struct {
	*rbxfile.Instance
}

func (m Metadata) GetInt(prop string) (i int) {
	switch v := m.Get(prop).(type) {
	case rbxfile.ValueInt:
		i = int(v)
	case rbxfile.ValueString:
		i, _ = strconv.Atoi(string(v))
	}
	return i
}

type ClassEntity struct {
	ID      string
	Element *rbxapijson.Class
	Patches []builds.Patch
	Removed bool

	Superclasses []*ClassEntity
	Subclasses   []*ClassEntity

	Members    map[string]*MemberEntity
	MemberList []*MemberEntity

	References    map[rbxapijson.Type]ElementTyper
	ReferenceList []ElementTyper
	Referrers     map[[2]string]Referrer
	ReferrerList  []Referrer

	Document  Document
	DocStatus DocStatus
	Metadata  Metadata
}

func (e *ClassEntity) IsRemoved() bool    { return e.Removed }
func (e *ClassEntity) Identifier() string { return e.ID }
func (e *ClassEntity) ElementType() rbxapijson.Type {
	return rbxapijson.Type{Category: "Class", Name: e.Element.Name}
}
func (e *ClassEntity) GetDocument() Document   { return e.Document }
func (e *ClassEntity) GetDocStatus() DocStatus { return e.DocStatus }

type MemberEntity struct {
	ID      [2]string
	Element rbxapi.Member
	Patches []builds.Patch
	Removed bool

	Parent *ClassEntity

	References    map[rbxapijson.Type]ElementTyper
	ReferenceList []ElementTyper

	Document  Document
	DocStatus DocStatus
	Metadata  Metadata
}

func (e *MemberEntity) IsRemoved() bool         { return e.Removed }
func (e *MemberEntity) GetDocument() Document   { return e.Document }
func (e *MemberEntity) GetDocStatus() DocStatus { return e.DocStatus }

type EnumEntity struct {
	ID      string
	Element *rbxapijson.Enum
	Patches []builds.Patch
	Removed bool

	Items    map[string]*EnumItemEntity
	ItemList []*EnumItemEntity

	Referrers    map[[2]string]Referrer
	ReferrerList []Referrer

	Document  Document
	DocStatus DocStatus
	Metadata  Metadata
}

func (e *EnumEntity) IsRemoved() bool    { return e.Removed }
func (e *EnumEntity) Identifier() string { return e.ID }
func (e *EnumEntity) ElementType() rbxapijson.Type {
	return rbxapijson.Type{Category: "Enum", Name: e.Element.Name}
}
func (e *EnumEntity) GetDocument() Document   { return e.Document }
func (e *EnumEntity) GetDocStatus() DocStatus { return e.DocStatus }

type EnumItemEntity struct {
	ID      [2]string
	Element *rbxapijson.EnumItem
	Patches []builds.Patch
	Removed bool

	Parent *EnumEntity

	Document  Document
	DocStatus DocStatus
	Metadata  Metadata
}

func (e *EnumItemEntity) IsRemoved() bool         { return e.Removed }
func (e *EnumItemEntity) GetDocument() Document   { return e.Document }
func (e *EnumItemEntity) GetDocStatus() DocStatus { return e.DocStatus }

type TypeEntity struct {
	ID      string
	Element rbxapijson.Type
	Removed bool

	Referrers    map[[2]string]Referrer
	ReferrerList []Referrer

	RemovedRefs    map[[2]string]Referrer
	RemovedRefList []Referrer

	Document  Document
	DocStatus DocStatus
	Metadata  Metadata
}

func (e *TypeEntity) IsRemoved() bool    { return e.Removed }
func (e *TypeEntity) Identifier() string { return e.ID }
func (e *TypeEntity) ElementType() rbxapijson.Type {
	return e.Element
}
func (e *TypeEntity) GetDocument() Document   { return e.Document }
func (e *TypeEntity) GetDocStatus() DocStatus { return e.DocStatus }

type TypeCategory struct {
	Name  string
	Types []*TypeEntity
}

type Referrer struct {
	Member    *MemberEntity
	Parameter *rbxapijson.Parameter
}

func addPatch(patches *[]builds.Patch, action *builds.Action, info builds.Info) {
	for i := len(*patches) - 1; i >= 0; i-- {
		if (*patches)[i].Info.Equal(info) {
			(*patches)[i].Actions = append((*patches)[i].Actions, *action)
			return
		}
	}
	*patches = append(*patches, builds.Patch{
		Info:    info,
		Actions: []builds.Action{*action},
	})
}

func (entities *Entities) AddClass(action *builds.Action, info builds.Info) {
	class := action.Class
	id := class.Name
	eclass := entities.Classes[id]
	if eclass == nil {
		eclass = &ClassEntity{
			ID:         id,
			Element:    class.Copy().(*rbxapijson.Class),
			Members:    map[string]*MemberEntity{},
			References: map[rbxapijson.Type]ElementTyper{},
			Referrers:  map[[2]string]Referrer{},
		}
		entities.Classes[id] = eclass
	}
	switch action.Type {
	case patch.Add:
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
		if eclass.Element != class && eclass.Element != nil {
			for _, member := range eclass.Element.Members {
				emember := eclass.Members[member.GetName()]
				if emember == nil {
					continue
				}
				if class.GetMember(member.GetName()) == nil {
					// Retroactively remove the member.
					emember.Removed = true
					if len(eclass.Patches) == 0 {
						continue
					}
					p := eclass.Patches[len(eclass.Patches)-1]
					// TODO: Is it possible for an entity patch which removes
					// the entity to have any action other than the removal?
					addPatch(&emember.Patches, &p.Actions[0], p.Info)
				}
			}
			for _, member := range class.Members {
				emember := eclass.Members[member.GetName()]
				if emember == nil {
					continue
				}
				if eclass.Element.GetMember(member.GetName()) == nil {
					// Include the patch that adds the member.
					addPatch(&emember.Patches, action, info)
				}
			}
		}
		eclass.Element = class.Copy().(*rbxapijson.Class)
		eclass.Removed = false
	case patch.Remove:
		eclass.Removed = true
	case patch.Change:
		eclass.Element.Patch([]patch.Action{action})
	}
	addPatch(&eclass.Patches, action, info)
}

func (entities *Entities) AddMember(action *builds.Action, info builds.Info) {
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

func (entities *Entities) AddEnum(action *builds.Action, info builds.Info) {
	enum := action.Enum
	id := enum.Name
	eenum := entities.Enums[id]
	if eenum == nil {
		eenum = &EnumEntity{
			ID:        id,
			Element:   enum.Copy().(*rbxapijson.Enum),
			Items:     map[string]*EnumItemEntity{},
			Referrers: map[[2]string]Referrer{},
		}
		entities.Enums[id] = eenum
	}
	switch action.Type {
	case patch.Add:
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
		if eenum.Element != enum && eenum.Element != nil {
			for _, item := range eenum.Element.Items {
				eitem := eenum.Items[item.Name]
				if eitem == nil {
					continue
				}
				if enum.GetEnumItem(item.Name) == nil {
					// Retroactively remove the item.
					eitem.Removed = true
					if len(eenum.Patches) == 0 {
						continue
					}
					p := eenum.Patches[len(eenum.Patches)-1]
					addPatch(&eitem.Patches, &p.Actions[0], p.Info)
				}
			}
		}
		eenum.Element = enum.Copy().(*rbxapijson.Enum)
		eenum.Removed = false
	case patch.Remove:
		eenum.Removed = true
	case patch.Change:
		eenum.Element.Patch([]patch.Action{action})
	}
	addPatch(&eenum.Patches, action, info)
}

func (entities *Entities) AddEnumItem(action *builds.Action, info builds.Info) {
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

func (entities *Entities) ListAll() []interface{} {
	n := len(entities.Classes)
	n += len(entities.Members)
	n += len(entities.Enums)
	n += len(entities.EnumItems)
	n += len(entities.TypeCats)
	n += len(entities.Types)
	all := make([]interface{}, 0, n)

	var addClasses func(classes []*ClassEntity)
	addClasses = func(classes []*ClassEntity) {
		for _, class := range classes {
			all = append(all, class)
			for _, member := range class.MemberList {
				all = append(all, member)
			}
			addClasses(class.Subclasses)
		}
	}
	addClasses(entities.TreeRoots)

	for _, enum := range entities.EnumList {
		all = append(all, enum)
		for _, item := range enum.ItemList {
			all = append(all, item)
		}
	}

	for _, cat := range entities.TypeCats {
		all = append(all, cat)
		for _, typ := range cat.Types {
			all = append(all, typ)
		}
	}

	return all
}

var memberTypeOrder = map[string]int{
	"Property": 0,
	"Function": 1,
	"Event":    2,
	"Callback": 3,
}

func GenerateEntities(patches []builds.Patch) (entities *Entities) {
	entities = &Entities{
		Classes:   make(map[string]*ClassEntity),
		Members:   make(map[[2]string]*MemberEntity),
		Enums:     make(map[string]*EnumEntity),
		EnumItems: make(map[[2]string]*EnumItemEntity),
		Types:     make(map[string]*TypeEntity),
	}

	for _, patch := range patches {
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

	referType := func(referrer Referrer, typ rbxapijson.Type, current bool) {
		var et ElementTyper
		switch typ.Category {
		case "Class":
			if !current {
				return
			}
			if referrer.Member.Removed {
				return
			}
			eclass := entities.Classes[typ.Name]
			if eclass == nil {
				return
			}
			if _, ok := eclass.Referrers[referrer.Member.ID]; !ok {
				eclass.Referrers[referrer.Member.ID] = referrer
				eclass.ReferrerList = append(eclass.ReferrerList, referrer)
			}
			et = eclass
		case "Enum":
			if !current {
				return
			}
			if referrer.Member.Removed {
				return
			}
			eenum := entities.Enums[typ.Name]
			if eenum == nil {
				return
			}
			if _, ok := eenum.Referrers[referrer.Member.ID]; !ok {
				eenum.Referrers[referrer.Member.ID] = referrer
				eenum.ReferrerList = append(eenum.ReferrerList, referrer)
			}
			et = eenum
		default:
			etype := entities.Types[typ.Name]
			if etype == nil {
				etype = &TypeEntity{
					ID:          typ.Name,
					Element:     typ,
					Removed:     true,
					Referrers:   map[[2]string]Referrer{},
					RemovedRefs: map[[2]string]Referrer{},
				}
				entities.Types[typ.Name] = etype
			}
			if !current || referrer.Member.Removed || referrer.Member.Parent.Removed {
				if _, ok := etype.RemovedRefs[referrer.Member.ID]; !ok {
					etype.RemovedRefs[referrer.Member.ID] = referrer
					etype.RemovedRefList = append(etype.RemovedRefList, referrer)
				}
			}
			if !current {
				return
			}
			if referrer.Member.Removed {
				return
			}
			if !referrer.Member.Removed && !referrer.Member.Parent.Removed {
				etype.Removed = false
			}
			if _, ok := etype.Referrers[referrer.Member.ID]; !ok {
				etype.Referrers[referrer.Member.ID] = referrer
				etype.ReferrerList = append(etype.ReferrerList, referrer)
			}
			et = etype
		}
		if referrer.Member.References[typ] == nil {
			referrer.Member.References[typ] = et
			referrer.Member.ReferenceList = append(referrer.Member.ReferenceList, et)
		}
	}

	for _, entity := range entities.Members {
		switch element := entity.Element.(type) {
		case *rbxapijson.Property:
			referType(Referrer{entity, nil}, element.ValueType, true)
			for _, p := range entity.Patches {
				for _, a := range p.Actions {
					member := a.Property
					if member == nil {
						continue
					}
					referType(Referrer{entity, nil}, member.ValueType, false)
				}
			}
		case *rbxapijson.Function:
			referType(Referrer{entity, nil}, element.ReturnType, true)
			for i, param := range element.Parameters {
				referType(Referrer{entity, &element.Parameters[i]}, param.Type, true)
			}
			for _, p := range entity.Patches {
				for _, a := range p.Actions {
					member := a.Function
					if member == nil {
						continue
					}
					referType(Referrer{entity, nil}, member.ReturnType, false)
					for i, param := range member.Parameters {
						referType(Referrer{entity, &member.Parameters[i]}, param.Type, false)
					}
				}
			}
		case *rbxapijson.Event:
			for i, param := range element.Parameters {
				referType(Referrer{entity, &element.Parameters[i]}, param.Type, true)
			}
			for _, p := range entity.Patches {
				for _, a := range p.Actions {
					member := a.Event
					if member == nil {
						continue
					}
					for i, param := range member.Parameters {
						referType(Referrer{entity, &member.Parameters[i]}, param.Type, false)
					}
				}
			}
		case *rbxapijson.Callback:
			referType(Referrer{entity, nil}, element.ReturnType, true)
			for i, param := range element.Parameters {
				referType(Referrer{entity, &element.Parameters[i]}, param.Type, true)
			}
			for _, p := range entity.Patches {
				for _, a := range p.Actions {
					member := a.Callback
					if member == nil {
						continue
					}
					referType(Referrer{entity, nil}, member.ReturnType, false)
					for i, param := range member.Parameters {
						referType(Referrer{entity, &member.Parameters[i]}, param.Type, false)
					}
				}
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
			if eclass.ReferrerList[i].Member.ID[0] == eclass.ReferrerList[j].Member.ID[0] {
				return eclass.ReferrerList[i].Member.ID[1] < eclass.ReferrerList[j].Member.ID[1]
			}
			return eclass.ReferrerList[i].Member.ID[0] < eclass.ReferrerList[j].Member.ID[0]
		})
	}

	for _, eenum := range entities.Enums {
		sort.Slice(eenum.ReferrerList, func(i, j int) bool {
			if eenum.ReferrerList[i].Member.ID[0] == eenum.ReferrerList[j].Member.ID[0] {
				return eenum.ReferrerList[i].Member.ID[1] < eenum.ReferrerList[j].Member.ID[1]
			}
			return eenum.ReferrerList[i].Member.ID[0] < eenum.ReferrerList[j].Member.ID[0]
		})
	}

	for _, etype := range entities.Types {
		sort.Slice(etype.ReferrerList, func(i, j int) bool {
			if etype.ReferrerList[i].Member.ID[0] == etype.ReferrerList[j].Member.ID[0] {
				return etype.ReferrerList[i].Member.ID[1] < etype.ReferrerList[j].Member.ID[1]
			}
			return etype.ReferrerList[i].Member.ID[0] < etype.ReferrerList[j].Member.ID[0]
		})
		sort.Slice(etype.RemovedRefList, func(i, j int) bool {
			if etype.RemovedRefList[i].Member.ID[0] == etype.RemovedRefList[j].Member.ID[0] {
				return etype.RemovedRefList[i].Member.ID[1] < etype.RemovedRefList[j].Member.ID[1]
			}
			return etype.RemovedRefList[i].Member.ID[0] < etype.RemovedRefList[j].Member.ID[0]
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
				if eenum.ItemList[i].Element.Value == eenum.ItemList[j].Element.Value {
					return eenum.ItemList[i].Element.Name < eenum.ItemList[j].Element.Name
				}
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

	for id, eclass := range entities.Classes {
		super := eclass.Element.Superclass
		if !eclass.Removed {
			if s := entities.Classes[super]; s == nil || s.Removed {
				entities.TreeRoots = append(entities.TreeRoots, eclass)
			}
		}
		for class := entities.Classes[super]; class != nil; class = entities.Classes[super] {
			if !class.Removed {
				eclass.Superclasses = append(eclass.Superclasses, class)
			}
			super = class.Element.Superclass
		}
		for _, sub := range entities.Classes {
			if sub.Element.Superclass == id && !sub.Removed {
				eclass.Subclasses = append(eclass.Subclasses, sub)
			}
		}
		sort.Slice(eclass.Subclasses, func(i, j int) bool {
			return eclass.Subclasses[i].ID < eclass.Subclasses[j].ID
		})
	}
	sort.Slice(entities.TreeRoots, func(i, j int) bool {
		return entities.TreeRoots[i].ID < entities.TreeRoots[j].ID
	})

	return entities
}
