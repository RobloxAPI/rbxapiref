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

type listFilter struct {
	Type    reflect.Type
	Filters string
}

var listFilters = map[listFilter]reflect.Value{}

func AddListFilter(t interface{}, filter string, fn interface{}) {
	var typ reflect.Type
	{
		typ = reflect.TypeOf(t)
		if typ == nil || typ.Kind() != reflect.Slice {
			panic("invalid list filter type")
		}
	}
	var filterFunc reflect.Value
	{
		filterFunc = reflect.ValueOf(fn)
		t := filterFunc.Type()
		if t == nil || t.Kind() != reflect.Func {
			panic("invalid list filter function")
		}
		if t.NumOut() != 1 || t.Out(0).Kind() != reflect.Bool {
			panic("invalid list filter function output parameter")
		}
		if t.NumIn() != 1 || t.In(0) != typ.Elem() {
			panic("invalid list filter function input parameter")
		}
	}
	listFilters[listFilter{typ, filter}] = filterFunc
}

func init() {
	AddListFilter([]*ClassEntity{}, "Added", func(v *ClassEntity) bool { return !v.Removed })
	AddListFilter([]*ClassEntity{}, "Removed", func(v *ClassEntity) bool { return v.Removed })
	AddListFilter([]*ClassEntity{}, "Documented", func(v *ClassEntity) bool { return v.Document != nil })

	AddListFilter([]*MemberEntity{}, "Added", func(v *MemberEntity) bool { return !v.Removed })
	AddListFilter([]*MemberEntity{}, "Removed", func(v *MemberEntity) bool { return v.Removed })
	AddListFilter([]*MemberEntity{}, "ImplicitAdded", func(v *MemberEntity) bool { return !v.Removed && !v.Parent.Removed })
	AddListFilter([]*MemberEntity{}, "ImplicitRemoved", func(v *MemberEntity) bool { return v.Removed || v.Parent.Removed })
	AddListFilter([]*MemberEntity{}, "Documented", func(v *MemberEntity) bool { return v.Document != nil })

	AddListFilter([]Referrer{}, "Added", func(v Referrer) bool { return !v.Member.Removed })
	AddListFilter([]Referrer{}, "Removed", func(v Referrer) bool { return v.Member.Removed })
	AddListFilter([]Referrer{}, "ImplicitAdded", func(v Referrer) bool { return !v.Member.Removed && !v.Member.Parent.Removed })
	AddListFilter([]Referrer{}, "ImplicitRemoved", func(v Referrer) bool { return v.Member.Removed || v.Member.Parent.Removed })
	AddListFilter([]Referrer{}, "Documented", func(v Referrer) bool { return v.Member.Document != nil })

	AddListFilter([]*EnumEntity{}, "Added", func(v *EnumEntity) bool { return !v.Removed })
	AddListFilter([]*EnumEntity{}, "Removed", func(v *EnumEntity) bool { return v.Removed })
	AddListFilter([]*EnumEntity{}, "Documented", func(v *EnumEntity) bool { return v.Document != nil })

	AddListFilter([]*EnumItemEntity{}, "Added", func(v *EnumItemEntity) bool { return !v.Removed })
	AddListFilter([]*EnumItemEntity{}, "Removed", func(v *EnumItemEntity) bool { return v.Removed })
	AddListFilter([]*EnumItemEntity{}, "ImplicitAdded", func(v *EnumItemEntity) bool { return !v.Removed && !v.Parent.Removed })
	AddListFilter([]*EnumItemEntity{}, "ImplicitRemoved", func(v *EnumItemEntity) bool { return v.Removed || v.Parent.Removed })
	AddListFilter([]*EnumItemEntity{}, "Documented", func(v *EnumItemEntity) bool { return v.Document != nil })

	AddListFilter([]*TypeEntity{}, "Added", func(v *TypeEntity) bool { return !v.Removed })
	AddListFilter([]*TypeEntity{}, "Removed", func(v *TypeEntity) bool { return v.Removed })
	AddListFilter([]*TypeEntity{}, "Documented", func(v *TypeEntity) bool { return v.Document != nil })

	AddListFilter([]ElementTyper{}, "Class", func(v ElementTyper) bool { return v.ElementType().Category == "Class" && !v.IsRemoved() })
	AddListFilter([]ElementTyper{}, "Enum", func(v ElementTyper) bool { return v.ElementType().Category == "Enum" && !v.IsRemoved() })
	AddListFilter([]ElementTyper{}, "Type", func(v ElementTyper) bool {
		cat := v.ElementType().Category
		return cat != "Class" && cat != "Enum" && !v.IsRemoved()
	})
}

func FilterList(list interface{}, filters ...string) interface{} {
	rlist := reflect.ValueOf(list)
	typ := rlist.Type()
	if typ == nil || typ.Kind() != reflect.Slice {
		return list
	}

	filterFuncs := []reflect.Value{}
	for _, filter := range filters {
		if fn, ok := listFilters[listFilter{typ, filter}]; ok {
			filterFuncs = append(filterFuncs, fn)
		}
	}
	if len(filterFuncs) == 0 {
		return list
	}

	filtered := reflect.MakeSlice(typ, 0, rlist.Len())
loop:
	for i, n := 0, rlist.Len(); i < n; i++ {
		v := rlist.Index(i)
		for _, filter := range filterFuncs {
			if !filter.Call([]reflect.Value{v})[0].Bool() {
				continue loop
			}
		}
		filtered = reflect.Append(filtered, v)
	}
	return filtered.Interface()
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
