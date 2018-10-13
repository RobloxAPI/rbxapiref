package main

import (
	"strconv"
)

type PageGenerator func(*Data) []Page

type Page struct {
	// File is the path to the output file.
	File string
	// Title is the text to be displayed in the title of the page.
	Title string
	// Styles is a list of resources representing CSS styles.
	Styles []Resource
	// Scripts is a list of resources representing javascript files.
	Scripts []Resource
	// Resources is a list of other resources.
	Resources []Resource
	// Template is the name of the template used to generate the page.
	Template string
	// Data is the data used by the template to generate the page.
	Data interface{}
}

type Resource struct {
	Name  string // Name of the resource file.
	Embed bool   // Embed the content of the resource.
	ID    string // Optional ID attribute.
}

func GeneratePageMain(data *Data) (pages []Page) {
	return []Page{{
		Styles:    []Resource{{Name: "main.css"}},
		Scripts:   []Resource{{Name: "search.js"}},
		Resources: []Resource{{Name: "icon-objectbrowser.png"}},
		Template:  "main",
	}}
}

func GeneratePageIndex(data *Data) (pages []Page) {
	return []Page{{
		File:     data.FilePath("index"),
		Styles:   []Resource{{Name: "index.css", Embed: true}},
		Scripts:  []Resource{{Name: "sort-classes.js"}},
		Template: "index",
	}}
}

func GeneratePageAbout(data *Data) (pages []Page) {
	return []Page{{
		File:     data.FilePath("about"),
		Title:    "About",
		Styles:   []Resource{{Name: "about.css", Embed: true}},
		Template: "about",
	}}
}

func GeneratePageUpdates(data *Data) (pages []Page) {
	if len(data.LatestPatches.Patches) == 0 {
		return nil
	}

	styles := []Resource{{Name: "updates.css", ID: "updates-style"}}
	scripts := []Resource{{Name: "updates.js"}}
	pages = make([]Page, len(data.PatchesByYear)+1)
	for i, patches := range data.PatchesByYear {
		year := strconv.Itoa(patches.Year)
		pages[i] = Page{
			File:     data.FilePath("updates", year),
			Title:    "Updates in " + year,
			Styles:   styles,
			Scripts:  scripts,
			Template: "updates",
			Data:     patches,
		}
	}
	pages[len(pages)-1] = Page{
		File:     data.FilePath("updates"),
		Title:    "Recent Updates",
		Styles:   styles,
		Scripts:  scripts,
		Template: "updates",
		Data:     data.LatestPatches,
	}
	return pages
}

func GeneratePageClass(data *Data) (pages []Page) {
	styles := []Resource{{Name: "class.css"}}
	scripts := []Resource{{Name: "class.js"}}
	pages = make([]Page, len(data.Entities.ClassList))
	for i, class := range data.Entities.ClassList {
		pages[i] = Page{
			File:     data.FilePath("class", class.ID),
			Title:    class.ID,
			Styles:   styles,
			Scripts:  scripts,
			Template: "class",
			Data:     class,
		}
	}
	return pages
}

func GeneratePageEnum(data *Data) (pages []Page) {
	styles := []Resource{{Name: "enum.css"}}
	pages = make([]Page, len(data.Entities.EnumList))
	for i, enum := range data.Entities.EnumList {
		pages[i] = Page{
			File:     data.FilePath("enum", enum.ID),
			Title:    enum.ID,
			Styles:   styles,
			Template: "enum",
			Data:     enum,
		}
	}
	return pages
}

func GeneratePageType(data *Data) (pages []Page) {
	pages = make([]Page, len(data.Entities.TypeList))
	for i, typ := range data.Entities.TypeList {
		pages[i] = Page{
			File:     data.FilePath("type", typ.Element.Category, typ.Element.Name),
			Title:    typ.ID,
			Template: "type",
			Data:     typ,
		}
	}
	return pages
}
