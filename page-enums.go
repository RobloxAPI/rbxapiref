package main

import (
	"os"
	"path/filepath"
)

func GenerateEnumPages(data *Data) error {
	type EnumPageData struct {
		Name   string
		Entity *EnumEntity
	}

	page := Page{
		CurrentYear: data.CurrentYear,
		Template:    "enum",
		Styles:      []Resource{{Name: "enum.css"}},
		Scripts:     []Resource{},
	}
	if err := os.MkdirAll(filepath.Dir(data.FilePath("enum", "enum")), 0666); err != nil {
		return err
	}
	for _, enum := range data.Entities.EnumList {
		path := data.FilePath("enum", enum.ID)
		file, err := os.Create(path)
		if err != nil {
			return err
		}

		page.Title = enum.ID
		page.Data = &EnumPageData{Name: enum.ID, Entity: enum}

		err = GeneratePage(data, file, page)
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
