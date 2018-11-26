package rbxapidoc

import (
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

const DocName = "!doc"

type ID struct {
	Type      string
	Primary   string
	Secondary string
}

type Document struct {
	Orphan   ast.Node
	Summary  ast.Node
	Details  ast.Node
	Examples ast.Node
}

type Section struct {
	Node  *ast.Node
	Level int
	Name  string
}

func Basename(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return base[0 : len(base)-len(ext)]
}

type Config struct {
	Root     string
	FileType string
}

func (cfg Config) queryFile(id ID) (file, sid string, err error) {
	if cfg.FileType != "" && cfg.FileType[0:1] != "." {
		return file, sid, errors.Errorf("invalid FileType \"%s\"", cfg.FileType)
	}
	typeDir := filepath.Join(cfg.Root, id.Type)
	if fi, err := os.Stat(typeDir); os.IsNotExist(err) {
		return file, sid, errors.Errorf("unknown type directory \"%s\"", id.Type)
	} else if !fi.IsDir() {
		return file, sid, errors.Errorf("type \"%s\" is not a directory", id.Type)
	}

	primaryFile, status, err := cfg.findFile(typeDir, id.Primary)
	if err != nil {
		return file, sid, err
	}
	switch {
	case status&1 != 0: // No file.
		return file, sid, errors.Errorf("unknown %s \"%s\"", id.Type, id.Primary)
	case status&2 != 0: // Wrong extension.
		return file, sid, errors.Errorf("file of type \"%s\" for %s \"%s\" not found", cfg.FileType, id.Type, id.Primary)
	}

	if id.Secondary == "" {
		if status&4 != 0 { // IsDir
			primaryFile = filepath.Join(primaryFile, DocName+cfg.FileType)
		}
		return primaryFile, "", nil
	}
	if status&4 == 0 { // !IsDir
		return primaryFile, id.Secondary, nil
	}

	secondaryFile, status, err := cfg.findFile(primaryFile, id.Secondary)
	if err != nil {
		return file, sid, err
	}
	switch {
	case status&1 != 0:
		return file, sid, errors.Errorf("unknown %s member \"%s.%s\"", id.Type, id.Primary, id.Secondary)
	case status&2 != 0:
		return file, sid, errors.Errorf("file of type \"%s\" for %s member \"%s.%s\" not found", cfg.FileType, id.Type, id.Primary, id.Secondary)
	case status&4 != 0:
		return file, sid, errors.Errorf("file of %s member \"%s.%s\" is a directory", id.Type, id.Primary, id.Secondary)
	}

	return secondaryFile, "", nil
}

func (cfg Config) findFile(dir string, id string) (filename string, status int, err error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", status, err
	}

	var file os.FileInfo
	for _, f := range files {
		// Base name must match.
		if Basename(f.Name()) != id {
			continue
		}
		file = f
		// Matching a directory is immediately correct.
		if f.IsDir() {
			status |= 4
			break
		}
		// File with matching extension is immediately correct.
		if filepath.Ext(f.Name()) == cfg.FileType {
			break
		}
	}

	if file == nil {
		status |= 1
		return "", status, nil
	}
	if filepath.Ext(file.Name()) != cfg.FileType {
		status |= 2
		return "", status, nil
	}
	filename = filepath.Join(dir, file.Name())
	return filename, status, nil
}

func (cfg *Config) readFile(file string, id string) (doc *Document, err error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	root := markdown.Parse(b, nil)
	if id != "" {
		cfg.extractSections(root, Section{&root, 1, "Members"})
		cfg.extractSections(root, Section{&root, 2, id})
		// Normalize headers to level 1.
		adjustHeaderLevel(root, -2)
	}
	doc = &Document{}
	cfg.extractSections(root,
		Section{&doc.Orphan, 1, ""},
		Section{&doc.Summary, 1, "Summary"},
		Section{&doc.Details, 1, "Details"},
		Section{&doc.Examples, 1, "Examples"},
	)
	if id == "" {
		// Adjust to rendered page.
		adjustHeaderLevel(root, 1)
	} else {
		// Undo normalization; adjust to rendered page.
		adjustHeaderLevel(root, 3)
	}
	return doc, nil
}

func (cfg *Config) extractSections(root ast.Node, sections ...Section) {
	if root == nil {
		return
	}
	children := root.GetChildren()
	for _, section := range sections {
		i, j := -1, -1
		if section.Name == "" {
			// Orphaned section.
			i, j = -1, len(children)
			for k, child := range children {
				heading, ok := child.(*ast.Heading)
				if ok && heading.Level <= section.Level {
					j = k
					break
				}
			}
		} else {
			for k, child := range children {
				heading, ok := child.(*ast.Heading)
				if !ok || heading.Level > section.Level {
					continue
				}
				if i < 0 {
					if getHeadingText(heading) == section.Name {
						i = k
						j = len(children)
					}
				} else {
					j = k
					break
				}
			}
		}
		if i < 0 && j < 0 {
			continue
		}
		doc := ast.Document{}
		if i < j {
			doc.Children = children[i+1 : j]
		}
		*section.Node = &doc
	}
}

// QueryFile retrieves the file location of a markdown document referred to by
// the given ID. Returns an empty string if no document is present.
func (cfg Config) QueryFile(id ID) (file string, err error) {
	file, _, err = cfg.queryFile(id)
	return file, err
}

// Query retrieves a markdown document referred to by the given ID. Returns
// nil if no document is present.
func (cfg Config) Query(id ID) (doc *Document, err error) {
	file, sid, err := cfg.queryFile(id)
	if err != nil {
		return nil, err
	}
	return cfg.readFile(file, sid)
}

func adjustHeaderLevel(root ast.Node, delta int) {
	if root == nil {
		return
	}
	if heading, ok := root.(*ast.Heading); ok {
		heading.Level += delta
	}
	for _, child := range root.GetChildren() {
		adjustHeaderLevel(child, delta)
	}
}

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
