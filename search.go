package main

import (
	"io"

	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"github.com/robloxapi/rbxapiref/internal/binio"
)

/*
// Search Database Format

main struct {
	// Database version.
	Version   uint:8 = 1
	// Number of icons.
	IconCount uint:16
	// Starting index of items that are classes. Subtracted from item index to
	// retrieve icon index.
	ClassOffset uint:16
	// Total number of items.
	ItemCount uint:16
	// List of ExplorerImageIndex for each class. Index corresponds to
	// Items[index - ClassOffset].
	Icons [.IconCount]uint:8
	// List of items.
	Items [.ItemCount]Item
	// List of item strings. Index corresponds to index of Items.
	Strings [.ItemCount]String
}

String struct {
	Size  uint:8
	Value [.Size]uint:8
}

ItemType enum uint:3 {
	Class
	Enum
	EnumItem
	Type
	Property
	Function
	Event
	Callback
}

Security enum uint:3 {
	None
	RobloxPlaceSecurity
	PluginSecurity
	LocalUserSecurity
	RobloxScriptSecurity
	RobloxSecurity
	NotAccessibleSecurity
}

Item struct {
	Type        ItemType
	Removed     bool:1
	Deprecated  bool:1
	Unbrowsable bool:1
	if .Type == Class {
		Uncreatable bool:1
	}
	if .Type == Property {
		Hidden bool:1
		@8
		ReadSecurity  Security
		WriteSecurity Security
	}
	if .Type > Property {
		@8
		Security Security
	}
	@16
}

*/

func writeDatabaseSecurity(data uint64, i int, security string) uint64 {
	var sec int
	switch security {
	case "RobloxPlaceSecurity":
		sec = 1
	case "PluginSecurity":
		sec = 2
	case "LocalUserSecurity":
		sec = 3
	case "RobloxScriptSecurity":
		sec = 4
	case "RobloxSecurity":
		sec = 5
	case "NotAccessibleSecurity":
		sec = 6
	}
	return binio.SetBits(data, i, i+3, sec)
}

func writeDatabaseItem(v interface{}, removed bool) uint16 {
	var data uint64

	var typ int
	switch v.(type) {
	case *rbxapijson.Class:
		typ = 0
	case *rbxapijson.Enum:
		typ = 1
	case *rbxapijson.EnumItem:
		typ = 2
	case rbxapijson.Type:
		typ = 3
	case *rbxapijson.Property:
		typ = 4
	case *rbxapijson.Function:
		typ = 5
	case *rbxapijson.Event:
		typ = 6
	case *rbxapijson.Callback:
		typ = 7
	}
	data = binio.SetBits(data, 0, 3, typ)
	data = binio.SetBit(data, 3, removed)

	if v, ok := v.(rbxapi.Taggable); ok {
		data = binio.SetBit(data, 4, v.GetTag("Deprecated"))
		data = binio.SetBit(data, 5, v.GetTag("NotBrowsable"))
		switch typ {
		case 0: // Class
			data = binio.SetBit(data, 6, v.GetTag("NotCreatable"))
		case 4: // Property
			data = binio.SetBit(data, 6, v.GetTag("Hidden"))
		}
	}

	if v, ok := v.(interface{ GetSecurity() string }); ok {
		data = writeDatabaseSecurity(data, 8, v.GetSecurity())
	} else if v, ok := v.(interface{ GetSecurity() (string, string) }); ok {
		r, w := v.GetSecurity()
		data = writeDatabaseSecurity(data, 8, r)
		data = writeDatabaseSecurity(data, 11, w)
	}

	return uint16(data)
}

func GenerateDatabase(w io.Writer, ent *Entities) error {
	bw := binio.NewWriter(w)

	// Version
	if !bw.Number(uint8(1)) {
		return bw.Err
	}

	// IconCount
	if !bw.Number(uint16(len(ent.ClassList))) {
		return bw.Err
	}

	items := 0
	items += len(ent.TypeList)
	// ClassOffset
	if !bw.Number(uint16(items)) {
		return bw.Err
	}
	items += len(ent.ClassList)
	items += len(ent.EnumList)
	for _, class := range ent.ClassList {
		items += len(class.MemberList)
	}
	for _, enum := range ent.EnumList {
		items += len(enum.ItemList)
	}
	// ItemCount
	if !bw.Number(uint16(items)) {
		return bw.Err
	}

	// Icons
	for _, class := range ent.ClassList {
		var icon int
		if class.Metadata.Instance != nil {
			icon = GetMetadataInt(class.Metadata, "ExplorerImageIndex")
		}
		if !bw.Number(uint8(icon)) {
			return bw.Err
		}
	}

	// Items
	for _, typ := range ent.TypeList {
		if !bw.Number(writeDatabaseItem(typ.Element, typ.Removed)) {
			return bw.Err
		}
	}
	for _, class := range ent.ClassList {
		if !bw.Number(writeDatabaseItem(class.Element, class.Removed)) {
			return bw.Err
		}
	}
	for _, enum := range ent.EnumList {
		if !bw.Number(writeDatabaseItem(enum.Element, enum.Removed)) {
			return bw.Err
		}
	}
	for _, class := range ent.ClassList {
		for _, member := range class.MemberList {
			if !bw.Number(writeDatabaseItem(member.Element, member.Removed)) {
				return bw.Err
			}
		}
	}
	for _, enum := range ent.EnumList {
		for _, item := range enum.ItemList {
			if !bw.Number(writeDatabaseItem(item.Element, item.Removed)) {
				return bw.Err
			}
		}
	}

	// Strings
	for _, typ := range ent.TypeList {
		if !bw.String(typ.ID) {
			return bw.Err
		}
	}
	for _, class := range ent.ClassList {
		if !bw.String(class.ID) {
			return bw.Err
		}
	}
	for _, enum := range ent.EnumList {
		if !bw.String(enum.ID) {
			return bw.Err
		}
	}
	for _, class := range ent.ClassList {
		for _, member := range class.MemberList {
			if !bw.String(member.ID[0] + "." + member.ID[1]) {
				return bw.Err
			}
		}
	}
	for _, enum := range ent.EnumList {
		for _, item := range enum.ItemList {
			if !bw.String(item.ID[0] + "." + item.ID[1]) {
				return bw.Err
			}
		}
	}

	return nil
}
