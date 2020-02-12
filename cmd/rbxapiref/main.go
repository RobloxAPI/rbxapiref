package main

import (
	"bufio"
	"bytes"
	"html/template"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/anaminus/but"
	"github.com/jessevdk/go-flags"
	"github.com/robloxapi/rbxapiref/builds"
	"github.com/robloxapi/rbxapiref/entities"
	"github.com/robloxapi/rbxapiref/manifest"
	"github.com/robloxapi/rbxapiref/settings"
)

type Range struct {
	Min, Max int
	Count    int
}

func (r *Range) UnmarshalFlag(flag string) (err error) {
	sep := strings.Index(flag, ":")
	if sep < 0 {
		r.Min, err = strconv.Atoi(flag)
		r.Count = 1
		return err
	}

	if r.Min, err = strconv.Atoi(flag[:sep]); err != nil {
		return err
	}
	if r.Max, err = strconv.Atoi(flag[sep+1:]); err != nil {
		return err
	}
	r.Count = 2
	return nil
}

func (r Range) Norm(length int) Range {
	if r.Count == 1 {
		r.Max = length
	}
	if r.Min < 0 {
		r.Min = length + r.Min
	} else if r.Min > length {
		r.Min = length
	}
	if r.Max < 0 {
		r.Max = length + r.Max
	} else if r.Max > length {
		r.Max = length
	}
	if r.Min > r.Max {
		r.Min, r.Max = 0, 0
	}
	r.Count = 2
	return r
}

type FlagOptions struct {
	Settings string `short:"s" long:"settings"`
	Force    bool   `long:"force"`
	ResOnly  bool   `long:"res-only"`
	Range    Range  `long:"range"`
	UseGit   bool   `long:"use-git"`
	NoGit    bool   `long:"no-git"`
	Rewind   bool   `long:"rewind"`
	NoRewind bool   `long:"no-rewind"`
}

var options = map[string]*flags.Option{
	"settings": &flags.Option{
		Description: "Specify a custom settings location.",
		ValueName:   "PATH",
	},
	"force": &flags.Option{
		Description: "Force a complete rebuild.",
	},
	"res-only": &flags.Option{
		Description: "Only regenerate resource files.",
	},
	"range": &flags.Option{
		Description: "Select a range of builds.",
		ValueName:   "RANGE",
	},
	"use-git": &flags.Option{
		Description: "Force git-aware document parsing.",
	},
	"no-git": &flags.Option{
		Description: "Force git-unaware document parsing.",
	},
	"rewind": &flags.Option{
		Description: "Force rewinding.",
	},
	"no-rewind": &flags.Option{
		Description: "Force no rewinding.",
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
		if err, ok := err.(*flags.Error); ok && err.Type == flags.ErrHelp {
			return
		}
		but.IfFatal(err, "flag parser error")
	}

	// Initialize root.
	data := &Data{
		Time:     time.Now(),
		Manifest: &manifest.Manifest{},
		ResOnly:  opt.ResOnly,
	}

	// Load settings.
	data.Settings = *settings.Default.Copy()
	but.IfFatal(data.Settings.ReadFile(opt.Settings))
	if opt.NoGit {
		data.Settings.Input.UseGit = false
	} else if opt.UseGit {
		data.Settings.Input.UseGit = true
	}
	if opt.NoRewind {
		data.Settings.Build.DisableRewind = true
	} else if opt.Rewind {
		data.Settings.Build.DisableRewind = false
	}

	// Load manifest.
	manifestPath := data.Settings.Output.AbsFilePath("manifest")
	if !opt.Force {
		if b, err := ioutil.ReadFile(manifestPath); err == nil {
			data.Manifest, err = manifest.Decode(bytes.NewReader(b))
			but.IfFatal(err, "read manifest")
		}
	}

	if !opt.ResOnly {
		// Fetch builds.
		builds, err := data.Settings.Build.Fetch()
		but.IfFatal(err)

		if opt.Range.Count > 0 {
			r := opt.Range.Norm(len(builds))
			builds = builds[r.Min:r.Max]
		}

		// Merge uncached builds.
		data.Manifest.Patches, err = data.Settings.Build.Merge(data.Manifest.Patches, builds)
		but.IfFatal(err)
	}

	// Generate entities.
	data.Entities = entities.GenerateEntities(data.Manifest.Patches)
	but.IfFatal(data.GenerateMetadata())
	data.GenerateDocuments()

	if !opt.ResOnly {
		// Compile templates.
		data.Templates, err = CompileTemplates(data.Settings.Input.Templates, template.FuncMap{
			"cards":     data.GenerateCardElements,
			"docstatus": GetDocStatus,
			"document":  entities.QueryDocument,
			"embed":     data.EmbedResource,
			"execute":   data.ExecuteTemplate,
			"filter":    FilterList,
			"history":   data.GenerateHistoryElements,
			"icon":      data.Icon,
			"istype":    IsType,
			"last":      LastIndex,
			"list":      ParseStringList,
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
				return data.Settings.Output.FileLink(linkType, sargs...)
			},
			"pack":       PackValues,
			"patchtype":  builds.PatchTypeString,
			"quantity":   FormatQuantity,
			"renderdoc":  entities.RenderDocument,
			"resources":  data.GenerateResourceElements,
			"sortedlist": SortedList,
			"status":     data.ElementStatusClasses,
			"subactions": builds.MakeSubactions,
			"tolower":    strings.ToLower,
			"tostring":   ToString,
			"type":       GetType,
			"unpack":     UnpackValues,
		})
		but.IfFatal(err, "open template")
	}

	// Generate pages.
	pages := data.GeneratePages([]PageGenerator{
		GeneratePageMain,
		GeneratePageIndex,
		GeneratePageAbout,
		GeneratePageDocmon,
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
	if opt.ResOnly {
		return
	}
	but.IfFatal(data.RenderPages(pages))

	// Generate search database.
	{
		f, err := os.Create(data.Settings.Output.AbsFilePath("search"))
		but.IfFatal(err, "create search database file")
		w := bufio.NewWriter(f)
		but.IfFatal(GenerateDatabase(w, data.Entities), "generate search database")
		but.IfFatal(w.Flush(), "write search database")
		f.Close()
	}

	// Save manifest.
	{
		var buf bytes.Buffer
		err := manifest.Encode(&buf, data.Manifest)
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
