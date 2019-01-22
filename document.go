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

type ID struct {
	Type      string
	Primary   string
	Secondary string
}

type Locator struct {
	root     *DirectorySection
	renderer markdown.Renderer
	content  string
}

// NewLocator creates a new Locator for a given directory path.
func NewLocator(path string) *Locator {
	locator := Locator{
		root: NewDirectorySection(path,
			MarkdownFileHandler,
		),
		renderer: markdownhtml.NewRenderer(markdownhtml.RendererOptions{}),
		content:  "!content",
	}
	return &locator
}

// Render locates a document from the given ID. It then retrieves the
// subsection referred to by the given name list. Each successive name refers
// to a subsection of the previous. Then, the root heading is adjusted to the
// given level, and all subheadings are adjusted by the same amount. Finally,
// the document is rendered to HTML, and the result is returned.
//
// If the document of the given ID or queried section could not be found, then
// an empty string is returned.
//
// Note that the root heading is never actually rendered, but is instead used
// to determine the relative difference in levels for subheadings. For
// example, passing level 0 causes the first heading level in the document to
// be 1, the next level to be 2, and so on. A level less than 0 causes no
// adjustment to be made to heading levels.
func (l *Locator) Render(id ID, level int, section ...string) template.HTML {
	typeDir, ok := l.root.Query(id.Type).(*DirectorySection)
	if !ok || typeDir == nil {
		return ""
	}

	// TODO: Make extension-agnostic.
	var result *MarkdownSection
	if id.Secondary == "" {
		for _, primary := range typeDir.QueryAll(id.Primary) {
			switch primary := primary.(type) {
			case *DirectorySection:
				// Try $PRIMARY/$CONTENT.$EXT
				if content, ok := primary.Query(l.content).(*MarkdownSection); ok && content != nil {
					result = content
					goto result
				}
			case *MarkdownSection:
				// Try $PRIMARY.$EXT
				result = primary
				goto result
			}
		}
	} else {
		for _, primary := range typeDir.QueryAll(id.Primary) {
			switch primary := primary.(type) {
			case *DirectorySection:
				// Try $PRIMARY/$SECONDARY.$EXT
				if secondary, ok := primary.Query(id.Secondary).(*MarkdownSection); ok && secondary != nil {
					result = secondary
					goto result
				}
				// Try $PRIMARY/$CONTENT.$EXT
				if content, ok := primary.Query(l.content).(*MarkdownSection); ok && content != nil {
					// Try $PRIMARY/$CONTENT.$EXT#$SECONDARY
					if secondary, ok := content.Query("Members", id.Secondary).(*MarkdownSection); ok && secondary != nil {
						result = secondary
						goto result
					}
				}
			case *MarkdownSection:
				// Try $PRIMARY.$EXT#$SECONDARY
				if secondary, ok := primary.Query("Members", id.Secondary).(*MarkdownSection); ok && secondary != nil {
					result = secondary
					goto result
				}
			}
		}
	}

result:
	if result == nil {
		return ""
	}

	if len(section) > 0 {
		result = result.Query(section...).(*MarkdownSection)
	}
	if result == nil {
		return ""
	}
	result.AdjustLevel(level - (result.TrueLevel() - 1))

	result.Renderer = l.renderer
	return result.Render()
}

// Free frees resources associated with the given ID.
func (l *Locator) Free(id ID) {
}

// FreeAll frees all resources used by the locator.
func (l *Locator) FreeAll() {
}
