package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/robloxapi/rbxapiref/fetch"
	"html/template"
	"os"
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
	if f, err := os.Open(manifestPath); err == nil {
		data.Manifest, err = ReadManifest(f)
		f.Close()
		IfFatal(err, "open manifest")
	} else {
		data.Manifest = &Manifest{}
	}

	// Fetch builds.
	builds, err := FetchBuilds(data.Settings)
	IfFatal(err)

	// Merge uncached builds.
	data.Manifest.Patches, err = MergeBuilds(data.Settings, data.Manifest.Patches, builds)
	IfFatal(err)

	// Fetch ReflectionMetadata.
	{
		latest := data.LatestPatch()
		data.Metadata, err = GenerateMetadata(&fetch.Client{
			Config:    data.Settings.Configs[latest.Config],
			CacheMode: fetch.CacheTemp,
		}, latest.Info.Hash)
		IfFatal(err)
	}

	// Generate entities.
	data.Entities = GenerateEntities(data.Manifest.Patches)
	data.GenerateDocuments()

	// Compile templates.
	data.Templates, err = CompileTemplates(data.Settings.Input.Templates, template.FuncMap{
		"cards":   data.GenerateCardElements,
		"embed":   data.EmbedResource,
		"execute": data.ExecuteTemplate,
		"filter":  FilterList,
		"history": data.GenerateHistoryElements,
		"icon":    data.Icon,
		"istype":  IsType,
		"last":    LastIndex,
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
	pages := data.GeneratePages([]PageGenerator{
		GeneratePageMain,
		GeneratePageIndex,
		GeneratePageAbout,
		GeneratePageUpdates,
		GeneratePageClass,
		GeneratePageEnum,
		GeneratePageType,
	})
	if len(filters) > 0 {
		pages, err = FilterPages(pages, filters)
		IfFatal(err)
		for _, page := range pages {
			Log("INCLUDE PAGE", page.File)
		}
	}
	IfFatal(data.RenderPageDirs(pages))
	IfFatal(data.RenderResources(pages))
	IfFatal(data.RenderPages(pages))

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
		err = WriteManifest(f, data.Manifest)
		f.Close()
		IfFatal(err, "encode manifest")
	}
}
