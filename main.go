package main

import (
	"bytes"
	"fmt"
	"github.com/anaminus/but"
	"github.com/jessevdk/go-flags"
	"html/template"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

type FlagOptions struct {
	Settings string `short:"s" long:"settings"`
	Force    bool   `long:"force"`
}

var options = map[string]*flags.Option{
	"settings": &flags.Option{
		Description: "Specify a custom settings location.",
		ValueName:   "PATH",
	},
	"force": &flags.Option{
		Description: "Force a complete rebuild.",
	},
}

func ParseOptions(data interface{}, opts flags.Options) *flags.Parser {
	fp := flags.NewParser(data, opts)
	for name, info := range options {
		opt := fp.FindOptionByLongName(name)
		if opt == nil {
			continue
		}
		opt.Description = info.Description
		opt.ValueName = info.ValueName
	}
	return fp
}

func main() {
	var err error

	// Parse flags.
	var opt FlagOptions
	var filters []string
	{
		fp := ParseOptions(&opt, flags.Default|flags.PassAfterNonOption)
		var err error
		filters, err = fp.Parse()
		if err != nil {
			if err, ok := err.(*flags.Error); ok && err.Type == flags.ErrHelp {
				fmt.Fprintln(os.Stdout, err)
				return
			}
		}
		but.IfFatal(err, "flag parser error")
	}

	// Initialize root.
	data := &Data{
		CurrentYear: time.Now().Year(),
		Manifest:    &Manifest{},
	}

	// Load settings.
	data.Settings = *DefaultSettings.Copy()
	but.IfFatal(data.Settings.ReadFile(opt.Settings))

	// Load manifest.
	manifestPath := data.AbsFilePath("manifest")
	if !opt.Force {
		if b, err := ioutil.ReadFile(manifestPath); err == nil {
			data.Manifest, err = ReadManifest(bytes.NewReader(b))
			but.IfFatal(err, "read manifest")
		}
	}

	// Fetch builds.
	builds, err := FetchBuilds(data.Settings)
	but.IfFatal(err)

	// Merge uncached builds.
	data.Manifest.Patches, err = MergeBuilds(data.Settings, data.Manifest.Patches, builds)
	but.IfFatal(err)

	// Generate entities.
	data.Entities = GenerateEntities(data.Manifest.Patches)
	but.IfFatal(data.GenerateMetadata())
	data.GenerateDocuments()

	// Compile templates.
	data.Templates, err = CompileTemplates(data.Settings.Input.Templates, template.FuncMap{
		"cards":    data.GenerateCardElements,
		"document": QueryDocument,
		"embed":    data.EmbedResource,
		"execute":  data.ExecuteTemplate,
		"filter":   FilterList,
		"history":  data.GenerateHistoryElements,
		"icon":     data.Icon,
		"istype":   IsType,
		"last":     LastIndex,
		"list":     ParseStringList,
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
		"status":     data.ElementStatusClasses,
		"subactions": MakeSubactions,
		"tolower":    strings.ToLower,
		"tostring":   ToString,
		"type":       GetType,
		"unpack":     UnpackValues,
	})
	but.IfFatal(err, "open template")

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
	but.IfFatal(data.ComparePages(pages), "remove unreferenced file")
	if len(filters) > 0 {
		pages, err = FilterPages(pages, filters)
		but.IfFatal(err)
		for _, page := range pages {
			but.Log("INCLUDE PAGE", page.File)
		}
	}
	but.IfFatal(data.RenderPageDirs(pages))
	but.IfFatal(data.RenderResources(pages))
	but.IfFatal(data.RenderPages(pages))

	// Generate search database.
	{
		f, err := os.Create(data.AbsFilePath("search"))
		but.IfFatal(err, "create search database file")
		db := dbWriter{data: data, w: f}
		db.GenerateDatabase()
		f.Close()
		but.IfFatal(db.err, "generate search database")
	}

	// Save manifest.
	{
		var buf bytes.Buffer
		err := WriteManifest(&buf, data.Manifest)
		but.IfFatal(err, "encode manifest")

		f, err := os.Create(manifestPath)
		but.IfFatal(err, "create manifest")
		defer f.Close()

		_, err = buf.WriteTo(f)
		but.IfFatal(err, "write manifest")

		err = f.Sync()
		but.IfFatal(err, "sync manifest")
	}
}
