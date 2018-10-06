package main

import (
	"os"
	"path/filepath"
)

func GenerateClassPages(data *Data) error {
	type ClassPageData struct {
		Name         string
		Entity       *ClassEntity
		Superclasses []*ClassEntity
		Subclasses   []*ClassEntity
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
		if tree := data.Tree[class.ID]; tree != nil {
			for _, class := range tree.Super {
				if entity := data.Entities.Classes[class]; entity != nil {
					pageData.Superclasses = append(pageData.Superclasses, entity)
				}
			}
			for _, class := range tree.Sub {
				if entity := data.Entities.Classes[class]; entity != nil {
					pageData.Subclasses = append(pageData.Subclasses, entity)
				}
			}
		}
		page.Data = pageData

		err = GeneratePage(data, file, page)
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
