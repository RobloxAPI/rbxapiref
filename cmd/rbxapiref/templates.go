package main

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"net/url"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"github.com/robloxapi/rbxapiref/builds"
	"github.com/robloxapi/rbxapiref/entities"
)

type listFilter struct {
	Type    reflect.Type
	Filters string
}

var listFilters = map[listFilter]reflect.Value{}

func AddListFilter(filter string, fn interface{}) {
	filterFunc := reflect.ValueOf(fn)
	t := filterFunc.Type()
	if t == nil || t.Kind() != reflect.Func {
		panic("invalid list filter function")
	}
	if t.NumOut() != 1 || t.Out(0).Kind() != reflect.Bool {
		panic("invalid list filter function output parameter")
	}
	if t.NumIn() != 1 {
		panic("invalid list filter function input parameter")
	}
	listFilters[listFilter{reflect.SliceOf(t.In(0)), filter}] = filterFunc
}

func init() {
	AddListFilter("Added", func(v *entities.ClassEntity) bool { return !v.Removed })
	AddListFilter("Removed", func(v *entities.ClassEntity) bool { return v.Removed })
	AddListFilter("Documented", func(v *entities.ClassEntity) bool { return v.Document != nil })

	AddListFilter("Added", func(v *entities.MemberEntity) bool { return !v.Removed })
	AddListFilter("Removed", func(v *entities.MemberEntity) bool { return v.Removed })
	AddListFilter("ImplicitAdded", func(v *entities.MemberEntity) bool { return !v.Removed && !v.Parent.Removed })
	AddListFilter("ImplicitRemoved", func(v *entities.MemberEntity) bool { return v.Removed || v.Parent.Removed })
	AddListFilter("Documented", func(v *entities.MemberEntity) bool { return v.Document != nil })

	AddListFilter("Added", func(v entities.Referrer) bool { return !v.Member.Removed })
	AddListFilter("Removed", func(v entities.Referrer) bool { return v.Member.Removed })
	AddListFilter("ImplicitAdded", func(v entities.Referrer) bool { return !v.Member.Removed && !v.Member.Parent.Removed })
	AddListFilter("ImplicitRemoved", func(v entities.Referrer) bool { return v.Member.Removed || v.Member.Parent.Removed })
	AddListFilter("Documented", func(v entities.Referrer) bool { return v.Member.Document != nil })

	AddListFilter("Added", func(v *entities.EnumEntity) bool { return !v.Removed })
	AddListFilter("Removed", func(v *entities.EnumEntity) bool { return v.Removed })
	AddListFilter("Documented", func(v *entities.EnumEntity) bool { return v.Document != nil })

	AddListFilter("Added", func(v *entities.EnumItemEntity) bool { return !v.Removed })
	AddListFilter("Removed", func(v *entities.EnumItemEntity) bool { return v.Removed })
	AddListFilter("ImplicitAdded", func(v *entities.EnumItemEntity) bool { return !v.Removed && !v.Parent.Removed })
	AddListFilter("ImplicitRemoved", func(v *entities.EnumItemEntity) bool { return v.Removed || v.Parent.Removed })
	AddListFilter("Documented", func(v *entities.EnumItemEntity) bool { return v.Document != nil })

	AddListFilter("Added", func(v *entities.TypeEntity) bool { return !v.Removed })
	AddListFilter("Removed", func(v *entities.TypeEntity) bool { return v.Removed })
	AddListFilter("Documented", func(v *entities.TypeEntity) bool { return v.Document != nil })

	AddListFilter("Class", func(v entities.ElementTyper) bool { return v.ElementType().Category == "Class" && !v.IsRemoved() })
	AddListFilter("Enum", func(v entities.ElementTyper) bool { return v.ElementType().Category == "Enum" && !v.IsRemoved() })
	AddListFilter("Type", func(v entities.ElementTyper) bool {
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

func TemplateFuncs(data *Data) template.FuncMap {
	funcs := template.FuncMap{}

	{
		outputSettings := data.Settings.Output
		funcs["cards"] = func(pages ...*Page) (elements []template.HTML, err error) {
			getField := func(name string) (value string, ok bool) {
				for _, page := range pages {
					if v, k := page.Meta[name]; k {
						value = v
						ok = true
					}
				}
				return value, ok
			}

			elements = append(elements,
				generateMetaTag("property", "og:type", "website"),
				generateMetaTag("name", "twitter:card", "summary"),
			)
			if title, ok := getField("Title"); ok {
				elements = append(elements,
					generateMetaTag("property", "og:title", title),
					generateMetaTag("name", "twitter:title", title),
				)
			}
			if desc, ok := getField("Description"); ok {
				elements = append(elements,
					generateMetaTag("property", "og:description", desc),
					generateMetaTag("name", "twitter:description", desc),
				)
			}
			if image, ok := getField("Image"); ok {
				u := (&url.URL{Scheme: "https", Host: outputSettings.Host, Path: outputSettings.FileLink("resource", image)}).String()
				elements = append(elements,
					generateMetaTag("property", "og:image", u),
					generateMetaTag("name", "twitter:image", u),
				)
			}

			return elements, nil
		}
	}
	funcs["docstatus"] = func(entity interface{}) entities.DocStatus {
		if entity, ok := entity.(entities.Documentable); ok {
			return entity.GetDocStatus()
		}
		return GenerateDocStatus(entity)
	}
	funcs["document"] = entities.QueryDocument
	{
		resources := data.Settings.Input.Resources
		funcs["embed"] = func(resource string) (interface{}, error) {
			b, err := ioutil.ReadFile(filepath.Join(resources, resource))
			switch filepath.Ext(resource) {
			case ".css":
				return template.CSS(b), err
			case ".js":
				return template.JS(b), err
			case ".html", ".svg":
				return template.HTML(b), err
			}
			return string(b), err
		}
	}
	funcs["execute"] = func(name string, tdata interface{}) (template.HTML, error) {
		var buf bytes.Buffer
		err := data.Templates.ExecuteTemplate(&buf, name, tdata)
		return template.HTML(buf.String()), err
	}
	funcs["filter"] = FilterList
	funcs["history"] = data.GenerateHistoryElements
	funcs["icon"] = data.Entities.Icon
	funcs["istype"] = func(v interface{}, t string) bool {
		if v == nil {
			return "nil" == t
		}
		return reflect.TypeOf(v).String() == t
	}

	funcs["last"] = func(v interface{}) int {
		length, err := reflectLength(v)
		if err != nil || length == 0 {
			return 0
		}
		return length - 1
	}
	funcs["list"] = func(v interface{}) []string {
		var s string
		r := reflect.ValueOf(v)
		if t := r.Type(); t.Kind() == reflect.String {
			s = r.String()
		} else if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
			s = string(r.Bytes())
		} else {
			return nil
		}
		list := strings.Split(s, ",")
		for i, s := range list {
			list[i] = strings.TrimSpace(s)
		}
		return list
	}
	funcs["link"] = func(linkType string, args ...interface{}) string {
		sargs := make([]string, len(args))
		for i, arg := range args {
			switch arg := arg.(type) {
			case int:
				sargs[i] = strconv.Itoa(arg)
			default:
				sargs[i] = arg.(string)
			}
		}
		return data.Settings.Output.FileLink(linkType, sargs...)
	}
	funcs["pack"] = func(a ...interface{}) []interface{} {
		return a
	}
	funcs["patchtype"] = builds.PatchTypeString
	funcs["quantity"] = func(i interface{}, singular, plural string) string {
		v, err := reflectLength(i)
		if err != nil || v == 1 {
			return singular
		}
		return plural
	}
	funcs["renderdoc"] = entities.RenderDocument
	funcs["resources"] = data.GenerateResourceElements
	funcs["sortedlist"] = func(list interface{}) interface{} {
		switch src := list.(type) {
		case []*entities.ClassEntity:
			dst := make([]*entities.ClassEntity, len(src))
			copy(dst, src)
			sort.Slice(dst, func(i, j int) bool { return dst[i].ID < dst[j].ID })
			list = dst
		case []*entities.MemberEntity:
			dst := make([]*entities.MemberEntity, len(src))
			copy(dst, src)
			sort.Slice(dst, func(i, j int) bool { return dst[i].ID[1] < dst[j].ID[1] })
			list = dst
		case []*entities.EnumEntity:
			dst := make([]*entities.EnumEntity, len(src))
			copy(dst, src)
			sort.Slice(dst, func(i, j int) bool { return dst[i].ID < dst[j].ID })
			list = dst
		case []*entities.EnumItemEntity:
			dst := make([]*entities.EnumItemEntity, len(src))
			copy(dst, src)
			sort.Slice(dst, func(i, j int) bool { return dst[i].ID[1] < dst[j].ID[1] })
			list = dst
		case []*entities.TypeEntity:
			dst := make([]*entities.TypeEntity, len(src))
			copy(dst, src)
			sort.Slice(dst, func(i, j int) bool { return dst[i].ID < dst[j].ID })
			list = dst
		case []entities.ElementTyper:
			dst := make([]entities.ElementTyper, len(src))
			copy(dst, src)
			sort.Slice(dst, func(i, j int) bool { return dst[i].Identifier() < dst[j].Identifier() })
			list = dst
		}
		return list
	}
	funcs["status"] = data.Entities.ElementStatusClasses
	funcs["subactions"] = builds.MakeSubactions
	funcs["tolower"] = strings.ToLower
	// Converts a value into a string. Only handles types found in rbxapi
	// structures.
	var tostring func(v interface{}) string
	tostring = func(v interface{}) string {
		switch v := v.(type) {
		case builds.Value:
			return tostring(v.V)
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
	funcs["tostring"] = tostring
	funcs["type"] = func(v interface{}) string {
		return reflect.TypeOf(v).String()
	}
	funcs["unpack"] = func(a []interface{}, args ...string) interface{} {
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

	return funcs
}

func compileTemplates(tmpl *template.Template, dir, sub string) (err error) {
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		if fi.IsDir() {
			err = compileTemplates(
				tmpl,
				filepath.Join(dir, fi.Name()),
				path.Join(sub, fi.Name()),
			)
			if err != nil {
				return err
			}
			continue
		}
		base := filepath.Base(fi.Name())
		name := base[:len(base)-len(filepath.Ext(base))]
		name = path.Join(sub, name)
		b, err := ioutil.ReadFile(filepath.Join(dir, fi.Name()))
		if err != nil {
			return err
		}
		t := tmpl.New(name)
		if _, err = t.Parse(string(b)); err != nil {
			return err
		}
	}
	return nil
}

// Compiles templates in specified folder as a single template. Templates are
// named as the file name without the extension.
func CompileTemplates(dir string, funcs template.FuncMap) (tmpl *template.Template, err error) {
	tmpl = template.New("").Funcs(funcs)
	err = compileTemplates(tmpl, dir, "")
	return
}
