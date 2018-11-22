package main

import (
	"github.com/pkg/errors"
	"github.com/robloxapi/rbxapiref/fetch"
	"github.com/robloxapi/rbxfile"
	"strconv"
)

type ReflectionMetadata struct {
	Classes map[string]ClassMetadata
	Enums   map[string]EnumMetadata
}

type ItemMetadata struct {
	Name string
	// Browsable       bool
	// ClassCategory   string
	// Constraint      string
	// Deprecated      bool
	// EditingDisabled bool
	// IsBackend       bool
	// ScriptContext   string
	// UIMaximum       float64
	// UIMinimum       float64
	// UINumTicks      float64
	// Summary         string
}

type ClassMetadata struct {
	ItemMetadata
	ExplorerImageIndex int
	// ExplorerOrder      int
	// Insertable         bool
	// PreferredParent    string
	// PreferredParents   string
}

type EnumMetadata struct {
	ItemMetadata
}

func getMetadataValue(p interface{}, v rbxfile.Value) {
	switch p := p.(type) {
	case *int:
		switch v := v.(type) {
		case rbxfile.ValueInt:
			*p = int(v)
		case rbxfile.ValueString:
			*p, _ = strconv.Atoi(string(v))
		}
	}
}

func GenerateMetadata(client *fetch.Client, hash string) (metadata ReflectionMetadata, err error) {
	rmd, err := client.ReflectionMetadata(hash)
	if err != nil {
		return metadata, errors.WithMessagef(err, "fetch metadata %s:", hash)
	}

	metadata.Classes = make(map[string]ClassMetadata)
	metadata.Enums = make(map[string]EnumMetadata)
	for _, list := range rmd.Instances {
		switch list.ClassName {
		case "ReflectionMetadataClasses":
			for _, class := range list.Children {
				if class.ClassName != "ReflectionMetadataClass" {
					continue
				}
				meta := ClassMetadata{ItemMetadata: ItemMetadata{Name: class.Name()}}
				getMetadataValue(&meta.ExplorerImageIndex, class.Properties["ExplorerImageIndex"])
				metadata.Classes[meta.Name] = meta
			}
		case "ReflectionMetadataEnums":
		}
	}
	return metadata, nil
}
