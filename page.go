package main

import (
	"io"
)

type Page struct {
	CurrentYear int
	Template    string
	Data        interface{}
	Title       string
	Styles      []Resource
	Scripts     []Resource
}

type Resource struct {
	Name  string // Name of the resource file.
	Embed bool   // Embed the content of the resource.
	ID    string // Optional ID attribute.
}

func GeneratePage(data *Data, w io.Writer, page Page) error {
	return data.Templates.ExecuteTemplate(w, "main", page)
}
