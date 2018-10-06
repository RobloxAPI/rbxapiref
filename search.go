package main

import (
	"encoding/binary"
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/rbxapijson"
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
	root := dw.data.Latest.API

	// Version
	if dw.writeNumber(byte(0)) {
		return true
	}

	// Class icon count
	if dw.writeNumber(uint16(len(root.Classes))) {
		return true
	}

	// Item count
	items := 0
	items += len(root.Classes)
	for _, class := range root.Classes {
		items += len(class.Members)
	}
	items += len(root.Enums)
	for _, enum := range root.Enums {
		items += len(enum.Items)
	}
	items += len(dw.data.Entities.TypeList)
	if dw.writeNumber(uint16(items)) {
		return true
	}

	// Class icons
	for _, class := range root.Classes {
		var icon int
		if m, ok := dw.data.Metadata.Classes[class.Name]; ok {
			icon = m.ExplorerImageIndex
		}
		if dw.writeNumber(uint8(icon)) {
			return true
		}
	}

	// Items
	for _, class := range root.Classes {
		if dw.writeItem(class) {
			return true
		}
	}
	for _, class := range root.Classes {
		for _, member := range class.Members {
			if dw.writeItem(member) {
				return true
			}
		}
	}
	for _, enum := range root.Enums {
		if dw.writeItem(enum) {
			return true
		}
	}
	for _, enum := range root.Enums {
		for _, item := range enum.Items {
			if dw.writeItem(item) {
				return true
			}
		}
	}
	for _, typ := range dw.data.Entities.TypeList {
		dw.writeItem(typ)
	}

	// Item strings
	for _, class := range root.Classes {
		if dw.writeString(class.Name) {
			return true
		}
	}
	for _, class := range root.Classes {
		for _, member := range class.Members {
			if dw.writeString(class.Name + "." + member.GetName()) {
				return true
			}
		}
	}
	for _, enum := range root.Enums {
		if dw.writeString(enum.Name) {
			return true
		}
	}
	for _, enum := range root.Enums {
		for _, item := range enum.Items {
			if dw.writeString(enum.Name + "." + item.Name) {
				return true
			}
		}
	}
	for _, typ := range dw.data.Entities.TypeList {
		if dw.writeString(typ.Element.Name) {
			return true
		}
	}

	return false
}
