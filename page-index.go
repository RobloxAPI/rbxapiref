package main

import (
	"os"
)

func GenerateIndexPage(data *Data) error {
	f, err := os.Create(data.FilePath("index"))
	if err != nil {
		return err
	}
	err = data.Templates.ExecuteTemplate(f, "index", data)
	f.Close()
	return err
}
