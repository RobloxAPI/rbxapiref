package main

import (
	"os"
)

func GenerateAboutPage(data *Data) error {
	f, err := os.Create(data.FilePath("about"))
	if err != nil {
		return err
	}
	err = GeneratePage(data, f, Page{
		CurrentYear: data.CurrentYear,
		Template:    "about",
		Data:        data,
		Styles:      []Resource{{Name: "about.css", Embed: true}},
	})
	f.Close()
	return err
}
