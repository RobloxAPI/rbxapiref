package main

import (
	"github.com/gomarkdown/markdown"
	"github.com/robloxapi/rbxapidoc"
	"html/template"
)

type Document interface {
	Query(name ...string) rbxapidoc.Section
	SetRender(renderer markdown.Renderer)
	Render() template.HTML
}

type dummyDocument struct{}

func (dummyDocument) Query(...string) rbxapidoc.Section { return nil }
func (dummyDocument) SetRender(markdown.Renderer)       {}
func (dummyDocument) Render() template.HTML             { return "" }
