package documents

import (
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
)

// FileHandler hooks into a directory query, transforming a file within the
// directory into a Section.
//
// The dir argument is the directory in which the file is located. The info
// argument is information about the current file. The query argument is the
// current query being made.
//
// The handler should return nil when the query does not match, or the
// information given about the file is inapplicable.
type FileHandler func(dir string, info os.FileInfo, query string) Section

// DirectorySection represents a directory in a file system.
type DirectorySection struct {
	Path     string
	Handlers []FileHandler
}

// NewDirectorySection returns a new DirectorySection from the given path.
// Optional handlers may be specified.
func NewDirectorySection(path string, handlers ...FileHandler) *DirectorySection {
	return &DirectorySection{Path: path, Handlers: handlers}
}

func (s DirectorySection) Name() string {
	return filepath.Base(s.Path)
}

func (s DirectorySection) query(queries ...string) (section Section, next []string, isfile bool) {
	if len(queries) == 0 {
		return nil, nil, false
	}
	query, next := queries[0], queries[1:]
	files, err := ioutil.ReadDir(s.Path)
	if err != nil {
		return nil, nil, false
	}
	for _, info := range files {
		// Try each handler.
		for _, handler := range s.Handlers {
			if section = handler(s.Path, info, query); section != nil {
				return section, next, true
			}
		}
		// Try subdirectory.
		if info.IsDir() && query == info.Name() {
			section = &DirectorySection{
				Path:     filepath.Join(s.Path, info.Name()),
				Handlers: s.Handlers,
			}
			break
		}
	}
	return section, next, false
}

// Query queries a file of the given name from the directory. For each file of
// the directory, each FileHandler is called in order. If a handler returns a
// non-nil Section, then that section becomes the result.
//
// If no handler returns a section, and the current file is a directory whose
// name matches the query, then the result will be a DirectorySection that
// inherits the handlers of the current section.
//
// Subsequent names are queried from the resulting section, if one exists.
func (s DirectorySection) Query(name ...string) Section {
	section, next, _ := s.query(name...)
	if section == nil {
		return nil
	}
	if len(next) > 0 {
		return section.Query(next...)
	}
	return section
}

// QueryAll is similar to Query, but returns all sections matching the first
// name.
func (s DirectorySection) QueryAll(name string) (sections []Section) {
	files, err := ioutil.ReadDir(s.Path)
	if err != nil {
		return nil
	}
	for _, info := range files {
		// Try each handler.
		for _, handler := range s.Handlers {
			if section := handler(s.Path, info, name); section != nil {
				sections = append(sections, section)
			}
		}
		// Try subdirectory.
		if info.IsDir() && name == info.Name() {
			sections = append(sections, &DirectorySection{
				Path:     filepath.Join(s.Path, info.Name()),
				Handlers: s.Handlers,
			})
		}
	}
	return sections
}

func (s DirectorySection) Subsections() []Section {
	files, err := ioutil.ReadDir(s.Path)
	if err != nil {
		return nil
	}
	var sections []Section
	for _, info := range files {
		if !info.IsDir() {
			continue
		}
		sections = append(sections, &DirectorySection{
			Path:     filepath.Join(s.Path, info.Name()),
			Handlers: s.Handlers,
		})
	}
	return sections
}

func (s DirectorySection) Render() template.HTML {
	return template.HTML(s.Path)
}
