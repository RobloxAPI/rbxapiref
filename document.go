package rbxapidoc

import (
	"github.com/gomarkdown/markdown"
	markdownhtml "github.com/gomarkdown/markdown/html"
	"html/template"
)

// Section represents a queryable portion of a resource.
type Section interface {
	// Name is the name of the section.
	Name() string
	// Query retrieves the subsection referred to by the given name list. Each
	// successive name refers to a subsection of the previous. Returns nil if
	// the subsection was not found.
	Query(name ...string) Section
	// Subsections returns a list of the subsections within the current
	// section.
	Subsections() []Section
	// Render returns the content rendered to HTML.
	Render() template.HTML
}
