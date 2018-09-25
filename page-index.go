package main

import (
	"os"
)

func GenerateIndexPage(data *Data) error {
	f, err := os.Create(data.FilePath("index"))
	if err != nil {
		return err
	}
	err = GeneratePage(data, f, Page{
		Template: "index",
		Data:     data,
		Styles:   []Resource{{Name: "index.css", Embed: true}},
		Scripts:  []Resource{{Name: "sort-classes.js"}},
	})
	f.Close()
	return err
}
