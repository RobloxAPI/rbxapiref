package rbxapidoc

import (
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

// Headingable extends a Section by representing an outline with traversable
// headings.
type Headingable interface {
	Section
	// AdjustLevels offsets the levels of all headings in the outline, such
	// that RootLevel returns the given value.
	AdjustLevels(level int)
	// RootLevel returns the level of the root heading. This is defined as one
	// level less than the lowest heading level present in the outline.
	RootLevel() int
}

// Linkable extends a Section by representing a document with traversable
// reference links.
type Linkable interface {
	Section
	// Links receives a walk function, which receives a link. The function is
	// applied to all links within the section, which can include those within
	// subsections.
	Links(walk func(link string))
	// SetLinks receives a walk function, which receives a link and returns an
	// adjusted link. The function is applied to all links within the section,
	// which can include those within subsections.
	SetLinks(walk func(link string) string)
}
