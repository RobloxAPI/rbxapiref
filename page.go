package main

import (
	"strconv"
)

type PageGenerator func(*Data) []Page

type Page struct {
	File     string
	Title    string
	Styles   []Resource
	Scripts  []Resource
	Template string
	Data     interface{}
}

type Resource struct {
	Name  string // Name of the resource file.
	Embed bool   // Embed the content of the resource.
	ID    string // Optional ID attribute.
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
