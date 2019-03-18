package main

import (
	"encoding/binary"
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"github.com/robloxapi/rbxfile"
	"io"
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

// Writer wrapper that keeps track of the number of bytes written.
type dbWriter struct {
	data *Data
	w    io.Writer
	n    int64
	err  error
}

func (f *dbWriter) write(p []byte) (failed bool) {
	if f.err != nil {
		return true
	}

	var n int
	n, f.err = f.w.Write(p)
	f.n += int64(n)

	if n < len(p) {
		return true
	}

	return false
}

func (f *dbWriter) end() (n int64, err error) {
	return f.n, f.err
}

func (f *dbWriter) writeNumber(data interface{}) (failed bool) {
	if f.err != nil {
		return true
	}
	m := 0
	b := make([]byte, 8)
	switch data := data.(type) {
	case int8:
		m = 1
		b[0] = uint8(data)
	case uint8:
		m = 1
		b[0] = data
	case int16:
		m = 2
		binary.LittleEndian.PutUint16(b, uint16(data))
	case uint16:
		m = 2
		binary.LittleEndian.PutUint16(b, data)
	case int32:
		m = 4
		binary.LittleEndian.PutUint32(b, uint32(data))
	case uint32:
		m = 4
		binary.LittleEndian.PutUint32(b, data)
	case int64:
		m = 8
		binary.LittleEndian.PutUint64(b, uint64(data))
	case uint64:
		m = 8
		binary.LittleEndian.PutUint64(b, data)
	default:
		panic("invalid type")
	}
	return f.write(b[:m])
}

func (f *dbWriter) writeString(data string) (failed bool) {
	if f.err != nil {
		return true
	}

	if f.writeNumber(uint8(len(data))) {
		return true
	}

	return f.write([]byte(data))
}

func setbit(p *uint64, a int, v bool) {
	if v {
		*p |= 1 << uint(a)
		return
	}
	*p &= ^(1 << uint(a))
}
func setbits(p *uint64, a, b, v int) {
	var m uint64 = 1<<uint(b-a) - 1
	*p = *p&^(m<<uint(a)) | (uint64(v)&m)<<uint(a)
}
func getbit(p uint64, a int) bool {
	return (p>>uint(a))&1 == 1
}
func getbits(p uint64, a, b int) int {
	return int((p >> uint(a)) & (1<<uint(b-a) - 1))
}

func (dw *dbWriter) writeSecurity(data *uint64, i int, security string) {
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
	setbits(data, i, i+3, sec)
}

func (dw *dbWriter) writeItem(v interface{}, removed bool) bool {
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
	setbits(&data, 0, 3, typ)
	setbit(&data, 3, removed)

	if v, ok := v.(rbxapi.Taggable); ok {
		setbit(&data, 4, v.GetTag("Deprecated"))
		setbit(&data, 5, v.GetTag("NotBrowsable"))
		switch typ {
		case 0: // Class
			setbit(&data, 6, v.GetTag("NotCreatable"))
		case 4: // Property
			setbit(&data, 6, v.GetTag("Hidden"))
		}
	}

	if v, ok := v.(interface{ GetSecurity() string }); ok {
		dw.writeSecurity(&data, 8, v.GetSecurity())
	} else if v, ok := v.(interface{ GetSecurity() (string, string) }); ok {
		r, w := v.GetSecurity()
		dw.writeSecurity(&data, 8, r)
		dw.writeSecurity(&data, 11, w)
	}

	return dw.writeNumber(uint16(data))
}

func (dw *dbWriter) GenerateDatabase() bool {
	// Version
	if dw.writeNumber(uint8(1)) {
		return true
	}

	// IconCount
	if dw.writeNumber(uint16(len(dw.data.Entities.ClassList))) {
		return true
	}

	items := 0
	items += len(dw.data.Entities.TypeList)
	// ClassOffset
	if dw.writeNumber(uint16(items)) {
		return true
	}
	items += len(dw.data.Entities.ClassList)
	items += len(dw.data.Entities.EnumList)
	for _, class := range dw.data.Entities.ClassList {
		items += len(class.MemberList)
	}
	for _, enum := range dw.data.Entities.EnumList {
		items += len(enum.ItemList)
	}
	// ItemCount
	if dw.writeNumber(uint16(items)) {
		return true
	}

	// Icons
	for _, class := range dw.data.Entities.ClassList {
		var icon int
		if class.Metadata.Instance != nil {
			i, _ := class.Metadata.Get("ExplorerImageIndex").(rbxfile.ValueInt)
			icon = int(i)
		}
		if dw.writeNumber(uint8(icon)) {
			return true
		}
	}

	// Items
	for _, typ := range dw.data.Entities.TypeList {
		if dw.writeItem(typ.Element, typ.Removed) {
			return true
		}
	}
	for _, class := range dw.data.Entities.ClassList {
		if dw.writeItem(class.Element, class.Removed) {
			return true
		}
	}
	for _, enum := range dw.data.Entities.EnumList {
		if dw.writeItem(enum.Element, enum.Removed) {
			return true
		}
	}
	for _, class := range dw.data.Entities.ClassList {
		for _, member := range class.MemberList {
			if dw.writeItem(member.Element, member.Removed) {
				return true
			}
		}
	}
	for _, enum := range dw.data.Entities.EnumList {
		for _, item := range enum.ItemList {
			if dw.writeItem(item.Element, item.Removed) {
				return true
			}
		}
	}

	// Strings
	for _, typ := range dw.data.Entities.TypeList {
		if dw.writeString(typ.ID) {
			return true
		}
	}
	for _, class := range dw.data.Entities.ClassList {
		if dw.writeString(class.ID) {
			return true
		}
	}
	for _, enum := range dw.data.Entities.EnumList {
		if dw.writeString(enum.ID) {
			return true
		}
	}
	for _, class := range dw.data.Entities.ClassList {
		for _, member := range class.MemberList {
			if dw.writeString(member.ID[0] + "." + member.ID[1]) {
				return true
			}
		}
	}
	for _, enum := range dw.data.Entities.EnumList {
		for _, item := range enum.ItemList {
			if dw.writeString(item.ID[0] + "." + item.ID[1]) {
				return true
			}
		}
	}

	return false
}
