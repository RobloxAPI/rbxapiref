package main

import (
	"bytes"
	"github.com/anaminus/but"
	"github.com/pkg/errors"
	"github.com/robloxapi/rbxapiref/fetch"
	"html/template"
	"image/png"
	"path"
	"strconv"
	"strings"
)

type PageGenerator func(*Data) []Page

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
}

func Title(sub string) string {
	if sub != "" {
		return sub + " " + TitleSep + " " + MainTitle
	}
	return MainTitle
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
					return nil, errors.WithMessagef(err, "filter #%d", i)
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

func GeneratePageMain(data *Data) (pages []Page) {
	// Fetch explorer icons.
	latest := data.LatestPatch()
	client := &fetch.Client{
		Config:    data.Settings.Configs[latest.Config],
		CacheMode: fetch.CacheTemp,
	}
	icon, err := client.ExplorerIcons(latest.Info.Hash)
	but.IfFatalf(err, "%s: fetch icons %s", latest.Info.Hash)
	var buf bytes.Buffer
	but.IfFatal(png.Encode(&buf, icon), "encode icons file")

	page := Page{
		Meta: Meta{
			"Title":       MainTitle,
			"Description": "Reference for the Roblox Lua API.",
			"Image":       "favicons/favicon-512x512.png",
		},
		Styles: []Resource{
			{Name: "main.css"},
			{Name: "doc.css"},
		},
		Scripts: []Resource{
			{Name: "main.js", Attr: Attrs{{"async", ""}}},
			{Name: "search.js", Attr: Attrs{{"async", ""}}},
			{Name: "settings.js", Attr: Attrs{{"async", ""}}},
		},
		Resources: []Resource{
			{Name: "icon-explorer.png", Content: buf.Bytes()},
			{Name: "icon-objectbrowser.png"},
			{Name: "icon-devhub.png"},
			{Name: "settings.svg"},
			{Name: "favicons/favicon-512x512.png"},
			{Name: "favicons/favicon-32x32.png"},
			{Name: "favicons/favicon-16x16.png"},
			{Name: "favicons/favicon.ico"},
		},
		Template: "main",
	}
	if data.CodeFormatter != nil && data.CodeStyle != nil {
		var buf strings.Builder
		buf.WriteString("/* Style: " + data.CodeStyle.Name + " */\n")
		if err := data.CodeFormatter.WriteCSS(&buf, data.CodeStyle); err == nil {
			page.Styles = append(page.Styles, Resource{
				Name:    "syntax-" + data.CodeStyle.Name + ".css",
				Content: []byte(buf.String()),
			})
		}
	}
	return []Page{page}
}

func GeneratePageIndex(data *Data) (pages []Page) {
	return []Page{{
		File:     data.FilePath("index"),
		Styles:   []Resource{{Name: "index.css", Embed: true}},
		Scripts:  []Resource{{Name: "sort-classes.js", Attr: []Attr{{"async", ""}}}},
		Template: "index",
	}}
}

func GeneratePageAbout(data *Data) (pages []Page) {
	return []Page{{
		File: data.FilePath("about"),
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

func GeneratePageUpdates(data *Data) (pages []Page) {
	if len(data.Manifest.Patches) < 2 {
		return nil
	}

	// Patches will be displayed latest-first.
	patches := make([]*Patch, len(data.Manifest.Patches))
	for i := len(data.Manifest.Patches) / 2; i >= 0; i-- {
		j := len(data.Manifest.Patches) - 1 - i
		patches[i], patches[j] = &data.Manifest.Patches[j], &data.Manifest.Patches[i]
	}
	// Exclude earliest patch.
	patches = patches[:len(patches)-1]

	type PatchSet struct {
		Year    int
		Years   []int
		Patches []*Patch
	}

	var latestPatches PatchSet
	latestYear := patches[0].Info.Date.Year()
	earliestYear := patches[len(patches)-1].Info.Date.Year()
	patchesByYear := make([]PatchSet, latestYear-earliestYear+1)
	years := make([]int, len(patchesByYear))
	for i := range years {
		years[i] = latestYear - i
	}

	{
		// Populate patchesByYear.
		i := 0
		current := latestYear
		for j, patch := range patches {
			year := patch.Info.Date.Year()
			if year < current {
				if j > i {
					patchesByYear[latestYear-current] = PatchSet{
						Year:    current,
						Years:   years,
						Patches: patches[i:j],
					}
				}
				current = year
				i = j
			}
		}
		if len(patches) > i {
			patchesByYear[latestYear-current] = PatchSet{
				Year:    current,
				Years:   years,
				Patches: patches[i:],
			}
		}

		// Populate latestPatches.
		i = len(patches)
		epoch := patches[0].Info.Date.AddDate(0, -3, 0)
		for j, patch := range patches {
			if patch.Info.Date.Before(epoch) {
				i = j - 1
				break
			}
		}
		latestPatches = PatchSet{
			Years:   years,
			Patches: patches[:i],
		}
	}

	styles := []Resource{{Name: "updates.css", Attr: []Attr{{"id", "updates-style"}}}}
	scripts := []Resource{{Name: "updates.js", Attr: []Attr{{"async", ""}}}}
	pages = make([]Page, len(patchesByYear)+1)
	for i, patches := range patchesByYear {
		year := strconv.Itoa(patches.Year)
		pages[i] = Page{
			File: data.FilePath("updates", year),
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
		File: data.FilePath("updates"),
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

func GeneratePageClass(data *Data) (pages []Page) {
	styles := []Resource{{Name: "class.css"}}
	scripts := []Resource{{Name: "class.js", Attr: []Attr{{"async", ""}}}}
	pages = make([]Page, len(data.Entities.ClassList))
	for i, class := range data.Entities.ClassList {
		pages[i] = Page{
			File: data.FilePath("class", class.ID),
			Meta: Meta{
				"Title":       Title(class.ID),
				"Description": "Information about the " + class.ID + " class in the Roblox Lua API."},
			Styles:       styles,
			Scripts:      scripts,
			DocResources: data.NormalizeDocReferences(class.Document),
			Template:     "class",
			Data:         class,
		}
	}
	return pages
}

func GeneratePageEnum(data *Data) (pages []Page) {
	styles := []Resource{{Name: "enum.css"}}
	pages = make([]Page, len(data.Entities.EnumList))
	for i, enum := range data.Entities.EnumList {
		pages[i] = Page{
			File: data.FilePath("enum", enum.ID),
			Meta: Meta{
				"Title":       Title(enum.ID),
				"Description": "Information about the " + enum.ID + " enum in the Roblox Lua API."},
			Styles:       styles,
			DocResources: data.NormalizeDocReferences(enum.Document),
			Template:     "enum",
			Data:         enum,
		}
	}
	return pages
}

func GeneratePageType(data *Data) (pages []Page) {
	styles := []Resource{{Name: "type.css"}}
	pages = make([]Page, len(data.Entities.TypeList))
	for i, typ := range data.Entities.TypeList {
		pages[i] = Page{
			File: data.FilePath("type", typ.Element.Category, typ.Element.Name),
			Meta: Meta{
				"Title":       Title(typ.ID),
				"Description": "Information about the " + typ.ID + " type in the Roblox Lua API."},
			Styles:       styles,
			DocResources: data.NormalizeDocReferences(typ.Document),
			Template:     "type",
			Data:         typ,
		}
	}
	return pages
}
