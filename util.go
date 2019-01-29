package main

import (
	"github.com/pkg/errors"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"html/template"
	"io"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

////////////////////////////////////////////////////////////////

// DecodeWrapper wraps a Reader so that the underlying error and byte count
// can be acquired after decoding.
type DecodeWrapper struct {
	r   io.Reader
	n   int64
	err error
}

func NewDecodeWrapper(r io.Reader) *DecodeWrapper {
	return &DecodeWrapper{r: r}
}

func (w *DecodeWrapper) Read(p []byte) (n int, err error) {
	n, err = w.r.Read(p)
	w.n += int64(n)
	w.err = err
	return n, err
}

func (w *DecodeWrapper) Result() (n int64, err error) {
	return w.n, w.err
}

func (w *DecodeWrapper) BytesRead() int64 {
	return w.n
}

func (w *DecodeWrapper) Error() error {
	return w.err
}

////////////////////////////////////////////////////////////////

// EncodeWrapper wraps a Writer so that the underlying error and byte count
// can be acquired after encoding.
type EncodeWrapper struct {
	w   io.Writer
	n   int64
	err error
}

func NewEncodeWrapper(w io.Writer) *EncodeWrapper {
	return &EncodeWrapper{w: w}
}

func (w *EncodeWrapper) Write(p []byte) (n int, err error) {
	n, err = w.w.Write(p)
	w.n += int64(n)
	w.err = err
	return n, err
}

func (w *EncodeWrapper) Result() (n int64, err error) {
	return w.n, w.err
}

func (w *EncodeWrapper) BytesRead() int64 {
	return w.n
}

func (w *EncodeWrapper) Error() error {
	return w.err
}

////////////////////////////////////////////////////////////////

// Converts a value into a string. Only handles types found in rbxapi
// structures.
func ToString(v interface{}) string {
	switch v := v.(type) {
	case Value:
		return ToString(v.V)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(v)
	case string:
		return v
	case rbxapijson.Type:
		return v.String()
	case []string:
		return "[" + strings.Join(v, ", ") + "]"
	case rbxapijson.Tags:
		return "[" + strings.Join(v, ", ") + "]"
	case rbxapijson.Parameters:
		n := v.GetLength()
		ss := make([]string, n)
		for i := 0; i < n; i++ {
			param := v.GetParameter(i).(rbxapijson.Parameter)
			ss[i] = param.Type.String() + " " + param.Name
			if param.HasDefault {
				ss[i] += " = " + param.Default
			}
		}
		return "(" + strings.Join(ss, ", ") + ")"
	}
	return "<unknown value " + reflect.TypeOf(v).String() + ">"
}

// Generates a list of actions for each member of the element.
func MakeSubactions(action Action) []Action {
	if class := action.Class; class != nil {
		actions := make([]Action, len(class.Members))
		for i, member := range class.Members {
			actions[i] = Action{
				Type:  action.GetType(),
				Class: class,
			}
			actions[i].SetMember(member)
		}
		return actions
	} else if enum := action.Enum; enum != nil {
		actions := make([]Action, len(enum.Items))
		for i, item := range enum.Items {
			actions[i] = Action{
				Type:     action.GetType(),
				Enum:     enum,
				EnumItem: item,
			}
		}
		return actions
	}
	return nil
}

func FilterList(filter string, list interface{}) interface{} {
	switch list := list.(type) {
	case []*ClassEntity:
		var filtered []*ClassEntity
		switch filter {
		case "added":
			for _, entity := range list {
				if !entity.Removed {
					filtered = append(filtered, entity)
				}
			}
			return filtered
		case "removed":
			for _, entity := range list {
				if entity.Removed {
					filtered = append(filtered, entity)
				}
			}
			return filtered
		}
	case []*MemberEntity:
		var filtered []*MemberEntity
		switch filter {
		case "added":
			for _, entity := range list {
				if !entity.Removed {
					filtered = append(filtered, entity)
				}
			}
			return filtered
		case "removed":
			for _, entity := range list {
				if entity.Removed {
					filtered = append(filtered, entity)
				}
			}
			return filtered
		case "implicit added":
			for _, entity := range list {
				if !entity.Removed && !entity.Parent.Removed {
					filtered = append(filtered, entity)
				}
			}
			return filtered
		case "implicit removed":
			for _, entity := range list {
				if entity.Removed || entity.Parent.Removed {
					filtered = append(filtered, entity)
				}
			}
			return filtered
		}
	case []Referrer:
		var filtered []Referrer
		switch filter {
		case "added":
			for _, referrer := range list {
				if !referrer.Member.Removed {
					filtered = append(filtered, referrer)
				}
			}
			return filtered
		case "removed":
			for _, referrer := range list {
				if referrer.Member.Removed {
					filtered = append(filtered, referrer)
				}
			}
			return filtered
		case "implicit added":
			for _, referrer := range list {
				if !referrer.Member.Removed && !referrer.Member.Parent.Removed {
					filtered = append(filtered, referrer)
				}
			}
			return filtered
		case "implicit removed":
			for _, referrer := range list {
				if referrer.Member.Removed || referrer.Member.Parent.Removed {
					filtered = append(filtered, referrer)
				}
			}
			return filtered
		}
	case []*EnumEntity:
		var filtered []*EnumEntity
		switch filter {
		case "added":
			for _, entity := range list {
				if !entity.Removed {
					filtered = append(filtered, entity)
				}
			}
			return filtered
		case "removed":
			for _, entity := range list {
				if entity.Removed {
					filtered = append(filtered, entity)
				}
			}
			return filtered
		}
	case []*EnumItemEntity:
		var filtered []*EnumItemEntity
		switch filter {
		case "implicit added":
			for _, entity := range list {
				if !entity.Removed && !entity.Parent.Removed {
					filtered = append(filtered, entity)
				}
			}
			return filtered
		case "implicit removed":
			for _, entity := range list {
				if entity.Removed || entity.Parent.Removed {
					filtered = append(filtered, entity)
				}
			}
			return filtered
		case "added":
			for _, entity := range list {
				if !entity.Removed {
					filtered = append(filtered, entity)
				}
			}
			return filtered
		case "removed":
			for _, entity := range list {
				if entity.Removed {
					filtered = append(filtered, entity)
				}
			}
			return filtered
		}
	case []*TypeEntity:
		var filtered []*TypeEntity
		switch filter {
		case "added":
			for _, entity := range list {
				if !entity.Removed {
					filtered = append(filtered, entity)
				}
			}
			return filtered
		case "removed":
			for _, entity := range list {
				if entity.Removed {
					filtered = append(filtered, entity)
				}
			}
			return filtered
		}
	case []ElementTyper:
		var filtered []ElementTyper
		switch filter {
		case "class":
			for _, entity := range list {
				if entity.ElementType().Category == "Class" && !entity.IsRemoved() {
					filtered = append(filtered, entity)
				}
			}
			return filtered
		case "enum":
			for _, entity := range list {
				if entity.ElementType().Category == "Enum" && !entity.IsRemoved() {
					filtered = append(filtered, entity)
				}
			}
			return filtered
		case "type":
			for _, entity := range list {
				if cat := entity.ElementType().Category; cat != "Class" && cat != "Enum" && !entity.IsRemoved() {
					filtered = append(filtered, entity)
				}
			}
			return filtered
		}
	}
	return list
}

func SortedList(list interface{}) interface{} {
	switch src := list.(type) {
	case []*ClassEntity:
		dst := make([]*ClassEntity, len(src))
		copy(dst, src)
		sort.Slice(dst, func(i, j int) bool { return dst[i].ID < dst[j].ID })
		list = dst
	case []*MemberEntity:
		dst := make([]*MemberEntity, len(src))
		copy(dst, src)
		sort.Slice(dst, func(i, j int) bool { return dst[i].ID[1] < dst[j].ID[1] })
		list = dst
	case []*EnumEntity:
		dst := make([]*EnumEntity, len(src))
		copy(dst, src)
		sort.Slice(dst, func(i, j int) bool { return dst[i].ID < dst[j].ID })
		list = dst
	case []*EnumItemEntity:
		dst := make([]*EnumItemEntity, len(src))
		copy(dst, src)
		sort.Slice(dst, func(i, j int) bool { return dst[i].ID[1] < dst[j].ID[1] })
		list = dst
	case []*TypeEntity:
		dst := make([]*TypeEntity, len(src))
		copy(dst, src)
		sort.Slice(dst, func(i, j int) bool { return dst[i].ID < dst[j].ID })
		list = dst
	case []ElementTyper:
		dst := make([]ElementTyper, len(src))
		copy(dst, src)
		sort.Slice(dst, func(i, j int) bool { return dst[i].Identifier() < dst[j].Identifier() })
		list = dst
	}
	return list
}

func reflectIndirect(v reflect.Value) (rv reflect.Value, isNil bool) {
	for ; v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface; v = v.Elem() {
		if v.IsNil() {
			return v, true
		}
	}
	return v, false
}

func reflectLength(item interface{}) (int, error) {
	v := reflect.ValueOf(item)
	if !v.IsValid() {
		return 0, errors.New("len of untyped nil")
	}
	v, isNil := reflectIndirect(v)
	if isNil {
		return 0, errors.New("len of nil pointer")
	}
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return v.Len(), nil
	case reflect.Int:
		return int(v.Int()), nil
	}
	return 0, errors.Errorf("len of type %s", v.Type())
}

func IsType(v interface{}, t string) bool {
	if v == nil {
		return "nil" == t
	}
	return reflect.TypeOf(v).String() == t
}

func FormatQuantity(i interface{}, singular, plural string) string {
	v, err := reflectLength(i)
	if err != nil || v == 1 {
		return singular
	}
	return plural
}

func GetType(v interface{}) string {
	return reflect.TypeOf(v).String()
}

func LastIndex(v interface{}) int {
	length, err := reflectLength(v)
	if err != nil || length == 0 {
		return 0
	}
	return length - 1
}

// Compiles templates in specified folder as a single template. Templates are
// named as the file name without the extension.
func CompileTemplates(dir string, funcs template.FuncMap) (tmpl *template.Template, err error) {
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	tmpl = template.New("")
	tmpl.Funcs(funcs)
	for _, fi := range fis {
		base := filepath.Base(fi.Name())
		name := base[:len(base)-len(filepath.Ext(base))]
		b, err := ioutil.ReadFile(filepath.Join(dir, fi.Name()))
		if err != nil {
			return nil, err
		}
		t := tmpl.New(name)
		if _, err = t.Parse(string(b)); err != nil {
			return nil, err
		}
		t.Funcs(funcs)
	}
	return
}

func PackValues(a ...interface{}) []interface{} {
	return a
}

func UnpackValues(a []interface{}, args ...string) interface{} {
	fields := make([]reflect.StructField, len(args))
	for i, arg := range args {
		var typ reflect.Type
		if i < len(a) {
			typ = reflect.TypeOf(a[i])
		} else {
			typ = reflect.TypeOf([]interface{}{}).Elem()
		}
		fields[i] = reflect.StructField{Name: arg, Type: typ}
	}
	v := reflect.New(reflect.StructOf(fields))
	for i, arg := range a {
		reflect.Indirect(v).Field(i).Set(reflect.ValueOf(arg))
	}
	return v.Interface()
}

////////////////////////////////////////////////////////////////

func MergePatches(left, right []Patch, filter func(*Action) bool) []Patch {
	var patches []Patch
	for _, l := range left {
		patch := Patch{
			Info:    l.Info,
			Actions: make([]Action, len(l.Actions)),
		}
		copy(patch.Actions, l.Actions)
		patches = append(patches, patch)
	}
loop:
	for _, r := range right {
		for p, patch := range patches {
			if patch.Info.Equal(r.Info) {
				if filter == nil {
					patches[p].Actions = append(patches[p].Actions, r.Actions...)
				} else {
					for _, action := range r.Actions {
						if filter(&action) {
							patches[p].Actions = append(patches[p].Actions, action)
						}
					}
				}
				continue loop
			}
		}
		patch := Patch{
			Info:    r.Info,
			Actions: make([]Action, len(r.Actions)),
		}
		if filter == nil {
			copy(patch.Actions, r.Actions)
		} else {
			patch.Actions = patch.Actions[:0]
			for _, action := range r.Actions {
				if filter(&action) {
					patch.Actions = append(patch.Actions, action)
				}
			}
		}
		patches = append(patches, patch)
	}
	return patches
}

////////////////////////////////////////////////////////////////
