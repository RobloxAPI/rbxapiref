package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/patch"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"io"
	"io/ioutil"
)

// Reader wrapper that keeps track of the number of bytes written.
type binaryReader struct {
	r   io.Reader
	buf []byte
	n   int64
	err error
}

func (f *binaryReader) Buf(n int) []byte {
	if cap(f.buf) < n {
		f.buf = make([]byte, n)
	}
	return f.buf[:n]
}

func (f *binaryReader) End() (n int64, err error) {
	return f.n, f.err
}

func (f *binaryReader) Bytes(p []byte) (failed bool) {
	if f.err != nil {
		return true
	}

	var n int
	n, f.err = io.ReadFull(f.r, p)
	f.n += int64(n)

	if f.err != nil {
		return true
	}

	return false
}

func (f *binaryReader) ReadAll() (data []byte, failed bool) {
	if f.err != nil {
		return nil, true
	}

	data, f.err = ioutil.ReadAll(f.r)
	f.n += int64(len(data))

	if f.err != nil {
		return nil, true
	}

	return data, false
}

func (f *binaryReader) Number(data interface{}) (failed bool) {
	if f.err != nil {
		return true
	}

	var b []byte
	switch data.(type) {
	case *int8, *uint8:
		b = f.Buf(1)
	case *int16, *uint16:
		b = f.Buf(2)
	case *int32, *uint32:
		b = f.Buf(4)
	case *int64, *uint64:
		b = f.Buf(8)
	default:
		goto invalid
	}

	if f.Bytes(b) {
		return true
	}
	switch data := data.(type) {
	case *int8:
		*data = int8(b[0])
	case *uint8:
		*data = b[0]
	case *int16:
		*data = int16(binary.LittleEndian.Uint16(b))
	case *uint16:
		*data = binary.LittleEndian.Uint16(b)
	case *int32:
		*data = int32(binary.LittleEndian.Uint32(b))
	case *uint32:
		*data = binary.LittleEndian.Uint32(b)
	case *int64:
		*data = int64(binary.LittleEndian.Uint64(b))
	case *uint64:
		*data = binary.LittleEndian.Uint64(b)
	default:
		goto invalid
	}
	return false
invalid:
	panic("invalid type")
}

func (f *binaryReader) String(data *string) (failed bool) {
	if f.err != nil {
		return true
	}

	var length uint8
	if f.Number(&length) {
		return true
	}

	s := f.Buf(int(length))
	if f.Bytes(s) {
		return true
	}

	*data = string(s)

	return false
}

// Writer wrapper that keeps track of the number of bytes written.
type binaryWriter struct {
	w   io.Writer
	buf []byte
	n   int64
	err error
}

func (f *binaryWriter) Buf(n int) []byte {
	if cap(f.buf) < n {
		f.buf = make([]byte, n)
	}
	return f.buf[:n]
}

func (f *binaryWriter) End() (n int64, err error) {
	return f.n, f.err
}

func (f *binaryWriter) Bytes(p []byte) (failed bool) {
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

func (f *binaryWriter) Number(data interface{}) (failed bool) {
	if f.err != nil {
		return true
	}

	var b []byte
	switch data.(type) {
	case int8, uint8:
		b = f.Buf(1)
	case int16, uint16:
		b = f.Buf(2)
	case int32, uint32:
		b = f.Buf(4)
	case int64, uint64:
		b = f.Buf(8)
	default:
		goto invalid
	}

	switch data := data.(type) {
	case int8:
		b[0] = uint8(data)
	case uint8:
		b[0] = data
	case int16:
		binary.LittleEndian.PutUint16(b, uint16(data))
	case uint16:
		binary.LittleEndian.PutUint16(b, data)
	case int32:
		binary.LittleEndian.PutUint32(b, uint32(data))
	case uint32:
		binary.LittleEndian.PutUint32(b, data)
	case int64:
		binary.LittleEndian.PutUint64(b, uint64(data))
	case uint64:
		binary.LittleEndian.PutUint64(b, data)
	default:
		goto invalid
	}
	return f.Bytes(b)

invalid:
	panic("invalid type")
}

func (f *binaryWriter) String(data string) (failed bool) {
	if f.err != nil {
		return true
	}

	if len(data) >= 1<<8 {
		panic("string too large")
	}
	if f.Number(uint8(len(data))) {
		return true
	}

	return f.Bytes([]byte(data))
}

type Manifest struct {
	Patches []Patch
}

func (man *Manifest) ReadFrom(r io.Reader) (n int64, err error) {
	br := &binaryReader{r: r, buf: make([]byte, 256)}
	var length uint32
	br.Number(&length)
	man.Patches = make([]Patch, length)
	for i, patch := range man.Patches {
		man.readPatch(br, &patch)
		man.Patches[i] = patch
		if br.err != nil {
			break
		}
	}
	return br.End()
}

func (man *Manifest) WriteTo(w io.Writer) (n int64, err error) {
	bw := &binaryWriter{w: w, buf: make([]byte, 256)}
	bw.Number(uint32(len(man.Patches)))
	for _, patch := range man.Patches {
		man.writePatch(bw, &patch)
		if bw.err != nil {
			break
		}
	}
	return bw.End()
}

func (man *Manifest) readPatch(br *binaryReader, patch *Patch) {
	man.readBuildInfo(br, &patch.Info)
	var b uint8
	br.Number(&b)
	if b != 0 {
		patch.Prev = &BuildInfo{}
		man.readBuildInfo(br, patch.Prev)
	}
	br.String(&patch.Config)
	var length uint32
	br.Number(&length)
	patch.Actions = make([]Action, length)
	for i, action := range patch.Actions {
		man.readAction(br, &action)
		patch.Actions[i] = action
		if br.err != nil {
			return
		}
	}
}

func (man *Manifest) writePatch(bw *binaryWriter, patch *Patch) {
	man.writeBuildInfo(bw, &patch.Info)
	if patch.Prev != nil {
		bw.Number(uint8(1))
		man.writeBuildInfo(bw, patch.Prev)
	} else {
		bw.Number(uint8(0))
	}
	bw.String(patch.Config)
	bw.Number(uint32(len(patch.Actions)))
	for _, action := range patch.Actions {
		man.writeAction(bw, &action)
		if bw.err != nil {
			return
		}
	}
}

func (man *Manifest) readBuildInfo(br *binaryReader, info *BuildInfo) {
	br.String(&info.Hash)
	var date string
	br.String(&date)
	if err := info.Date.UnmarshalBinary([]byte(date)); err != nil {
		br.err = err
		return
	}
	var v uint32
	br.Number(&v)
	info.Version.Major = int(v)
	br.Number(&v)
	info.Version.Minor = int(v)
	br.Number(&v)
	info.Version.Maint = int(v)
	br.Number(&v)
	info.Version.Build = int(v)
}

func (man *Manifest) writeBuildInfo(bw *binaryWriter, info *BuildInfo) {
	bw.String(info.Hash)
	date, err := info.Date.MarshalBinary()
	if err != nil {
		bw.err = err
		return
	}
	bw.String(string(date))
	bw.Number(uint32(info.Version.Major))
	bw.Number(uint32(info.Version.Minor))
	bw.Number(uint32(info.Version.Maint))
	bw.Number(uint32(info.Version.Build))
}

func (man *Manifest) readAction(br *binaryReader, action *Action) {
	var data uint8
	br.Number(&data)
	action.Type = patch.Type(getbits(uint64(data), 0, 2) - 1)
	switch getbits(uint64(data), 2, 5) {
	case 1:
		man.readClass(br, &action.Class)
		man.readProperty(br, &action.Property)
	case 2:
		man.readClass(br, &action.Class)
		man.readFunction(br, &action.Function)
	case 3:
		man.readClass(br, &action.Class)
		man.readEvent(br, &action.Event)
	case 4:
		man.readClass(br, &action.Class)
		man.readCallback(br, &action.Callback)
	case 5:
		man.readClass(br, &action.Class)
	case 6:
		man.readEnum(br, &action.Enum)
		man.readEnumItem(br, &action.EnumItem)
	case 7:
		man.readEnum(br, &action.Enum)
	default:
		br.err = errors.New("invalid action")
		return
	}
	if action.Type == patch.Change {
		br.String(&action.Field)
		man.readValue(br, &action.Prev)
		man.readValue(br, &action.Next)
	}
}

func (man *Manifest) writeAction(bw *binaryWriter, action *Action) {
	var data uint64
	setbits(&data, 0, 2, int(action.Type)+1)
	switch {
	case action.Property != nil:
		setbits(&data, 2, 5, 1)
		bw.Number(uint8(data))
		man.writeClass(bw, action.Class)
		man.writeProperty(bw, action.Property)
	case action.Function != nil:
		setbits(&data, 2, 5, 2)
		bw.Number(uint8(data))
		man.writeClass(bw, action.Class)
		man.writeFunction(bw, action.Function)
	case action.Event != nil:
		setbits(&data, 2, 5, 3)
		bw.Number(uint8(data))
		man.writeClass(bw, action.Class)
		man.writeEvent(bw, action.Event)
	case action.Callback != nil:
		setbits(&data, 2, 5, 4)
		bw.Number(uint8(data))
		man.writeClass(bw, action.Class)
		man.writeCallback(bw, action.Callback)
	case action.Class != nil:
		setbits(&data, 2, 5, 5)
		bw.Number(uint8(data))
		man.writeClass(bw, action.Class)
	case action.EnumItem != nil:
		setbits(&data, 2, 5, 6)
		bw.Number(uint8(data))
		man.writeEnum(bw, action.Enum)
		man.writeEnumItem(bw, action.EnumItem)
	case action.Enum != nil:
		setbits(&data, 2, 5, 7)
		bw.Number(uint8(data))
		man.writeEnum(bw, action.Enum)
	default:
		bw.err = errors.New("invalid action")
		return
	}
	if getbits(data, 0, 2) == 1 {
		bw.String(action.Field)
		man.writeValue(bw, action.Prev)
		man.writeValue(bw, action.Next)
	}
}

func (man *Manifest) readClass(br *binaryReader, p **rbxapijson.Class) {
	class := rbxapijson.Class{}
	br.String(&class.Name)
	br.String(&class.Superclass)
	br.String(&class.MemoryCategory)
	var length uint32
	br.Number(&length)
	class.Members = make([]rbxapi.Member, length)
	for i := range class.Members {
		var memberType uint8
		br.Number(&memberType)
		switch memberType {
		case 0:
			var member *rbxapijson.Property
			man.readProperty(br, &member)
			class.Members[i] = member
		case 1:
			var member *rbxapijson.Function
			man.readFunction(br, &member)
			class.Members[i] = member
		case 2:
			var member *rbxapijson.Event
			man.readEvent(br, &member)
			class.Members[i] = member
		case 3:
			var member *rbxapijson.Callback
			man.readCallback(br, &member)
			class.Members[i] = member
		default:
			br.err = errors.New("invalid member type")
		}
		if br.err != nil {
			return
		}
	}
	var tags []string
	man.readTags(br, &tags)
	class.Tags = rbxapijson.Tags(tags)
	*p = &class
}

func (man *Manifest) writeClass(bw *binaryWriter, class *rbxapijson.Class) {
	bw.String(class.Name)
	bw.String(class.Superclass)
	bw.String(class.MemoryCategory)
	bw.Number(uint32(len(class.Members)))
	for _, member := range class.Members {
		switch member := member.(type) {
		case *rbxapijson.Property:
			bw.Number(uint8(0))
			man.writeProperty(bw, member)
		case *rbxapijson.Function:
			bw.Number(uint8(1))
			man.writeFunction(bw, member)
		case *rbxapijson.Event:
			bw.Number(uint8(2))
			man.writeEvent(bw, member)
		case *rbxapijson.Callback:
			bw.Number(uint8(3))
			man.writeCallback(bw, member)
		}
	}
	man.writeTags(bw, []string(class.Tags))
}

func (man *Manifest) readProperty(br *binaryReader, p **rbxapijson.Property) {
	member := rbxapijson.Property{}
	br.String(&member.Name)
	br.String(&member.ValueType.Category)
	br.String(&member.ValueType.Name)
	br.String(&member.Category)
	br.String(&member.ReadSecurity)
	br.String(&member.WriteSecurity)
	var ser uint8
	br.Number(&ser)
	member.CanLoad = getbit(uint64(ser), 0)
	member.CanSave = getbit(uint64(ser), 1)
	var tags []string
	man.readTags(br, &tags)
	member.Tags = rbxapijson.Tags(tags)
	*p = &member
}

func (man *Manifest) writeProperty(bw *binaryWriter, member *rbxapijson.Property) {
	bw.String(member.Name)
	bw.String(member.ValueType.Category)
	bw.String(member.ValueType.Name)
	bw.String(member.Category)
	bw.String(member.ReadSecurity)
	bw.String(member.WriteSecurity)
	var ser uint64
	setbit(&ser, 0, member.CanLoad)
	setbit(&ser, 1, member.CanSave)
	bw.Number(uint8(ser))
	man.writeTags(bw, []string(member.Tags))
}

func (man *Manifest) readFunction(br *binaryReader, p **rbxapijson.Function) {
	member := rbxapijson.Function{}
	br.String(&member.Name)
	man.readParameters(br, &member.Parameters)
	br.String(&member.ReturnType.Category)
	br.String(&member.ReturnType.Name)
	br.String(&member.Security)
	var tags []string
	man.readTags(br, &tags)
	member.Tags = rbxapijson.Tags(tags)
	*p = &member
}

func (man *Manifest) writeFunction(bw *binaryWriter, member *rbxapijson.Function) {
	bw.String(member.Name)
	man.writeParameters(bw, member.Parameters)
	bw.String(member.ReturnType.Category)
	bw.String(member.ReturnType.Name)
	bw.String(member.Security)
	man.writeTags(bw, []string(member.Tags))
}

func (man *Manifest) readEvent(br *binaryReader, p **rbxapijson.Event) {
	member := rbxapijson.Event{}
	br.String(&member.Name)
	man.readParameters(br, &member.Parameters)
	br.String(&member.Security)
	var tags []string
	man.readTags(br, &tags)
	member.Tags = rbxapijson.Tags(tags)
	*p = &member
}

func (man *Manifest) writeEvent(bw *binaryWriter, member *rbxapijson.Event) {
	bw.String(member.Name)
	man.writeParameters(bw, member.Parameters)
	bw.String(member.Security)
	man.writeTags(bw, []string(member.Tags))
}

func (man *Manifest) readCallback(br *binaryReader, p **rbxapijson.Callback) {
	member := rbxapijson.Callback{}
	br.String(&member.Name)
	man.readParameters(br, &member.Parameters)
	br.String(&member.ReturnType.Category)
	br.String(&member.ReturnType.Name)
	br.String(&member.Security)
	var tags []string
	man.readTags(br, &tags)
	member.Tags = rbxapijson.Tags(tags)
	*p = &member
}

func (man *Manifest) writeCallback(bw *binaryWriter, member *rbxapijson.Callback) {
	bw.String(member.Name)
	man.writeParameters(bw, member.Parameters)
	bw.String(member.ReturnType.Category)
	bw.String(member.ReturnType.Name)
	bw.String(member.Security)
	man.writeTags(bw, []string(member.Tags))
}

func (man *Manifest) readParameters(br *binaryReader, params *[]rbxapijson.Parameter) {
	var length uint32
	br.Number(&length)
	*params = make([]rbxapijson.Parameter, length)
	for i, param := range *params {
		br.String(&param.Type.Category)
		br.String(&param.Type.Name)
		br.String(&param.Name)
		var hasDefault uint8
		br.Number(&hasDefault)
		if hasDefault != 0 {
			param.HasDefault = true
			br.String(&param.Default)
		}
		(*params)[i] = param
	}
}

func (man *Manifest) writeParameters(bw *binaryWriter, params []rbxapijson.Parameter) {
	bw.Number(uint32(len(params)))
	for _, param := range params {
		bw.String(param.Type.Category)
		bw.String(param.Type.Name)
		bw.String(param.Name)
		if param.HasDefault {
			bw.Number(uint8(1))
			bw.String(param.Default)
		} else {
			bw.Number(uint8(0))
		}
	}
}

func (man *Manifest) readEnum(br *binaryReader, p **rbxapijson.Enum) {
	enum := rbxapijson.Enum{}
	br.String(&enum.Name)
	var length uint32
	br.Number(&length)
	enum.Items = make([]*rbxapijson.EnumItem, length)
	for i := range enum.Items {
		man.readEnumItem(br, &enum.Items[i])
	}
	var tags []string
	man.readTags(br, &tags)
	enum.Tags = rbxapijson.Tags(tags)
	*p = &enum
}

func (man *Manifest) writeEnum(bw *binaryWriter, enum *rbxapijson.Enum) {
	bw.String(enum.Name)
	bw.Number(uint32(len(enum.Items)))
	for _, item := range enum.Items {
		man.writeEnumItem(bw, item)
	}
	man.writeTags(bw, []string(enum.Tags))
}

func (man *Manifest) readEnumItem(br *binaryReader, p **rbxapijson.EnumItem) {
	item := rbxapijson.EnumItem{}
	br.String(&item.Name)
	var value uint32
	br.Number(&value)
	item.Value = int(value)
	var tags []string
	man.readTags(br, &tags)
	item.Tags = rbxapijson.Tags(tags)
	*p = &item
}

func (man *Manifest) writeEnumItem(bw *binaryWriter, item *rbxapijson.EnumItem) {
	bw.String(item.Name)
	bw.Number(uint32(item.Value))
	man.writeTags(bw, []string(item.Tags))
}

func (man *Manifest) readValue(br *binaryReader, p **Value) {
	value := Value{}
	var valueType uint8
	br.Number(&valueType)
	switch valueType {
	case 1:
		value.V = false
	case 2:
		value.V = true
	case 3:
		var v uint32
		br.Number(&v)
		value.V = int(v)
	case 4:
		var v string
		br.String(&v)
		value.V = v
	case 5:
		var v rbxapijson.Type
		br.String(&v.Category)
		br.String(&v.Name)
		value.V = v
	case 6:
		var v []string
		man.readTags(br, &v)
		value.V = v
	case 7:
		var v []rbxapijson.Parameter
		man.readParameters(br, &v)
		value.V = rbxapijson.Parameters{List: &v}
	}
	*p = &value
}

func (man *Manifest) writeValue(bw *binaryWriter, value *Value) {
	switch value := value.V.(type) {
	case bool:
		if !value {
			bw.Number(uint8(1))
		} else {
			bw.Number(uint8(2))
		}
	case int:
		bw.Number(uint8(3))
		bw.Number(uint32(value))
	case string:
		bw.Number(uint8(4))
		bw.String(value)
	case rbxapijson.Type:
		bw.Number(uint8(5))
		bw.String(value.Category)
		bw.String(value.Name)
	case []string:
		bw.Number(uint8(6))
		man.writeTags(bw, value)
	case rbxapijson.Parameters:
		bw.Number(uint8(7))
		if value.List == nil {
			man.writeParameters(bw, nil)
		} else {
			man.writeParameters(bw, *value.List)
		}
	}
}

func (man *Manifest) readTags(br *binaryReader, tags *[]string) {
	var length uint32
	br.Number(&length)
	*tags = make([]string, length)
	for i, tag := range *tags {
		br.String(&tag)
		(*tags)[i] = tag
	}
}

func (man *Manifest) writeTags(bw *binaryWriter, tags []string) {
	bw.Number(uint32(len(tags)))
	for _, tag := range tags {
		bw.String(tag)
	}
}

func ReadManifestJSON(r io.Reader) (manifest *Manifest, err error) {
	manifest = &Manifest{}
	err = json.NewDecoder(r).Decode(manifest)
	return manifest, err
}

func WriteManifestJSON(w io.Writer, manifest *Manifest) (err error) {
	je := json.NewEncoder(w)
	je.SetEscapeHTML(false)
	je.SetIndent("", "\t")
	return je.Encode(manifest)
}

func ReadManifest(r io.Reader) (manifest *Manifest, err error) {
	manifest = &Manifest{}
	_, err = manifest.ReadFrom(r)
	return manifest, err
}

func WriteManifest(w io.Writer, manifest *Manifest) (err error) {
	_, err = manifest.WriteTo(w)
	return err
}
