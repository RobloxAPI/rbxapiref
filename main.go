package main

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/robloxapi/rbxapi/rbxapijson/diff"
	"github.com/robloxapi/rbxapiref/fetch"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Initialize root.
	data := &Data{CurrentYear: time.Now().Year()}

	// Load settings.
	IfFatal(data.Settings.ReadFile(""))

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
			manifest, err := ReadManifest(f)
			f.Close()
			IfFatal(err, "open manifest")
			prevPatches = manifest.Patches
		}
	}

	// Load builds.
	client.CacheMode = fetch.CacheNone
	builds := []Build{}
	for _, cfg := range data.Settings.UseConfigs {
		client.Config = data.Settings.Configs[cfg]
		bs, err := client.Builds()
		IfFatal(err, "fetch build")
		for _, b := range bs {
			builds = append(builds, Build{Config: cfg, Info: BuildInfo(b)})
		}
	}
	client.CacheMode = fetch.CacheTemp

	sort.Slice(builds, func(i, j int) bool {
		return builds[i].Info.Date.Before(builds[j].Info.Date)
	})

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
					Log("STALE", patch.Info)
					break
				}
			} else {
				if patch.Prev == nil {
					// Cached build was not originally the first, but now is;
					// actions are stale.
					Log("STALE", patch.Info)
					break
				}
				if !data.Latest.Info.Equal(*patch.Prev) {
					// Latest build does not match previous build; actions are
					// stale.
					Log("STALE", patch.Info)
					break
				}
			}
			// Cached actions are still fresh; set them directly.
			data.Patches = append(data.Patches, patch)
			data.Latest = &Build{Info: patch.Info, Config: patch.Config}
			continue loop
		}
		Log("NEW", build.Info)
		client.Config = data.Settings.Configs[build.Config]
		root, err := client.APIDump(build.Info.Hash)
		if IfErrorf(err, "%s: fetch build %s", build.Config, build.Info.Hash) {
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
				if IfErrorf(err, "%s: fetch build %s", data.Latest.Config, data.Latest.Info.Hash) {
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
		IfFatalf(err, "fetch build %s", data.Latest.Info.Hash)
		data.Latest.API = root
	}

	for i, patch := range data.Patches {
		for j := range patch.Actions {
			data.Patches[i].Actions[j].Index = j
		}
	}

	// Fetch ReflectionMetadata.
	{
		rmd, err := client.ReflectionMetadata(data.Latest.Info.Hash)
		IfFatal(err, "fetch metadata ", data.Latest.Info.Hash)
		data.GenerateMetadata(rmd)
	}

	data.GenerateUpdates()
	data.Entities = GenerateEntities(data)
	data.GenerateTree()

	// Compile templates.
	var err error
	data.Templates, err = CompileTemplates(data.Settings.Input.Templates, template.FuncMap{
		"embed":   data.EmbedResource,
		"execute": data.ExecuteTemplate,
		"filter":  FilterList,
		"history": data.GenerateHistoryElements,
		"icon":    data.Icon,
		"istype":  IsType,
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
		"patchtype":  PatchTypeString,
		"quantity":   FormatQuantity,
		"resources":  data.GenerateResourceElements,
		"subactions": MakeSubactions,
		"tolower":    strings.ToLower,
		"tostring":   ToString,
		"type":       GetType,
	})
	IfFatal(err, "open template")

	// Generate pages.
	{
		pageGenerators := []PageGenerator{
			GeneratePageMain,
			GeneratePageIndex,
			GeneratePageAbout,
			GeneratePageUpdates,
			GeneratePageClass,
			GeneratePageEnum,
			GeneratePageType,
		}

		var pages []Page
		for _, generator := range pageGenerators {
			pages = append(pages, generator(data)...)
		}

		// Ensure directories exist.
		dirs := map[string]struct{}{}
		for _, page := range pages {
			dir := filepath.Dir(page.File)
			if _, ok := dirs[dir]; ok {
				continue
			}
			IfFatal(os.MkdirAll(dir, 0755), "make directory")
			dirs[dir] = struct{}{}
		}

		// Copy resources.
		IfFatal(os.MkdirAll(data.FilePath("resource"), 0755), "make directory")
		resources := map[string]*Resource{}
		addResource := func(resource *Resource) {
			if resource.Name == "" || resource.Embed {
				return
			}
			if r, ok := resources[resource.Name]; ok {
				if r.Content != nil {
					IfFatal(errors.Errorf("multiple definitions of resource %s", resource.Name))
				}
			}
			resources[resource.Name] = resource
		}
		for _, page := range pages {
			for i := range page.Styles {
				addResource(&page.Styles[i])
			}
			for i := range page.Scripts {
				addResource(&page.Scripts[i])
			}
			for i := range page.Resources {
				addResource(&page.Resources[i])
			}
		}
		for name, resource := range resources {
			var src io.ReadCloser
			var err error
			if resource.Content != nil {
				src = ioutil.NopCloser(bytes.NewReader(resource.Content))
			} else {
				src, err = os.Open(filepath.Join(data.Settings.Input.Resources, name))
				IfFatal(err, "open resource")
			}
			dst, err := os.Create(data.FilePath("resource", name))
			if err != nil {
				src.Close()
				IfFatal(err, "create resource")
			}
			_, err = io.Copy(dst, src)
			dst.Close()
			src.Close()
			IfFatal(err, "write resource")
		}

		// Generate pages.
		var rootData struct {
			Data     *Data
			MainPage *Page
			Page     *Page
		}
		rootData.Data = data
		// Treat first page with unspecified filename as main page.
		for _, page := range pages {
			if page.File == "" {
				rootData.MainPage = &page
				break
			}
		}
		if rootData.MainPage == nil {
			IfFatal(errors.New("no main template"))
		}
		for _, page := range pages {
			if page.File == "" {
				continue
			}
			file, err := os.Create(page.File)
			IfFatal(err, "create file")
			if page.Data == nil {
				page.Data = data
			}
			rootData.Page = &page
			err = data.Templates.ExecuteTemplate(file, rootData.MainPage.Template, rootData)
			file.Close()
			IfFatal(err, "generate page")
		}
	}

	// Generate search database.
	{
		f, err := os.Create(data.FilePath("search"))
		IfFatal(err, "create search database file")
		db := dbWriter{data: data, w: f}
		db.GenerateDatabase()
		f.Close()
		IfFatal(db.err, "generate search database")
	}

	// Save cache.
	{
		f, err := os.Create(manifestPath)
		IfFatal(err, "create manifest")
		err = WriteManifest(f, &Manifest{data.Patches})
		f.Close()
		IfFatal(err, "encode manifest")
	}
}
