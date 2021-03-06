package main

import (
	"bytes"
	"fmt"
	"html/template"
	"image/png"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/anaminus/but"
	"github.com/robloxapi/rbxapiref/builds"
	"github.com/robloxapi/rbxapiref/documents"
	"github.com/robloxapi/rbxapiref/entities"
	"github.com/robloxapi/rbxapiref/fetch"
	"github.com/robloxapi/rbxapiref/settings"
)

type Page struct {
	// File is the path to the output file.
	File string
	// Meta is a set of extra metadata about the page.
	Meta Meta
	// Styles is a list of resources representing CSS styles.
	Styles []Resource
	// Scripts is a list of resources representing javascript files.
	Scripts []Resource
	// Resources is a list of other resources.
	Resources []Resource
	// DocResources is a list of document resources.
	DocResources []Resource
	// Template is the name of the template used to generate the page.
	Template string
	// Data is the data used by the template to generate the page.
	Data interface{}
}

type Meta map[string]string

type Attr struct {
	Name  template.HTMLAttr
	Value string
}

type Attrs []Attr

func (a Attrs) Find(name string) *Attr {
	for i := range a {
		if string(a[i].Name) == name {
			return &a[i]
		}
	}
	return nil
}

func (a *Attrs) Merge(b Attrs) {
	for _, attrb := range b {
		if attra := a.Find(string(attrb.Name)); attra != nil {
			attra.Value = attrb.Value
			continue
		}
		*a = append(*a, attrb)
	}
}

type Resource struct {
	// Name indicates the name of the source file located in the input
	// resource directory, as well as the name of the generated file within
	// the output resource directory.
	Name string
	// Content, if non-nil, specifies the content of the file directly, rather
	// than reading from a source file.
	Content []byte
	// Embed causes the content of the resource to be embedded within a
	// generated page, rather than being written to the output resource
	// directory.
	Embed bool
	// Attr contains additional attributes of the generated HTML node
	// representing the resource.
	Attr Attrs
	// Ignore allows the resource to exist, but otherwise be ignored when
	// copying. This will prevent the resource destination from being deleted.
	Ignore bool
}

func Title(sub string) string {
	if sub != "" {
		return sub + " " + settings.TitleSep + " " + settings.MainTitle
	}
	return settings.MainTitle
}

func FilterPages(pages []Page, filters []string) ([]Page, error) {
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
				} else if err != nil {
					return nil, fmt.Errorf("filter #%d: %w", i, err)
				}
				dir = path.Dir(dir)
				if dir == "." || dir == "/" || dir == "" {
					break
				}
			}
		}
	}
	return p, nil
}

////////////////////////////////////////////////////////////////

func generatePageMain(data *Data) (pages []Page) {
	page := Page{
		Meta: Meta{
			"Title":       settings.MainTitle,
			"Description": "Reference for the Roblox Lua API.",
			"Image":       "favicons/favicon-512x512.png",
		},
		Styles: []Resource{
			{Name: "theme-light.css"},
			{Name: "theme-dark.css"},
			{Name: "main.css"},
			{Name: "doc.css"},
		},
		Scripts: []Resource{
			{Name: "quick-theme.js", Embed: true},
			{Name: "main.js", Attr: Attrs{{"async", ""}}},
			{Name: "search.js", Attr: Attrs{{"async", ""}}},
			{Name: "settings.js", Attr: Attrs{{"async", ""}}},
			{Name: "actions.js", Attr: Attrs{{"async", ""}}},
		},
		Template: "main",
	}
	if data.ResOnly {
		page.Resources = append(page.Resources,
			Resource{Name: "icon-explorer.png", Ignore: true},
		)
	} else {
		// Fetch explorer icons.
		latest := data.Manifest.Patches[len(data.Manifest.Patches)-1]
		client := &fetch.Client{
			Config:    data.Settings.Build.Configs[latest.Config],
			CacheMode: fetch.CacheTemp,
		}
		icon, err := client.ExplorerIcons(latest.Info.Hash)
		but.IfFatalf(err, "%s: fetch icons", latest.Info.Hash)
		var buf bytes.Buffer
		but.IfFatal(png.Encode(&buf, icon), "encode icons file")
		page.Resources = append(page.Resources,
			Resource{Name: "icon-explorer.png", Content: buf.Bytes()},
		)
	}
	page.Resources = append(page.Resources,
		Resource{Name: "icon-objectbrowser.png"},
		Resource{Name: "icon-devhub.png"},
		Resource{Name: "favicons/favicon-512x512.png"},
		Resource{Name: "favicons/favicon-32x32.png"},
		Resource{Name: "favicons/favicon-16x16.png"},
		Resource{Name: "favicons/favicon.ico"},
	)
	return []Page{page}
}

func generatePageIndex(output settings.Output) (pages []Page) {
	return []Page{{
		File:     output.FilePath("index"),
		Styles:   []Resource{{Name: "index.css", Embed: true}},
		Scripts:  []Resource{{Name: "index.js", Attr: []Attr{{"async", ""}}}},
		Template: "index",
	}}
}

func generatePageAbout(output settings.Output) (pages []Page) {
	return []Page{{
		File: output.FilePath("about"),
		Meta: Meta{
			"Title":       Title("About"),
			"Description": "About the Roblox API Reference.",
		},
		Styles: []Resource{{Name: "about.css", Embed: true}},
		Resources: []Resource{
			{Name: "license-badge.png"},
		},
		Template: "about",
	}}
}

func generatePageDocmon(output settings.Output, entities *entities.Entities) (pages []Page) {
	return []Page{{
		File: output.FilePath("docmon"),
		Meta: Meta{
			"Title":       Title("Documentation monitor"),
			"Description": "Status of documentation on the Roblox API Reference.",
		},
		Styles:   []Resource{{Name: "docmon.css", Embed: true}},
		Scripts:  []Resource{{Name: "docmon.js", Attr: []Attr{{"async", ""}}}},
		Template: "docmon",
		Data:     entities,
	}}
}

func generatePageUpdates(output settings.Output, patches []builds.Patch) (pages []Page) {
	if len(patches) <= 1 {
		return nil
	}

	// Patches are displayed latest-first.
	patchlist := make([]*builds.Patch, len(patches))
	for i := len(patches) / 2; i >= 0; i-- {
		j := len(patches) - 1 - i
		patchlist[i], patchlist[j] = &patches[j], &patches[i]
	}
	// Exclude earliest patch.
	patchlist = patchlist[:len(patchlist)-1]

	type PatchSet struct {
		Year    int
		Years   []int
		Patches []*builds.Patch
	}

	var latestPatches PatchSet
	latestYear := patchlist[0].Info.Date.Year()
	earliestYear := patchlist[len(patchlist)-1].Info.Date.Year()
	patchesByYear := make([]PatchSet, latestYear-earliestYear+1)
	years := make([]int, len(patchesByYear))
	for i := range years {
		years[i] = latestYear - i
	}

	{
		// Populate patchesByYear.
		i := 0
		current := latestYear
		for j, patch := range patchlist {
			year := patch.Info.Date.Year()
			if year < current {
				if j > i {
					patchesByYear[latestYear-current] = PatchSet{
						Year:    current,
						Years:   years,
						Patches: patchlist[i:j],
					}
				}
				current = year
				i = j
			}
		}
		if len(patchlist) > i {
			patchesByYear[latestYear-current] = PatchSet{
				Year:    current,
				Years:   years,
				Patches: patchlist[i:],
			}
		}

		// Populate latestPatches.
		i = len(patchlist)
		epoch := patchlist[0].Info.Date.AddDate(0, -3, 0)
		for j, patch := range patchlist {
			if patch.Info.Date.Before(epoch) {
				i = j - 1
				break
			}
		}
		latestPatches = PatchSet{
			Years:   years,
			Patches: patchlist[:i],
		}
	}

	styles := []Resource{{Name: "updates.css", Attr: []Attr{{"id", "updates-style"}}}}
	scripts := []Resource{{Name: "updates.js", Attr: []Attr{{"async", ""}}}}
	pages = make([]Page, len(patchesByYear)+1)
	for i, patches := range patchesByYear {
		year := strconv.Itoa(patches.Year)
		pages[i] = Page{
			File: output.FilePath("updates", year),
			Meta: Meta{
				"Title":       Title("Updates in " + year),
				"Description": "A list of updates to the Roblox Lua API in " + year + ".",
			},
			Styles:   styles,
			Scripts:  scripts,
			Template: "updates",
			Data:     patches,
		}
	}
	pages[len(pages)-1] = Page{
		File: output.FilePath("updates"),
		Meta: Meta{
			"Title":       Title("Recent Updates"),
			"Description": "A list of recent updates to the Roblox Lua API."},
		Styles:   styles,
		Scripts:  scripts,
		Template: "updates",
		Data:     latestPatches,
	}
	return pages
}

// Normalizes the references within a document according to ParseDocReference,
// and returns any resources that the document refers to.
func NormalizeDocReferences(output settings.Output, document entities.Document) []Resource {
	doc, ok := document.(documents.Linkable)
	if !ok {
		return nil
	}
	resources := map[string]*Resource{}
	doc.SetLinks(func(link string) string {
		scheme, path, link := output.ParseDocReference(link)
		if scheme == "res" {
			if _, ok := resources[path]; !ok {
				resources[path] = &Resource{Name: path}
			}
		}
		return link
	})
	docres := make([]Resource, 0, len(resources))
	for _, resource := range resources {
		docres = append(docres, *resource)
	}
	sort.Slice(docres, func(i, j int) bool {
		return docres[i].Name < docres[j].Name
	})
	return docres
}

func generatePageClass(output settings.Output, classes []*entities.Class) (pages []Page) {
	styles := []Resource{{Name: "class.css"}}
	scripts := []Resource{{Name: "class.js", Attr: []Attr{{"async", ""}}}}
	pages = make([]Page, len(classes))
	for i, class := range classes {
		pages[i] = Page{
			File: output.FilePath("class", class.ID),
			Meta: Meta{
				"Title":       Title(class.ID),
				"Description": "Information about the " + class.ID + " class in the Roblox Lua API."},
			Styles:       styles,
			Scripts:      scripts,
			DocResources: NormalizeDocReferences(output, class.Document),
			Template:     "class",
			Data:         class,
		}
	}
	return pages
}

func generatePageEnum(output settings.Output, enums []*entities.Enum) (pages []Page) {
	styles := []Resource{{Name: "enum.css"}}
	scripts := []Resource{{Name: "enum.js", Attr: []Attr{{"async", ""}}}}
	pages = make([]Page, len(enums))
	for i, enum := range enums {
		pages[i] = Page{
			File: output.FilePath("enum", enum.ID),
			Meta: Meta{
				"Title":       Title(enum.ID),
				"Description": "Information about the " + enum.ID + " enum in the Roblox Lua API."},
			Styles:       styles,
			Scripts:      scripts,
			DocResources: NormalizeDocReferences(output, enum.Document),
			Template:     "enum",
			Data:         enum,
		}
	}
	return pages
}

func generatePageType(output settings.Output, types []*entities.Type) (pages []Page) {
	styles := []Resource{{Name: "type.css"}}
	scripts := []Resource{{Name: "type.js", Attr: []Attr{{"async", ""}}}}
	pages = make([]Page, len(types))
	for i, typ := range types {
		pages[i] = Page{
			File: output.FilePath("type", typ.Element.Category, typ.Element.Name),
			Meta: Meta{
				"Title":       Title(typ.ID),
				"Description": "Information about the " + typ.ID + " type in the Roblox Lua API."},
			Styles:       styles,
			Scripts:      scripts,
			DocResources: NormalizeDocReferences(output, typ.Document),
			Template:     "type",
			Data:         typ,
		}
	}
	return pages
}

func GeneratePages(data *Data) (pages []Page) {
	pages = append(pages, generatePageMain(data)...)
	pages = append(pages, generatePageIndex(data.Settings.Output)...)
	pages = append(pages, generatePageAbout(data.Settings.Output)...)
	pages = append(pages, generatePageDocmon(data.Settings.Output, data.Entities)...)
	pages = append(pages, generatePageUpdates(data.Settings.Output, data.Manifest.Patches)...)
	pages = append(pages, generatePageClass(data.Settings.Output, data.Entities.ClassList)...)
	pages = append(pages, generatePageEnum(data.Settings.Output, data.Entities.EnumList)...)
	pages = append(pages, generatePageType(data.Settings.Output, data.Entities.TypeList)...)
	return pages
}
