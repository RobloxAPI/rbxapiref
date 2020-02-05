package main

import (
	"github.com/gomarkdown/markdown"
	"github.com/robloxapi/rbxapiref/documents"
	"html/template"
)

type Document interface {
	Query(name ...string) documents.Section
	SetRender(renderer markdown.Renderer)
	Render() template.HTML
}
