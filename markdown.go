package rbxapidoc

import (
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
)

// MarkdownSection represents a portion of a markdown document. Sections are
// delimited by headings.
type MarkdownSection struct {
	// Heading is the name of the outer heading enclosing the section.
	Heading string
	// Level is the level of the outer heading enclosing the section.
	Level int
	// Document is the raw content of the section.
	Document *ast.Document
	// Sections contains each subsection.
	Sections []*MarkdownSection
	// Renderer specifies a custom renderer to use when rendering the section
	// content to HTML. If nil, the default HTML renderer is used.
	Renderer markdown.Renderer
}

// MarkdownFileHandler is a FileHandler that parses a markdown file.
func MarkdownFileHandler(dir string, info os.FileInfo, query string) Section {
	if info.IsDir() {
		return nil
	}
	ext := filepath.Ext(info.Name())
	if ext != ".md" {
		return nil
	}
	base := filepath.Base(info.Name())
	if base[:len(base)-len(ext)] != query {
		return nil
	}

	b, err := ioutil.ReadFile(filepath.Join(dir, info.Name()))
	if err != nil {
		return nil
	}
	doc, ok := markdown.Parse(b, nil).(*ast.Document)
	if !ok {
		return nil
	}
	return NewMarkdownSection(doc)
}

// getHeadingText returns the text from an ast.Heading.
func getHeadingText(heading *ast.Heading) string {
	if len(heading.Children) != 1 {
		return ""
	}
	text, ok := heading.Children[0].(*ast.Text)
	if !ok {
		return ""
	}
	return string(text.Literal)
}

func parseMarkdownSection(section *MarkdownSection, level int) {
	children := section.Document.Children

	var i int
	var name string
	var hasSub bool
	for k, child := range children {
		heading, ok := child.(*ast.Heading)
		if !ok || heading.Level > level {
			if ok {
				// Section has a subsection.
				hasSub = true
			}
			continue
		}
		sub := MarkdownSection{
			Heading:  name,
			Level:    level,
			Document: &ast.Document{},
		}
		if i < k {
			sub.Document.Children = children[i:k]
		}
		parseMarkdownSection(&sub, level+1)
		section.Sections = append(section.Sections, &sub)
		hasSub = false
		i = k + 1
		name = getHeadingText(heading)
	}
	sub := MarkdownSection{
		Heading:  name,
		Level:    level,
		Document: &ast.Document{},
		Renderer: section.Renderer,
	}
	// `i>=len` means that the previously added section is the last, and is
	// empty.
	if i < len(children) {
		sub.Document.Children = children[i:]
		if hasSub {
			parseMarkdownSection(&sub, level+1)
		}
	}
	// `i==0` means that the section is an orphan that covers the same
	// children as its parent, and therefore has no siblings. Sections without
	// siblings or children are avoided to prevent infinite recursion.
	if i > 0 || hasSub {
		section.Sections = append(section.Sections, &sub)
	}
}

// NewMarkdownSection creates a new MarkdownSection from an ast.Document.
//
// Subsections are created by outlining the headings of the document; each
// subheading corresponds to a subsection, which can be queried by the name of
// the heading. Also included are "orphaned" sections, which enclose parts of
// the document without a heading. These can be queried with an empty string.
//
// Only headings which are direct children of the document are outlined. Note
// that all subsections share the same underlying document. i.e. if a node
// within a section is modified, the parent section will be affected.
func NewMarkdownSection(document *ast.Document) *MarkdownSection {
	section := &MarkdownSection{Document: document}
	parseMarkdownSection(section, 1)
	return section
}

func (s *MarkdownSection) Name() string {
	return s.Heading
}

func (s *MarkdownSection) Query(name ...string) Section {
	if len(name) == 0 {
		return nil
	}
	for _, sub := range s.Sections {
		if sub.Heading != name[0] {
			continue
		}
		if len(name) > 1 {
			return sub.Query(name[1:]...)
		}
		return sub
	}
	return nil
}

func (s *MarkdownSection) Subsections() []Section {
	subs := make([]Section, len(s.Sections))
	for i, sub := range s.Sections {
		subs[i] = sub
	}
	return subs
}

func (s *MarkdownSection) SetRender(renderer markdown.Renderer) {
	s.Renderer = renderer
}

func (s *MarkdownSection) Render() template.HTML {
	renderer := s.Renderer
	if renderer == nil {
		renderer = html.NewRenderer(html.RendererOptions{})
	}
	render := markdown.Render(s.Document, renderer)
	for _, b := range render {
		switch b {
		case '\t', '\n', '\v', '\f', '\r', ' ', 0x85, 0xA0:
			continue
		}
		return template.HTML(render)
	}
	// Return empty string if all characters are spaces.
	return ""
}

// AdjustLevel adjusts the level of each heading node in the document such
// that RootLevel returns the given value. This does not affect the Level
// field of the section and subsections.
func (s *MarkdownSection) AdjustLevel(level int) {
	root := s.RootLevel()
	if root < 0 {
		return
	}
	delta := level - root
	for _, child := range s.Document.GetChildren() {
		if heading, ok := child.(*ast.Heading); ok {
			heading.Level += delta
		}
	}
}

// RootLevel returns the level of the root heading, which is defined as one
// less than the lowest heading level present in the document. Returns -1 if
// there are no headings in the document. Heading levels are assumed to be
// positive.
func (s *MarkdownSection) RootLevel() (level int) {
	for _, child := range s.Document.GetChildren() {
		if heading, ok := child.(*ast.Heading); ok && (level == 0 || heading.Level < level) {
			level = heading.Level
		}
	}
	return level - 1
}
