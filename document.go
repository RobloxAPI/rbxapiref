package main

import (
	"github.com/gomarkdown/markdown"
	"github.com/robloxapi/rbxapiref/rbxapidoc"
	"html/template"
)

type Document interface {
	Query(name ...string) rbxapidoc.Section
	SetRender(renderer markdown.Renderer)
	Render() template.HTML
}
