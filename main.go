package main

import (
	"bytes"
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	"github.com/robloxapi/rbxapiref/fetch"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type FlagOptions struct {
	Settings string `
		short:"s"
		long:"settings"
		description:"Specify a custom settings location."
		value-name:"PATH"`
}

func main() {
	var err error

	// Parse flags.
	var flagOptions FlagOptions
	var filters []string
	{
		fp := flags.NewParser(&flagOptions, flags.Default|flags.PassAfterNonOption)
		var err error
		filters, err = fp.Parse()
		if err != nil {
			if err, ok := err.(*flags.Error); ok && err.Type == flags.ErrHelp {
				fmt.Fprintln(os.Stdout, err)
				return
			}
		}
		IfFatal(err, "flag parser error")
	}

	// Initialize root.
	data := &Data{CurrentYear: time.Now().Year()}

	// Load settings.
	data.Settings = *DefaultSettings.Copy()
	IfFatal(data.Settings.ReadFile(flagOptions.Settings))

	// Load manifest.
	manifestPath := filepath.Join(
		data.Settings.Output.Root,
		data.Settings.Output.Sub,
		data.Settings.Output.Manifest,
	)
	var manifest *Manifest
	if f, err := os.Open(manifestPath); err == nil {
		manifest, err = ReadManifest(f)
		f.Close()
		IfFatal(err, "open manifest")
	}

	// Fetch builds.
	builds, err := FetchBuilds(data.Settings)
	IfFatal(err)

	// Merge uncached builds.
	data.Patches, data.Latest, err = MergeBuilds(data.Settings, manifest.Patches, builds)
	IfFatal(err)

	// Fetch ReflectionMetadata.
	data.Metadata, err = GenerateMetadata(&fetch.Client{
		Config:    data.Settings.Configs[data.Latest.Config],
		CacheMode: fetch.CacheTemp,
	}, data.Latest.Info.Hash)
	IfFatal(err)

	// Generate entities.
	data.Entities = GenerateEntities(data.Patches)
	data.TreeRoots = GenerateTree(data.Entities.Classes)

	// Compile templates.
	data.Templates, err = CompileTemplates(data.Settings.Input.Templates, template.FuncMap{
		"cards":   data.GenerateCardElements,
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
		"pack":       PackValues,
		"patchtype":  PatchTypeString,
		"quantity":   FormatQuantity,
		"resources":  data.GenerateResourceElements,
		"sortedlist": SortedList,
		"subactions": MakeSubactions,
		"tolower":    strings.ToLower,
		"tostring":   ToString,
		"type":       GetType,
		"unpack":     UnpackValues,
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

		// Filter pages.
		if len(filters) > 0 {
			p := pages[:0]
			for _, page := range pages {
				if page.File == "" {
					p = append(p, page)
					continue
				}
				name := path.Clean(strings.Replace(page.File, "\\", "/", -1))
				for i, filter := range filters {
					for dir, file := name, ""; ; {
						file = path.Join(path.Base(dir), file)
						if ok, err := path.Match(filter, file); ok && err == nil {
							p = append(p, page)
							break
						} else {
							IfFatalf(err, "filter #%d", i)
						}
						dir = path.Dir(dir)
						if dir == "." || dir == "/" || dir == "" {
							break
						}
					}
				}
			}
			pages = p
			for _, page := range pages {
				Log("INCLUDE PAGE", page.File)
			}
		}

		// Ensure directories exist.
		dirs := map[string]struct{}{}
		for _, page := range pages {
			dir := filepath.Join(data.Settings.Output.Root, filepath.Dir(page.File))
			if _, ok := dirs[dir]; ok {
				continue
			}
			IfFatal(os.MkdirAll(dir, 0755), "make directory")
			dirs[dir] = struct{}{}
		}

		// Copy resources.
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
			dstname := data.AbsFilePath("resource", name)
			{
				dir := filepath.Dir(dstname)
				if _, ok := dirs[dir]; !ok {
					IfFatal(os.MkdirAll(dir, 0755), "make directory")
					dirs[dir] = struct{}{}
				}
			}
			dst, err := os.Create(dstname)
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
			file, err := os.Create(filepath.Join(data.Settings.Output.Root, page.File))
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
		f, err := os.Create(data.AbsFilePath("search"))
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
