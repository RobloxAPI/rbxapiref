package main

import (
	"os"
	"path/filepath"
)

func GenerateClassPages(data *Data) error {
	type ClassPageData struct {
		Name   string
		Entity *ClassEntity
	}

	page := Page{
		CurrentYear: data.CurrentYear,
		Template:    "class",
		Styles:      []Resource{{Name: "class.css"}},
		Scripts:     []Resource{{Name: "class.js"}},
	}
	if err := os.MkdirAll(filepath.Dir(data.FilePath("class", "class")), 0666); err != nil {
		return err
	}
	for _, class := range data.Entities.ClassList {
		path := data.FilePath("class", class.ID)
		file, err := os.Create(path)
		if err != nil {
			return err
		}

		page.Title = class.ID
		pageData := &ClassPageData{Name: class.ID, Entity: class}
		page.Data = pageData

		err = GeneratePage(data, file, page)
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
