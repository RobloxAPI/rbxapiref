package main

import (
	"encoding/json"
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/patch"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"reflect"
)

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
func (a *Action) GetClass() rbxapi.Class {
	if a.Class == nil {
		return nil
	}
	return a.Class
}
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
func (a *Action) GetEnum() rbxapi.Enum {
	if a.Enum == nil {
		return nil
	}
	return a.Enum
}
func (a *Action) GetEnumItem() rbxapi.EnumItem {
	if a.EnumItem == nil {
		return nil
	}
	return a.EnumItem
}
func (a *Action) GetType() patch.Type { return a.Type }
func (a *Action) GetField() string    { return a.Field }
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
	case rbxapijson.Type, rbxapijson.Parameters:
		w.V = v
	case rbxapi.Type:
		w.V = rbxapijson.Type{
			Category: v.GetCategory(),
			Name:     v.GetName(),
		}
	case rbxapi.Parameters:
		n := v.GetLength()
		params := make([]rbxapijson.Parameter, n)
		for i := 0; i < n; i++ {
			p := v.GetParameter(i)
			params[i] = rbxapijson.Parameter{
				Type: rbxapijson.Type{
					Category: p.GetType().GetCategory(),
					Name:     p.GetType().GetName(),
				},
				Name: p.GetName(),
			}
			params[i].Default, params[i].HasDefault = p.GetDefault()
		}
		w.V = rbxapijson.Parameters{List: &params}
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
	case rbxapijson.Parameters:
		w.Type = "Parameters"
		w.Value = v
	default:
		panic("unknown action value type " + reflect.TypeOf(v).String())
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
		var value struct{ Value rbxapijson.Parameters }
		if err = json.Unmarshal(b, &value); err != nil {
			return err
		}
		v.V = value.Value
	}
	return nil
}
