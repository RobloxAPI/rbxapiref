package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/anaminus/but"
	"github.com/jessevdk/go-flags"
	"github.com/robloxapi/rbxapiref/entities"
	"github.com/robloxapi/rbxapiref/manifest"
	"github.com/robloxapi/rbxapiref/settings"
)

type FileSet struct {
	root string
	set  map[string]struct{}
}

func NewFileSet(root string, file ...string) *FileSet {
	files := &FileSet{root: root, set: map[string]struct{}{}}
	for _, file := range file {
		files.Add(file)
	}
	return files
}

func (files *FileSet) Add(file string) {
	files.set[file] = struct{}{}
}

func (files *FileSet) Has(file string) bool {
	_, ok := files.set[file]
	return ok
}

func (files *FileSet) Files() []string {
	fs := make([]string, 0, len(files.set))
	for file := range files.set {
		fs = append(fs, file)
	}
	sort.Strings(fs)
	return fs
}

func ComparePages(outputSettings settings.Output, pages []Page) error {
	// Accumulate generated files.
	files := NewFileSet("")
	files.Add(outputSettings.FilePath("manifest"))
	files.Add(outputSettings.FilePath("search"))
	for _, page := range pages {
		if page.File != "" {
			files.Add(page.File)
		}
		for _, res := range page.Styles {
			files.Add(outputSettings.FilePath("resource", res.Name))
		}
		for _, res := range page.Scripts {
			files.Add(outputSettings.FilePath("resource", res.Name))
		}
		for _, res := range page.Resources {
			files.Add(outputSettings.FilePath("resource", res.Name))
		}
		for _, res := range page.DocResources {
			files.Add(outputSettings.FilePath("docres", res.Name))
		}
	}
	// Include directories.
	for _, file := range files.Files() {
		for {
			dir := filepath.Dir(file)
			if dir == file || dir == string(filepath.Separator) || dir == "." {
				break
			}
			file = dir
			files.Add(dir)
		}
	}

	// Walk the output tree.
	dirs := []string{}
	root := filepath.Dir(outputSettings.AbsFilePath(""))
	err := filepath.Walk(outputSettings.AbsFilePath(""), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if path, err = filepath.Rel(root, path); err != nil {
			return nil
		}
		// Skip "hidden" files that start with a dot.
		if info.Name()[0] == '.' {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip generated files.
		if files.Has(path) {
			return nil
		}
		if info.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		return os.Remove(filepath.Join(root, path))
	})
	if err != nil {
		return err
	}
	for _, path := range dirs {
		if err := os.Remove(filepath.Join(root, path)); err != nil {
			return err
		}
	}
	return nil
}

func RenderPageDirs(root string, pages []Page) error {
	dirs := map[string]struct{}{}
	for _, page := range pages {
		dir := filepath.Join(root, filepath.Dir(page.File))
		if _, ok := dirs[dir]; ok {
			continue
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("make directory: %w", err)
		}
		dirs[dir] = struct{}{}
	}
	return nil
}

func copyResources(outputSettings settings.Output, srcPath, dstType string, resources map[string]*Resource) error {
	dirs := map[string]struct{}{}
	for name, resource := range resources {
		var src io.ReadCloser
		if resource.Content != nil {
			src = ioutil.NopCloser(bytes.NewReader(resource.Content))
		} else {
			var err error
			if src, err = os.Open(filepath.Join(srcPath, name)); err != nil {
				return fmt.Errorf("open resource: %w", err)
			}
		}
		dstname := outputSettings.AbsFilePath(dstType, name)
		dir := filepath.Dir(dstname)
		if _, ok := dirs[dir]; !ok {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("make directory: %w", err)
			}
			dirs[dir] = struct{}{}
		}
		dst, err := os.Create(dstname)
		if err != nil {
			src.Close()
			return fmt.Errorf("create resource: %w", err)
		}
		_, err = io.Copy(dst, src)
		dst.Close()
		src.Close()
		if err != nil {
			return fmt.Errorf("write resource: %w", err)
		}
	}
	return nil
}

type Resources map[string]*Resource

func (r Resources) Add(resource *Resource) {
	// Avoid empty, embedded, or ignored resources.
	if resource.Name == "" || resource.Embed || resource.Ignore {
		return
	}
	if res, ok := r[resource.Name]; ok {
		// Prioritize resources with internal content.
		if res.Content != nil {
			return
		}
	}
	r[resource.Name] = resource
}

func RenderResources(settings settings.Settings, pages []Page) error {
	resources := Resources{}
	docres := Resources{}
	for _, page := range pages {
		for i := range page.Styles {
			resources.Add(&page.Styles[i])
		}
		for i := range page.Scripts {
			resources.Add(&page.Scripts[i])
		}
		for i := range page.Resources {
			resources.Add(&page.Resources[i])
		}
		for i := range page.DocResources {
			docres.Add(&page.DocResources[i])
		}
	}
	if err := copyResources(settings.Output, settings.Input.Resources, "resource", resources); err != nil {
		return err
	}
	return copyResources(settings.Output, settings.Input.DocResources, "docres", docres)
}

type Range struct {
	Min, Max int
	Count    int
}

func (r *Range) UnmarshalFlag(flag string) (err error) {
	sep := strings.Index(flag, ":")
	if sep < 0 {
		r.Min, err = strconv.Atoi(flag)
		r.Count = 1
		return err
	}

	if r.Min, err = strconv.Atoi(flag[:sep]); err != nil {
		return err
	}
	if r.Max, err = strconv.Atoi(flag[sep+1:]); err != nil {
		return err
	}
	r.Count = 2
	return nil
}

func (r Range) Norm(length int) Range {
	if r.Count == 1 {
		r.Max = length
	}
	if r.Min < 0 {
		r.Min = length + r.Min
	} else if r.Min > length {
		r.Min = length
	}
	if r.Max < 0 {
		r.Max = length + r.Max
	} else if r.Max > length {
		r.Max = length
	}
	if r.Min > r.Max {
		r.Min, r.Max = 0, 0
	}
	r.Count = 2
	return r
}

type FlagOptions struct {
	Settings string `short:"s" long:"settings"`
	Force    bool   `long:"force"`
	ResOnly  bool   `long:"res-only"`
	Range    Range  `long:"range"`
	UseGit   bool   `long:"use-git"`
	NoGit    bool   `long:"no-git"`
	Rewind   bool   `long:"rewind"`
	NoRewind bool   `long:"no-rewind"`
}

var options = map[string]*flags.Option{
	"settings": &flags.Option{
		Description: "Specify a custom settings location.",
		ValueName:   "PATH",
	},
	"force": &flags.Option{
		Description: "Force a complete rebuild.",
	},
	"res-only": &flags.Option{
		Description: "Only regenerate resource files.",
	},
	"range": &flags.Option{
		Description: "Select a range of builds.",
		ValueName:   "RANGE",
	},
	"use-git": &flags.Option{
		Description: "Force git-aware document parsing.",
	},
	"no-git": &flags.Option{
		Description: "Force git-unaware document parsing.",
	},
	"rewind": &flags.Option{
		Description: "Force rewinding.",
	},
	"no-rewind": &flags.Option{
		Description: "Force no rewinding.",
	},
}

func ParseOptions(data interface{}, opts flags.Options) *flags.Parser {
	fp := flags.NewParser(data, opts)
	for name, info := range options {
		opt := fp.FindOptionByLongName(name)
		if opt == nil {
			continue
		}
		opt.Description = info.Description
		opt.ValueName = info.ValueName
	}
	return fp
}

func main() {
	var err error

	// Parse flags.
	var opt FlagOptions
	var filters []string
	{
		fp := ParseOptions(&opt, flags.Default|flags.PassAfterNonOption)
		var err error
		filters, err = fp.Parse()
		if err, ok := err.(*flags.Error); ok && err.Type == flags.ErrHelp {
			return
		}
		but.IfFatal(err, "flag parser error")
	}

	// Initialize root.
	data := &Data{
		Time:     time.Now(),
		Manifest: &manifest.Manifest{},
		ResOnly:  opt.ResOnly,
	}

	// Load settings.
	data.Settings = *settings.Default.Copy()
	but.IfFatal(data.Settings.ReadFile(opt.Settings))
	if opt.NoGit {
		data.Settings.Input.UseGit = false
	} else if opt.UseGit {
		data.Settings.Input.UseGit = true
	}
	if opt.NoRewind {
		data.Settings.Build.DisableRewind = true
	} else if opt.Rewind {
		data.Settings.Build.DisableRewind = false
	}

	// Load manifest.
	manifestPath := data.Settings.Output.AbsFilePath("manifest")
	if !opt.Force {
		if b, err := ioutil.ReadFile(manifestPath); err == nil {
			data.Manifest, err = manifest.Decode(bytes.NewReader(b))
			but.IfFatal(err, "read manifest")
		}
	}

	if !opt.ResOnly {
		// Fetch builds.
		builds, err := data.Settings.Build.Fetch()
		but.IfFatal(err)

		if opt.Range.Count > 0 {
			r := opt.Range.Norm(len(builds))
			builds = builds[r.Min:r.Max]
		}

		// Merge uncached builds.
		data.Manifest.Patches, err = data.Settings.Build.Merge(data.Manifest.Patches, builds)
		but.IfFatal(err)
	}

	// Generate entities.
	data.Entities = entities.GenerateEntities(data.Manifest.Patches)
	but.IfFatal(data.GenerateMetadata())
	data.GenerateDocuments()

	if !opt.ResOnly {
		// Compile templates.
		data.Templates, err = CompileTemplates(data.Settings.Input.Templates, TemplateFuncs(data))
		but.IfFatal(err, "open template")
	}

	// Generate pages.
	pages := GeneratePages(data)
	but.IfFatal(ComparePages(data.Settings.Output, pages), "remove unreferenced file")
	if len(filters) > 0 {
		pages, err = FilterPages(pages, filters)
		but.IfFatal(err)
		for _, page := range pages {
			but.Log("INCLUDE PAGE", page.File)
		}
	}
	but.IfFatal(RenderPageDirs(data.Settings.Output.Root, pages))
	but.IfFatal(RenderResources(data.Settings, pages))
	if opt.ResOnly {
		return
	}
	but.IfFatal(data.RenderPages(pages))

	// Generate search database.
	{
		f, err := os.Create(data.Settings.Output.AbsFilePath("search"))
		but.IfFatal(err, "create search database file")
		w := bufio.NewWriter(f)
		but.IfFatal(GenerateDatabase(w, data.Entities), "generate search database")
		but.IfFatal(w.Flush(), "write search database")
		f.Close()
	}

	// Save manifest.
	{
		var buf bytes.Buffer
		err := manifest.Encode(&buf, data.Manifest)
		but.IfFatal(err, "encode manifest")

		f, err := os.Create(manifestPath)
		but.IfFatal(err, "create manifest")
		defer f.Close()

		_, err = buf.WriteTo(f)
		but.IfFatal(err, "write manifest")

		err = f.Sync()
		but.IfFatal(err, "sync manifest")
	}
}
