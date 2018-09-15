package main

import (
	"fmt"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"sort"
)

func GenerateRefPage(data *Data) error {
	stalePageSet := map[string]bool{}
	findType := func(v interface{}, stale bool) {
		switch v := v.(type) {
		case rbxapijson.Type:
			stalePageSet[data.FileLink("type", v.Category, v.Name)] = stale
		case []rbxapijson.Parameter:
			for _, p := range v {
				stalePageSet[data.FileLink("type", p.Type.Category, p.Type.Name)] = stale
			}
		}
	}
	for _, patch := range data.Patches {
		if !patch.Stale {
			continue
		}
		for _, action := range patch.Actions {
			switch {
			case action.Class != nil:
				stalePageSet[data.FileLink("class", action.Class.Name)] = patch.Stale
				stalePageSet[data.FileLink("class", action.Class.Superclass)] = patch.Stale
				for _, member := range action.Class.Members {
					switch member.GetMemberType() {
					case "Property":
						member := member.(*rbxapijson.Property)
						findType(member.ValueType, patch.Stale)
					case "Function":
						member := member.(*rbxapijson.Function)
						findType(member.ReturnType, patch.Stale)
						findType(member.Parameters, patch.Stale)
					case "Event":
						member := member.(*rbxapijson.Event)
						findType(member.Parameters, patch.Stale)
					case "Callback":
						member := member.(*rbxapijson.Callback)
						findType(member.ReturnType, patch.Stale)
						findType(member.Parameters, patch.Stale)
					}
				}
			case action.Enum != nil:
				stalePageSet[data.FileLink("enum", action.Enum.Name)] = patch.Stale
			}
			findType(action.GetPrev(), patch.Stale)
			findType(action.GetNext(), patch.Stale)
		}
	}
	stalePages := make([]string, 0, len(stalePageSet))
	for k := range stalePageSet {
		stalePages = append(stalePages, k)
	}
	sort.Strings(stalePages)
	for _, page := range stalePages {
		fmt.Println("PAGE", stalePageSet[page], page)
	}
	return nil
}
