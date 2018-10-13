package main

import (
	"github.com/pkg/errors"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"io"
	"reflect"
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

////////////////////////////////////////////////////////////////
