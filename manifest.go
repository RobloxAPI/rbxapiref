package main

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxapi/patch"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"github.com/robloxapi/rbxapiref/internal/binio"
)

type Manifest struct {
	Patches []Patch
}

func (man *Manifest) ReadFrom(r io.Reader) (n int64, err error) {
	br := binio.NewReader(r)
	var length uint32
	br.Number(&length)
	man.Patches = make([]Patch, length)
	for i, patch := range man.Patches {
		man.readPatch(br, &patch)
		man.Patches[i] = patch
		if br.Err != nil {
			break
		}
	}
	return br.End()
}

func (man *Manifest) WriteTo(w io.Writer) (n int64, err error) {
	bw := binio.NewWriter(w)
	bw.Number(uint32(len(man.Patches)))
	for _, patch := range man.Patches {
		man.writePatch(bw, &patch)
		if bw.Err != nil {
			break
		}
	}
	return bw.End()
}

func (man *Manifest) readPatch(br *binio.Reader, patch *Patch) {
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
		if br.Err != nil {
			return
		}
	}
}

func (man *Manifest) writePatch(bw *binio.Writer, patch *Patch) {
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
		if bw.Err != nil {
			return
		}
	}
}

func (man *Manifest) readBuildInfo(br *binio.Reader, info *BuildInfo) {
	br.String(&info.Hash)
	var date string
	br.String(&date)
	if err := info.Date.UnmarshalBinary([]byte(date)); err != nil {
		br.Err = err
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

func (man *Manifest) writeBuildInfo(bw *binio.Writer, info *BuildInfo) {
	bw.String(info.Hash)
	date, err := info.Date.MarshalBinary()
	if err != nil {
		bw.Err = err
		return
	}
	bw.String(string(date))
	bw.Number(uint32(info.Version.Major))
	bw.Number(uint32(info.Version.Minor))
	bw.Number(uint32(info.Version.Maint))
	bw.Number(uint32(info.Version.Build))
}

func (man *Manifest) readAction(br *binio.Reader, action *Action) {
	var data uint8
	br.Number(&data)
	action.Type = patch.Type(binio.GetBits(uint64(data), 0, 2) - 1)
	switch binio.GetBits(uint64(data), 2, 5) {
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
		br.Err = errors.New("invalid action")
		return
	}
	if action.Type == patch.Change {
		br.String(&action.Field)
		man.readValue(br, &action.Prev)
		man.readValue(br, &action.Next)
	}
}

func (man *Manifest) writeAction(bw *binio.Writer, action *Action) {
	var data uint64
	data = binio.SetBits(data, 0, 2, int(action.Type)+1)
	switch {
	case action.Property != nil:
		data = binio.SetBits(data, 2, 5, 1)
		bw.Number(uint8(data))
		man.writeClass(bw, action.Class)
		man.writeProperty(bw, action.Property)
	case action.Function != nil:
		data = binio.SetBits(data, 2, 5, 2)
		bw.Number(uint8(data))
		man.writeClass(bw, action.Class)
		man.writeFunction(bw, action.Function)
	case action.Event != nil:
		data = binio.SetBits(data, 2, 5, 3)
		bw.Number(uint8(data))
		man.writeClass(bw, action.Class)
		man.writeEvent(bw, action.Event)
	case action.Callback != nil:
		data = binio.SetBits(data, 2, 5, 4)
		bw.Number(uint8(data))
		man.writeClass(bw, action.Class)
		man.writeCallback(bw, action.Callback)
	case action.Class != nil:
		data = binio.SetBits(data, 2, 5, 5)
		bw.Number(uint8(data))
		man.writeClass(bw, action.Class)
	case action.EnumItem != nil:
		data = binio.SetBits(data, 2, 5, 6)
		bw.Number(uint8(data))
		man.writeEnum(bw, action.Enum)
		man.writeEnumItem(bw, action.EnumItem)
	case action.Enum != nil:
		data = binio.SetBits(data, 2, 5, 7)
		bw.Number(uint8(data))
		man.writeEnum(bw, action.Enum)
	default:
		bw.Err = errors.New("invalid action")
		return
	}
	if binio.GetBits(data, 0, 2) == 1 {
		bw.String(action.Field)
		man.writeValue(bw, action.Prev)
		man.writeValue(bw, action.Next)
	}
}

func (man *Manifest) readClass(br *binio.Reader, p **rbxapijson.Class) {
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
			br.Err = errors.New("invalid member type")
		}
		if br.Err != nil {
			return
		}
	}
	var tags []string
	man.readTags(br, &tags)
	class.Tags = rbxapijson.Tags(tags)
	*p = &class
}

func (man *Manifest) writeClass(bw *binio.Writer, class *rbxapijson.Class) {
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

func (man *Manifest) readProperty(br *binio.Reader, p **rbxapijson.Property) {
	member := rbxapijson.Property{}
	br.String(&member.Name)
	br.String(&member.ValueType.Category)
	br.String(&member.ValueType.Name)
	br.String(&member.Category)
	br.String(&member.ReadSecurity)
	br.String(&member.WriteSecurity)
	var ser uint8
	br.Number(&ser)
	member.CanLoad = binio.GetBit(uint64(ser), 0)
	member.CanSave = binio.GetBit(uint64(ser), 1)
	var tags []string
	man.readTags(br, &tags)
	member.Tags = rbxapijson.Tags(tags)
	*p = &member
}

func (man *Manifest) writeProperty(bw *binio.Writer, member *rbxapijson.Property) {
	bw.String(member.Name)
	bw.String(member.ValueType.Category)
	bw.String(member.ValueType.Name)
	bw.String(member.Category)
	bw.String(member.ReadSecurity)
	bw.String(member.WriteSecurity)
	var ser uint64
	ser = binio.SetBit(ser, 0, member.CanLoad)
	ser = binio.SetBit(ser, 1, member.CanSave)
	bw.Number(uint8(ser))
	man.writeTags(bw, []string(member.Tags))
}

func (man *Manifest) readFunction(br *binio.Reader, p **rbxapijson.Function) {
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

func (man *Manifest) writeFunction(bw *binio.Writer, member *rbxapijson.Function) {
	bw.String(member.Name)
	man.writeParameters(bw, member.Parameters)
	bw.String(member.ReturnType.Category)
	bw.String(member.ReturnType.Name)
	bw.String(member.Security)
	man.writeTags(bw, []string(member.Tags))
}

func (man *Manifest) readEvent(br *binio.Reader, p **rbxapijson.Event) {
	member := rbxapijson.Event{}
	br.String(&member.Name)
	man.readParameters(br, &member.Parameters)
	br.String(&member.Security)
	var tags []string
	man.readTags(br, &tags)
	member.Tags = rbxapijson.Tags(tags)
	*p = &member
}

func (man *Manifest) writeEvent(bw *binio.Writer, member *rbxapijson.Event) {
	bw.String(member.Name)
	man.writeParameters(bw, member.Parameters)
	bw.String(member.Security)
	man.writeTags(bw, []string(member.Tags))
}

func (man *Manifest) readCallback(br *binio.Reader, p **rbxapijson.Callback) {
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

func (man *Manifest) writeCallback(bw *binio.Writer, member *rbxapijson.Callback) {
	bw.String(member.Name)
	man.writeParameters(bw, member.Parameters)
	bw.String(member.ReturnType.Category)
	bw.String(member.ReturnType.Name)
	bw.String(member.Security)
	man.writeTags(bw, []string(member.Tags))
}

func (man *Manifest) readParameters(br *binio.Reader, params *[]rbxapijson.Parameter) {
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

func (man *Manifest) writeParameters(bw *binio.Writer, params []rbxapijson.Parameter) {
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

func (man *Manifest) readEnum(br *binio.Reader, p **rbxapijson.Enum) {
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

func (man *Manifest) writeEnum(bw *binio.Writer, enum *rbxapijson.Enum) {
	bw.String(enum.Name)
	bw.Number(uint32(len(enum.Items)))
	for _, item := range enum.Items {
		man.writeEnumItem(bw, item)
	}
	man.writeTags(bw, []string(enum.Tags))
}

func (man *Manifest) readEnumItem(br *binio.Reader, p **rbxapijson.EnumItem) {
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

func (man *Manifest) writeEnumItem(bw *binio.Writer, item *rbxapijson.EnumItem) {
	bw.String(item.Name)
	bw.Number(uint32(item.Value))
	man.writeTags(bw, []string(item.Tags))
}

func (man *Manifest) readValue(br *binio.Reader, p **Value) {
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

func (man *Manifest) writeValue(bw *binio.Writer, value *Value) {
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

func (man *Manifest) readTags(br *binio.Reader, tags *[]string) {
	var length uint32
	br.Number(&length)
	*tags = make([]string, length)
	for i, tag := range *tags {
		br.String(&tag)
		(*tags)[i] = tag
	}
}

func (man *Manifest) writeTags(bw *binio.Writer, tags []string) {
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
