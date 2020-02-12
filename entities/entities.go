package entities

import (
	"fmt"
	"html/template"
	"sort"
	"strconv"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/patch"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"github.com/robloxapi/rbxapiref/builds"
	"github.com/robloxapi/rbxapiref/documents"
	"github.com/robloxapi/rbxfile"
)

type Entities struct {
	Classes   map[string]*Class
	ClassList []*Class
	Members   map[[2]string]*Member
	TreeRoots []*Class

	Enums     map[string]*Enum
	EnumList  []*Enum
	EnumItems map[[2]string]*EnumItem

	Types    map[string]*Type
	TypeList []*Type
	TypeCats []TypeCategory

	Coverage float32
}

func (e *Entities) CoverageString() string {
	return fmt.Sprintf("%.2f%%", e.Coverage*100)
}

// Sorted by most to least permissive.
var securityContexts = map[string]int{
	"None":                  0,
	"RobloxPlaceSecurity":   1,
	"PluginSecurity":        2,
	"LocalUserSecurity":     3,
	"RobloxScriptSecurity":  4,
	"RobloxSecurity":        5,
	"NotAccessibleSecurity": 6,
}

func (e *Entities) ElementStatusClasses(suffix bool, v ...interface{}) string {
	var t rbxapi.Taggable
	var action *builds.Action
	var removed bool
	switch value := v[0].(type) {
	case rbxapi.Taggable:
		t = value
	case string:
		switch value {
		case "class":
			class, ok := e.Classes[v[1].(string)]
			if !ok {
				return ""
			}
			removed = class.Removed
			t = class.Element
		case "member":
			class, ok := e.Classes[v[1].(string)]
			if !ok {
				return ""
			}
			member, ok := class.Members[v[2].(string)]
			if !ok {
				return ""
			}
			removed = member.Removed
			t = member.Element
		case "enum":
			enum, ok := e.Enums[v[1].(string)]
			if !ok {
				return ""
			}
			removed = enum.Removed
			t = enum.Element
		case "enumitem":
			enum, ok := e.Enums[v[1].(string)]
			if !ok {
				return ""
			}
			item, ok := enum.Items[v[2].(string)]
			if !ok {
				return ""
			}
			removed = item.Removed
			t = item.Element
		default:
			return ""
		}
	case *Class:
		removed = value.Removed
		t = value.Element
	case *Member:
		removed = value.Removed
		t = value.Element
	case *Enum:
		removed = value.Removed
		t = value.Element
	case *EnumItem:
		removed = value.Removed
		t = value.Element
	case builds.Action:
		action = &value
		t, _ = value.GetElement().(rbxapi.Taggable)
	default:
		return ""
	}

	var s []string
	if action != nil {
		switch action.Type {
		case patch.Change:
			switch action.Field {
			case "Security", "ReadSecurity", "WriteSecurity":
				// Select the most permissive context. This will cause changes
				// to visible contexts to always be displayed.
				p, _ := action.GetPrev().(string)
				n, _ := action.GetNext().(string)
				if securityContexts[p] < securityContexts[n] {
					if p != "" && p != "None" {
						s = append(s, "api-sec-"+p)
					}
				} else {
					if n != "" && n != "None" {
						s = append(s, "api-sec-"+n)
					}
				}
				goto finish
			case "Tags":
				// Include tag unless it is being changed.
				p := rbxapijson.Tags(action.GetPrev().([]string))
				n := rbxapijson.Tags(action.GetNext().([]string))
				if p.GetTag("Deprecated") && n.GetTag("Deprecated") {
					s = append(s, "api-deprecated")
				}
				if p.GetTag("NotBrowsable") && n.GetTag("NotBrowsable") {
					s = append(s, "api-not-browsable")
				}
				if p.GetTag("Hidden") && n.GetTag("Hidden") {
					s = append(s, "api-hidden")
				}
				goto finish
			}
		}
	}

	for _, tag := range t.GetTags() {
		switch tag {
		case "Deprecated":
			s = append(s, "api-deprecated")
		case "NotBrowsable":
			s = append(s, "api-not-browsable")
		case "Hidden":
			s = append(s, "api-hidden")
		}
	}
	if removed {
		s = append(s, "api-removed")
	}
	switch m := t.(type) {
	case interface{ GetSecurity() (string, string) }:
		r, w := m.GetSecurity()
		if r == w {
			if r != "" && r != "None" {
				s = append(s, "api-sec-"+r)
			}
		} else {
			s = append(s, "api-sec-"+r)
			s = append(s, "api-sec-"+w)
		}
	case interface{ GetSecurity() string }:
		if v := m.GetSecurity(); v != "" && v != "None" {
			s = append(s, "api-sec-"+v)
		}
	}

finish:
	if len(s) == 0 {
		return ""
	}

	sort.Strings(s)
	j := 0
	for i := 1; i < len(s); i++ {
		if s[j] != s[i] {
			j++
			s[j] = s[i]
		}
	}
	s = s[:j+1]

	classes := strings.Join(s, " ")
	if suffix {
		classes = " " + classes
	}
	return classes
}

var memberIconIndex = map[string]int{
	"Property": 6,
	"Function": 4,
	"Event":    11,
	"Callback": 16,
}

func (e *Entities) Icon(v ...interface{}) template.HTML {
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
			entity, ok := e.Classes[v[1].(string)]
			if !ok {
				goto finish
			}
			v = []interface{}{entity}
			goto retry
		case "member":
			entity := e.Members[[2]string{v[1].(string), v[2].(string)}]
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
	case *Class:
		class = "class-icon"
		title = "Class"
		if value.Metadata.Instance == nil {
			goto finish
		}
		index = value.Metadata.GetInt("ExplorerImageIndex")
	case *Member:
		if value.Element == nil {
			goto finish
		}
		v = []interface{}{value.Element}
		goto retry
	case *Enum:
		if value.Element == nil {
			goto finish
		}
		v = []interface{}{value.Element}
		goto retry
	case *EnumItem:
		if value.Element == nil {
			goto finish
		}
		v = []interface{}{value.Element}
		goto retry
	case TypeCategory:
		class = "member-icon"
		title = "TypeCategory"
		index = 0
	case *Type:
		class = "member-icon"
		title = "Type"
		index = 3
	case *rbxapijson.Class:
		entity, ok := e.Classes[value.Name]
		if !ok {
			goto finish
		}
		v = []interface{}{entity}
		goto retry
	case rbxapi.Member:
		class = "member-icon"
		title = value.GetMemberType()
		index = memberIconIndex[title]
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

type Class struct {
	ID      string
	Element *rbxapijson.Class
	Patches []builds.Patch
	Removed bool

	Superclasses []*Class
	Subclasses   []*Class

	Members    map[string]*Member
	MemberList []*Member

	References    map[rbxapijson.Type]ElementTyper
	ReferenceList []ElementTyper
	Referrers     map[[2]string]Referrer
	ReferrerList  []Referrer

	Document  Document
	DocStatus DocStatus
	Metadata  Metadata
}

func (e *Class) IsRemoved() bool    { return e.Removed }
func (e *Class) Identifier() string { return e.ID }
func (e *Class) ElementType() rbxapijson.Type {
	return rbxapijson.Type{Category: "Class", Name: e.Element.Name}
}
func (e *Class) GetDocument() Document   { return e.Document }
func (e *Class) GetDocStatus() DocStatus { return e.DocStatus }

type Member struct {
	ID      [2]string
	Element rbxapi.Member
	Patches []builds.Patch
	Removed bool

	Parent *Class

	References    map[rbxapijson.Type]ElementTyper
	ReferenceList []ElementTyper

	Document  Document
	DocStatus DocStatus
	Metadata  Metadata
}

func (e *Member) IsRemoved() bool         { return e.Removed }
func (e *Member) GetDocument() Document   { return e.Document }
func (e *Member) GetDocStatus() DocStatus { return e.DocStatus }

type Enum struct {
	ID      string
	Element *rbxapijson.Enum
	Patches []builds.Patch
	Removed bool

	Items    map[string]*EnumItem
	ItemList []*EnumItem

	Referrers    map[[2]string]Referrer
	ReferrerList []Referrer

	Document  Document
	DocStatus DocStatus
	Metadata  Metadata
}

func (e *Enum) IsRemoved() bool    { return e.Removed }
func (e *Enum) Identifier() string { return e.ID }
func (e *Enum) ElementType() rbxapijson.Type {
	return rbxapijson.Type{Category: "Enum", Name: e.Element.Name}
}
func (e *Enum) GetDocument() Document   { return e.Document }
func (e *Enum) GetDocStatus() DocStatus { return e.DocStatus }

type EnumItem struct {
	ID      [2]string
	Element *rbxapijson.EnumItem
	Patches []builds.Patch
	Removed bool

	Parent *Enum

	Document  Document
	DocStatus DocStatus
	Metadata  Metadata
}

func (e *EnumItem) IsRemoved() bool         { return e.Removed }
func (e *EnumItem) GetDocument() Document   { return e.Document }
func (e *EnumItem) GetDocStatus() DocStatus { return e.DocStatus }

type Type struct {
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

func (e *Type) IsRemoved() bool    { return e.Removed }
func (e *Type) Identifier() string { return e.ID }
func (e *Type) ElementType() rbxapijson.Type {
	return e.Element
}
func (e *Type) GetDocument() Document   { return e.Document }
func (e *Type) GetDocStatus() DocStatus { return e.DocStatus }

type TypeCategory struct {
	Name  string
	Types []*Type
}

type Referrer struct {
	Member    *Member
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
		eclass = &Class{
			ID:         id,
			Element:    class.Copy().(*rbxapijson.Class),
			Members:    map[string]*Member{},
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
				emember = &Member{
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
		emember = &Member{
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
		eenum = &Enum{
			ID:        id,
			Element:   enum.Copy().(*rbxapijson.Enum),
			Items:     map[string]*EnumItem{},
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
				eitem = &EnumItem{
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
		eitem = &EnumItem{
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

	var addClasses func(classes []*Class)
	addClasses = func(classes []*Class) {
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
		Classes:   make(map[string]*Class),
		Members:   make(map[[2]string]*Member),
		Enums:     make(map[string]*Enum),
		EnumItems: make(map[[2]string]*EnumItem),
		Types:     make(map[string]*Type),
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
				etype = &Type{
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
		entities.ClassList = make([]*Class, len(entities.Classes))
		i := 0
		for _, eclass := range entities.Classes {
			entities.ClassList[i] = eclass
			i++

			eclass.MemberList = make([]*Member, len(eclass.Members))
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
		entities.EnumList = make([]*Enum, len(entities.Enums))
		i := 0
		for _, eenum := range entities.Enums {
			entities.EnumList[i] = eenum
			i++

			eenum.ItemList = make([]*EnumItem, len(eenum.Items))
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

	entities.TypeList = make([]*Type, 0, len(entities.Types))
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
			Types: []*Type{etype},
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
