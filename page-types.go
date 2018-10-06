package main

import (
	"os"
	"path/filepath"
)

func GenerateTypePages(data *Data) error {
	type TypePageData struct {
		Name   string
		Entity *TypeEntity
	}

	page := Page{
		CurrentYear: data.CurrentYear,
		Template:    "type",
		Styles:      []Resource{},
		Scripts:     []Resource{},
	}
	if err := os.MkdirAll(filepath.Dir(data.FilePath("type", "type", "type")), 0666); err != nil {
		return err
	}
	for _, typ := range data.Entities.TypeList {
		path := data.FilePath("type", typ.Element.Category, typ.Element.Name)
		file, err := os.Create(path)
		if err != nil {
			return err
		}

		page.Title = typ.ID
		page.Data = &TypePageData{Name: typ.ID, Entity: typ}

		err = GeneratePage(data, file, page)
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
