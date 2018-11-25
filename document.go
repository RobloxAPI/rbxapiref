package main

import (
	"github.com/gomarkdown/markdown"
	"github.com/robloxapi/rbxapidoc"
	"html/template"
)

type Document struct {
	Summary  template.HTML
	Details  template.HTML
	Examples template.HTML
}

func RenderDocument(renderer markdown.Renderer, doc *rbxapidoc.Document) (result Document) {
	if doc.Summary == nil {
		if doc.Orphan != nil {
			result.Summary = template.HTML(markdown.Render(doc.Orphan, renderer))
		}
	} else {
		result.Summary = template.HTML(markdown.Render(doc.Summary, renderer))
	}
	if doc.Details != nil {
		result.Details = template.HTML(markdown.Render(doc.Details, renderer))
	}
	if doc.Examples != nil {
		result.Examples = template.HTML(markdown.Render(doc.Examples, renderer))
	}
	return result
}
