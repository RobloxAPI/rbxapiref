package main

import (
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"github.com/robloxapi/rbxapi/rbxapijson/diff"
	"github.com/robloxapi/rbxapiref/fetch"
	"html/template"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// Converts a value into a string. Only handles types found in rbxapi
// structures.
func toString(v interface{}) string {
	switch v := v.(type) {
	case Value:
		return toString(v.V)
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
	case []rbxapijson.Parameter:
		ss := make([]string, len(v))
		for i, param := range v {
			ss[i] = param.Type.String() + " " + param.Name
			if param.Default != nil {
				ss[i] += " = " + *param.Default
			}
		}
		return "(" + strings.Join(ss, ", ") + ")"
	}
	return "<unknown value>"
}

// Generates a list of actions for each member of the element.
func makeSubactions(action Action) []Action {
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

// Compiles templates in specified folder as a single template. Templates are
// named as the file name without the extension.
func compileTemplates(dir string, funcs template.FuncMap) (tmpl *template.Template, err error) {
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

const SettingsFile = "settings.json"

func main() {
	spew.Config.DisableMethods = true
	spew.Config.DisablePointerMethods = true
	spew.Config.DisablePointerAddresses = true
	spew.Config.Indent = "\t"

	// Initialize root.
	data := &Data{}

	// Load settings.
	if f, err := os.Open(SettingsFile); err == nil {
		err := json.NewDecoder(f).Decode(&data.Settings)
		f.Close()
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	manifestPath := filepath.Join(
		data.Settings.Output.Root,
		data.Settings.Output.Sub,
		data.Settings.Output.Manifest,
	)

	// Load cache.
	client := &fetch.Client{}
	prevPatches := []Patch{}
	{
		f, err := os.Open(manifestPath)
		if err == nil {
			err = json.NewDecoder(f).Decode(&prevPatches)
			f.Close()
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}

	// Load builds.
	client.CacheMode = fetch.CacheNone
	builds := []Build{}
	for _, cfg := range data.Settings.UseConfigs {
		client.Config = data.Settings.Configs[cfg]
		bs, err := client.Builds()
		if err != nil {
			fmt.Println(cfg, "error fetching builds:", err)
			return
		}
		for _, b := range bs {
			builds = append(builds, Build{Config: cfg, Info: BuildInfo(b)})
		}
	}
	client.CacheMode = fetch.CacheTemp

	// Fetch uncached builds.
loop:
	for _, build := range builds {
		for _, patch := range prevPatches {
			if !build.Info.Equal(patch.Info) {
				// Not relevant; skip.
				continue
			}
			// Current build has a cached version.
			if data.Latest == nil {
				if patch.Prev != nil {
					// Cached build is now the first, but was not originally;
					// actions are stale.
					fmt.Println("STALE ", patch.Info)
					break
				}
			} else {
				if patch.Prev == nil {
					// Cached build was not originally the first, but now is;
					// actions are stale.
					fmt.Println("STALE ", patch.Info)
					break
				}
				if !data.Latest.Info.Equal(*patch.Prev) {
					// Latest build does not match previous build; actions are
					// stale.
					fmt.Println("STALE ", patch.Info)
					break
				}
			}
			// Cached actions are still fresh; set them directly.
			data.Patches = append(data.Patches, patch)
			data.Latest = &Build{Info: patch.Info, Config: patch.Config}
			continue loop
		}
		fmt.Println("NEW ", build.Info)
		client.Config = data.Settings.Configs[build.Config]
		root, err := client.APIDump(build.Info.Hash)
		if err != nil {
			fmt.Println(build.Config, "failed to get build ", build.Info.Hash, err)
			continue
		}
		build.API = root
		var actions []Action
		if data.Latest == nil {
			// First build; compare with nothing.
			actions = WrapActions((&diff.Diff{Prev: nil, Next: build.API}).Diff())
		} else {
			if data.Latest.API == nil {
				// Previous build was cached; fetch its data to compare with
				// current build.
				client.Config = data.Settings.Configs[data.Latest.Config]
				root, err := client.APIDump(data.Latest.Info.Hash)
				if err != nil {
					fmt.Println(data.Latest.Config, "failed to get build ", data.Latest.Info.Hash, err)
					continue
				}
				data.Latest.API = root
			}
			actions = WrapActions((&diff.Diff{Prev: data.Latest.API, Next: build.API}).Diff())
		}
		patch := Patch{Stale: true, Info: build.Info, Config: build.Config, Actions: actions}
		if data.Latest != nil {
			prev := data.Latest.Info
			patch.Prev = &prev
		}
		data.Patches = append(data.Patches, patch)
		b := build
		data.Latest = &b
	}
	// Ensure that the latest API is present.
	if data.Latest.API == nil {
		client.Config = data.Settings.Configs[data.Latest.Config]
		root, err := client.APIDump(data.Latest.Info.Hash)
		if err != nil {
			fmt.Println(data.Latest.Config, "failed to get build ", data.Latest.Info.Hash, err)
			return
		}
		data.Latest.API = root
	}

	// Fetch ReflectionMetadata.
	{
		rmd, err := client.ReflectionMetadata(data.Latest.Info.Hash)
		if err != nil {
			fmt.Println(data.Latest.Config, "failed to get metadata ", data.Latest.Info.Hash, err)
			return
		}
		data.GenerateMetadata(rmd)
	}

	// Fetch explorer icons.
	{
		icon, err := client.ExplorerIcons(data.Latest.Info.Hash)
		if err != nil {
			fmt.Println(data.Latest.Config, "failed to get icons ", data.Latest.Info.Hash, err)
			return
		}
		if err := os.MkdirAll(data.FilePath("resource"), 0666); err != nil {
			fmt.Println(err)
			return
		}
		f, err := os.Create(data.FilePath("resource", "icon-explorer.png"))
		if err != nil {
			fmt.Println(err)
			return
		}
		err = png.Encode(f, icon)
		f.Close()
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	data.GenerateEntities()
	data.GenerateTree()

	// Compile templates.
	var err error
	data.Templates, err = compileTemplates(data.Settings.Input.Templates, template.FuncMap{
		"embed": func(resource string) (interface{}, error) {
			b, err := ioutil.ReadFile(filepath.Join(data.Settings.Input.Resources, resource))
			switch filepath.Ext(resource) {
			case ".css":
				return template.CSS(b), err
			case ".js":
				return template.JS(b), err
			}
			return string(b), err
		},
		"execute": data.ExecuteTemplate,
		"icon":    data.Icon,
		"istype": func(v interface{}, t string) bool {
			if v == nil {
				return "nil" == t
			}
			return reflect.TypeOf(v).String() == t
		},
		"link": func(linkType string, args ...interface{}) string {
			sargs := make([]string, len(args))
			for i, arg := range args {
				switch arg := arg.(type) {
				case int:
					sargs[i] = strconv.Itoa(arg)
				default:
					sargs[i] = arg.(string)
				}
			}
			return data.FileLink(linkType, sargs...)
		},
		"subactions": makeSubactions,
		"subclasses": func(name string) []string {
			node := data.Tree[name]
			if node == nil {
				return nil
			}
			return node.Sub
		},
		"tostring": toString,
		"type": func(v interface{}) string {
			return reflect.TypeOf(v).String()
		},
	})
	if err != nil {
		fmt.Println("failed to open template", err)
		return
	}

	// Generate pages.
	pages := []func(*Data) error{
		GenerateResPage,
		GenerateIndexPage,
		GenerateUpdatesPage,
		GenerateRefPage,
	}
	for _, page := range pages {
		if err := page(data); err != nil {
			fmt.Println(err)
			return
		}
	}

	// Save cache.
	{
		f, err := os.Create(manifestPath)
		if err != nil {
			fmt.Println(err)
			return
		}
		je := json.NewEncoder(f)
		je.SetEscapeHTML(false)
		je.SetIndent("", "\t")
		err = je.Encode(data.Patches)
		f.Close()
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}
