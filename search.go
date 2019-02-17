package main

import (
	"encoding/binary"
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"github.com/robloxapi/rbxfile"
	"io"
)

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

func (dw *dbWriter) writeItem(v interface{}) bool {
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

	if v, ok := v.(rbxapi.Taggable); ok {
		setbit(&data, 4, v.GetTag("Deprecated"))
	}

	if v, ok := v.(interface{ GetSecurity() string }); ok {
		s := v.GetSecurity()
		setbit(&data, 5, s != "" && s != "None")
	} else if v, ok := v.(interface{ GetSecurity() (string, string) }); ok {
		r, w := v.GetSecurity()
		setbit(&data, 5, (r != "" && r != "None") || (w != "" && w != "None"))
	}

	return dw.writeNumber(uint8(data))
}

func (dw *dbWriter) GenerateDatabase() bool {
	// Version
	if dw.writeNumber(byte(0)) {
		return true
	}

	// Class icon count
	if dw.writeNumber(uint16(len(dw.data.Entities.ClassList))) {
		return true
	}

	// Item count
	items := 0
	items += len(dw.data.Entities.ClassList)
	for _, class := range dw.data.Entities.ClassList {
		items += len(class.MemberList)
	}
	items += len(dw.data.Entities.EnumList)
	for _, enum := range dw.data.Entities.EnumList {
		items += len(enum.ItemList)
	}
	items += len(dw.data.Entities.TypeList)
	if dw.writeNumber(uint16(items)) {
		return true
	}

	// Class icons
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
	for _, class := range dw.data.Entities.ClassList {
		if dw.writeItem(class.Element) {
			return true
		}
	}
	for _, class := range dw.data.Entities.ClassList {
		for _, member := range class.MemberList {
			if dw.writeItem(member.Element) {
				return true
			}
		}
	}
	for _, enum := range dw.data.Entities.EnumList {
		if dw.writeItem(enum.Element) {
			return true
		}
	}
	for _, enum := range dw.data.Entities.EnumList {
		for _, item := range enum.ItemList {
			if dw.writeItem(item.Element) {
				return true
			}
		}
	}
	for _, typ := range dw.data.Entities.TypeList {
		dw.writeItem(typ.Element)
	}

	// Item strings
	for _, class := range dw.data.Entities.ClassList {
		if dw.writeString(class.ID) {
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
		if dw.writeString(enum.ID) {
			return true
		}
	}
	for _, enum := range dw.data.Entities.EnumList {
		for _, item := range enum.ItemList {
			if dw.writeString(item.ID[0] + "." + item.ID[1]) {
				return true
			}
		}
	}
	for _, typ := range dw.data.Entities.TypeList {
		if dw.writeString(typ.ID) {
			return true
		}
	}

	return false
}
