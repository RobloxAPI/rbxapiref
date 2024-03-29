package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/alecthomas/chroma"
	chhtml "github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/gomarkdown/markdown/ast"
	mdhtml "github.com/gomarkdown/markdown/html"
	"github.com/robloxapi/rbxapiref/builds"
	"github.com/robloxapi/rbxapiref/documents"
	"github.com/robloxapi/rbxapiref/entities"
	"github.com/robloxapi/rbxapiref/fetch"
	"github.com/robloxapi/rbxapiref/manifest"
	"github.com/robloxapi/rbxapiref/settings"
	"github.com/robloxapi/rbxfile"
)

type Data struct {
	Settings      settings.Settings
	Manifest      *manifest.Manifest
	Time          time.Time
	Entities      *entities.Entities
	Templates     *template.Template
	CodeFormatter *chhtml.Formatter
	ResOnly       bool
}

func (data *Data) GenerateResourceElements(resources []Resource) (v []interface{}, err error) {
	for _, resource := range resources {
		var ResData struct {
			Type     string
			Resource Resource
			Content  interface{}
		}
		ResData.Type = filepath.Ext(resource.Name)
		ResData.Resource = resource
		ResData.Resource.Attr = nil
		if resource.Embed {
			var content []byte
			if resource.Content != nil {
				content = resource.Content
			} else {
				filename := filepath.Join(data.Settings.Input.Resources, resource.Name)
				if content, err = ioutil.ReadFile(filename); err != nil {
					return nil, err
				}
			}
			switch ResData.Type {
			case ".css":
				ResData.Content = template.CSS(content)
			case ".js":
				ResData.Content = template.JS(content)
			case ".html", ".svg":
				ResData.Content = template.HTML(content)
			default:
				ResData.Content = string(content)
			}
		} else {
			switch ResData.Type {
			case ".css":
				ResData.Resource.Attr = append(ResData.Resource.Attr,
					Attr{Name: "href", Value: data.Settings.Output.FileLink("resource", resource.Name)},
					Attr{Name: "rel", Value: "stylesheet"},
					Attr{Name: "type", Value: "text/css"},
				)
			case ".js":
				ResData.Resource.Attr = append(ResData.Resource.Attr,
					Attr{Name: "src", Value: data.Settings.Output.FileLink("resource", resource.Name)},
					Attr{Name: "charset", Value: "utf-8"},
				)
			}
		}
		ResData.Resource.Attr.Merge(resource.Attr)

		var buf bytes.Buffer
		if err := data.Templates.ExecuteTemplate(&buf, "resource", ResData); err != nil {
			return nil, err
		}
		v = append(v, template.HTML(buf.String()))
	}
	return v, nil
}

func (data *Data) GenerateHistoryElements(entity interface{}, button bool, ascending bool) (template.HTML, error) {
	var patches []builds.Patch
	switch entity := entity.(type) {
	case *entities.Class:
		patches = builds.MergePatches(entity.Patches, nil, nil)
		for _, member := range entity.MemberList {
			patches = builds.MergePatches(patches, member.Patches, func(action *builds.Action) bool {
				// Filter actions where the parent entity is the cause.
				return action.GetMember() != nil
			})
		}
	case *entities.Member:
		patches = builds.MergePatches(entity.Patches, nil, nil)
	case *entities.Enum:
		patches = builds.MergePatches(entity.Patches, nil, nil)
		for _, item := range entity.ItemList {
			patches = builds.MergePatches(patches, item.Patches, func(action *builds.Action) bool {
				return action.GetEnumItem() != nil
			})
		}
	case *entities.EnumItem:
		patches = builds.MergePatches(entity.Patches, nil, nil)
	default:
		return "", nil
	}
	if len(patches) == 0 {
		return "", nil
	}
	if len(patches) == 1 && data.Manifest.Patches[0].Info.Equal(patches[0].Info) {
		return "", nil
	}
	if ascending {
		sort.Slice(patches, func(i, j int) bool {
			return patches[i].Info.Date.Before(patches[j].Info.Date)
		})
	} else {
		sort.Slice(patches, func(i, j int) bool {
			return patches[i].Info.Date.After(patches[j].Info.Date)
		})
	}
	var buf bytes.Buffer
	err := data.Templates.ExecuteTemplate(&buf, "history", struct {
		First   builds.Info
		Patches []builds.Patch
		Button  bool
	}{data.Manifest.Patches[0].Info, patches, button})
	if err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

func (data *Data) RenderPages(pages []Page) error {
	var rootData struct {
		Data     *Data
		MainPage *Page
		Page     *Page
	}
	rootData.Data = data
	// Treat first page with unspecified filename as main page.
	for _, page := range pages {
		if page.File == "" {
			rootData.MainPage = &page
			break
		}
	}
	if rootData.MainPage == nil {
		return errors.New("no main template")
	}
	for _, page := range pages {
		if page.File == "" {
			continue
		}
		file, err := os.Create(filepath.Join(data.Settings.Output.Root, page.File))
		if err != nil {
			return fmt.Errorf("create file: %w", err)
		}
		if page.Data == nil {
			page.Data = data
		}
		rootData.Page = &page
		err = data.Templates.ExecuteTemplate(file, rootData.MainPage.Template, rootData)
		file.Close()
		if err != nil {
			return fmt.Errorf("generate page: %w", err)
		}
	}
	return nil
}

func (data *Data) GenerateMetadata() error {
	if data.ResOnly {
		return nil
	}

	// If latest metadata fails, retry with previous metadata.
	var rmd *rbxfile.Root
	const retryCount = 3
	for i := range data.Manifest.Patches {
		latest := data.Manifest.Patches[len(data.Manifest.Patches)-1-i]
		client := &fetch.Client{
			Config:    data.Settings.Build.Configs[latest.Config],
			CacheMode: fetch.CacheTemp,
		}
		var err error
		rmd, err = client.ReflectionMetadata(latest.Info.Hash)
		if err != nil {
			if i <= retryCount {
				continue
			}
			return fmt.Errorf("fetch metadata %s: %w", latest.Info.Hash, err)
		}
		break
	}

	for _, list := range rmd.Instances {
		switch list.ClassName {
		case "ReflectionMetadataClasses":
			for _, class := range list.Children {
				if class.ClassName != "ReflectionMetadataClass" {
					continue
				}
				entity := data.Entities.Classes[class.Name()]
				if entity == nil {
					continue
				}
				entity.Metadata.Instance = class

				for _, memberTypeList := range class.Children {
					for _, member := range memberTypeList.Children {
						if member.ClassName != "ReflectionMetadataMember" {
							continue
						}
						entity := entity.Members[member.Name()]
						if entity == nil {
							continue
						}
						entity.Metadata.Instance = member
					}
				}
			}
		case "ReflectionMetadataEnums":
			for _, enum := range list.Children {
				if enum.ClassName != "ReflectionMetadataEnum" {
					continue
				}
				entity := data.Entities.Enums[enum.Name()]
				if entity == nil {
					continue
				}
				entity.Metadata.Instance = enum

				for _, item := range enum.Children {
					if item.ClassName != "ReflectionMetadataEnumItem" {
						continue
					}
					entity := entity.Items[item.Name()]
					if entity == nil {
						continue
					}
					entity.Metadata.Instance = item
				}
			}
		}
	}
	return nil
}

func generateDocStatus(entity interface{}) (s entities.DocStatus) {
	setStatus := func(status *int, hasDoc bool, section documents.Section) {
		if !hasDoc {
			*status = 0
		} else if section == nil {
			*status = 1
		} else if count, ok := section.(documents.Countable); ok && count.Count() == 0 {
			*status = 2
		} else {
			*status = 3
		}
	}

	var document entities.Document
	if doc, ok := entity.(entities.Documentable); ok {
		document = doc.GetDocument()
	}
	var summary documents.Section
	var details documents.Section
	var examples documents.Section
	if document != nil {
		s.HasDocument = true
		if summary = document.Query("Summary"); summary == nil {
			if summary = document.Query(""); summary != nil {
				s.SummaryOrphaned = true
			}
		}
		details = document.Query("Details")
		examples = document.Query("Examples")
	}
	setStatus(&s.SummaryStatus, s.HasDocument, summary)
	setStatus(&s.DetailsStatus, s.HasDocument, details)
	setStatus(&s.ExamplesStatus, s.HasDocument, examples)
	if count, ok := details.(documents.Countable); ok {
		s.DetailsSections = count.Count()
	}
	if count, ok := examples.(documents.Countable); ok {
		s.ExampleCount = count.Count()
	}

	var count int
	var total int
	switch entity := entity.(type) {
	case *entities.Class:
		total += 3
		if s.SummaryStatus >= 3 {
			count++
		}
		if s.DetailsStatus >= 3 {
			count++
		}
		if s.ExamplesStatus >= 3 {
			count++
		}
		for _, member := range entity.MemberList {
			total += 3
			if member.DocStatus.SummaryStatus >= 3 {
				count++
			}
			if member.DocStatus.DetailsStatus >= 3 {
				count++
			}
			if member.DocStatus.ExamplesStatus >= 3 {
				count++
			}
			if s.HasDocument {
				// Entity has a document, which means all members have a
				// document.
				if member.DocStatus.SummaryStatus == 0 {
					member.DocStatus.SummaryStatus = 1
				}
				if member.DocStatus.DetailsStatus == 0 {
					member.DocStatus.DetailsStatus = 1
				}
				if member.DocStatus.ExamplesStatus == 0 {
					member.DocStatus.ExamplesStatus = 1
				}
				if member.DocStatus.AggregateStatus == 0 {
					member.DocStatus.AggregateStatus = 1
				}
			}
		}
	case *entities.Member:
		total += 3
		if s.SummaryStatus >= 3 {
			count++
		}
		if s.DetailsStatus >= 3 {
			count++
		}
		if s.ExamplesStatus >= 3 {
			count++
		}
	case *entities.Enum:
		// Examples not required for enums.
		total += 2
		if s.SummaryStatus >= 3 {
			count++
		}
		if s.DetailsStatus >= 3 {
			count++
		}
		// Show no status unless section is provided.
		if s.ExamplesStatus < 2 {
			s.ExamplesStatus = 0
		}
		for _, item := range entity.ItemList {
			total += 1
			if item.DocStatus.SummaryStatus >= 3 {
				count++
			}
			if s.HasDocument {
				if item.DocStatus.SummaryStatus == 0 {
					item.DocStatus.SummaryStatus = 1
				}
				if item.DocStatus.AggregateStatus == 0 {
					item.DocStatus.AggregateStatus = 1
				}
			}
		}
	case *entities.EnumItem:
		// Only include summary. In most cases, details and examples for every
		// single enum item is going overboard.
		total += 1
		if s.SummaryStatus >= 3 {
			count++
		}
		if s.DetailsStatus < 2 {
			s.DetailsStatus = 0
		}
		if s.ExamplesStatus < 2 {
			s.ExamplesStatus = 0
		}
	case entities.TypeCategory:
		for _, typ := range entity.Types {
			total += 3
			if typ.DocStatus.SummaryStatus >= 3 {
				count++
			}
			if typ.DocStatus.DetailsStatus >= 3 {
				count++
			}
			if typ.DocStatus.ExamplesStatus >= 3 {
				count++
			}
		}
	case *entities.Type:
		total += 3
		if s.SummaryStatus >= 3 {
			count++
		}
		if s.DetailsStatus >= 3 {
			count++
		}
		if s.ExamplesStatus >= 3 {
			count++
		}
	}

	switch {
	case count == total:
		s.AggregateStatus = 3
	case count > 0:
		s.AggregateStatus = 2
	case count == 0 && s.HasDocument:
		s.AggregateStatus = 1
	}
	if total > 0 {
		s.AggregateProgress = float64(count) / float64(total)
	} else {
		s.AggregateProgress = 1
	}
	return s
}

func (data *Data) GenerateDocuments() {
	if data.Settings.Input.Documents == "" {
		return
	}

	data.CodeFormatter = chhtml.New(
		chhtml.WithClasses(),
		chhtml.TabWidth(4),
		chhtml.WithLineNumbers(),
		chhtml.LineNumbersInTable(),
	)

	if data.ResOnly {
		return
	}

	renderer := func() *mdhtml.Renderer {
		return mdhtml.NewRenderer(mdhtml.RendererOptions{
			HeadingIDPrefix: "doc-",
			RenderNodeHook: func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
				switch node := node.(type) {
				case *ast.CodeBlock:
					// io.WriteString
					i := bytes.IndexAny(node.Info, "\t ")
					if i < 0 {
						i = len(node.Info)
					}
					lang := string(node.Info[:i])
					lexer := lexers.Get(lang)
					if lexer == nil {
						return ast.GoToNext, false
					}
					lexer = chroma.Coalesce(lexer)
					iterator, err := lexer.Tokenise(nil, string(node.Literal))
					if err != nil {
						return ast.GoToNext, false
					}
					var buf bytes.Buffer
					if err := data.CodeFormatter.Format(&buf, settings.StyleRobloxLight, iterator); err != nil {
						return ast.GoToNext, false
					}
					io.Copy(w, &buf)

					return ast.SkipChildren, true
				}
				return ast.GoToNext, false
			},
		})
	}

	docDir := documents.NewDirectorySection(
		data.Settings.Input.Documents,
		documents.MarkdownHandler{
			UseGit:        data.Settings.Input.UseGit,
			StripComments: true,
		}.FileHandler,
	)
	if apiDir := docDir.Query("api"); apiDir != nil {
		for _, entity := range data.Entities.ClassList {
			if entity.Document, _ = apiDir.Query("class", entity.ID).(entities.Document); entity.Document != nil {
				entity.Document.SetRender(renderer())
				GenerateDocumentTypeIDs(entity.Document)
				for _, member := range entity.MemberList {
					if member.Document, _ = entity.Document.Query("Members", member.ID[1]).(entities.Document); member.Document != nil {
						member.Document.SetRender(renderer())
					}
				}
			}
		}
		for _, entity := range data.Entities.EnumList {
			if entity.Document, _ = apiDir.Query("enum", entity.ID).(entities.Document); entity.Document != nil {
				entity.Document.SetRender(renderer())
				for _, item := range entity.ItemList {
					if item.Document, _ = entity.Document.Query("Members", item.ID[1]).(entities.Document); item.Document != nil {
						item.Document.SetRender(renderer())
					}
				}
			}
		}
		for _, entity := range data.Entities.TypeList {
			if entity.Document, _ = apiDir.Query("type", entity.ID).(entities.Document); entity.Document != nil {
				entity.Document.SetRender(renderer())
				GenerateDocumentTypeIDs(entity.Document)
			}
		}
	}

	total := float64(len(data.Entities.ClassList) +
		len(data.Entities.EnumList) +
		len(data.Entities.TypeList))
	var count float64
	for _, entity := range data.Entities.ClassList {
		for _, member := range entity.MemberList {
			member.DocStatus = generateDocStatus(member)
		}
		entity.DocStatus = generateDocStatus(entity)
		count += entity.DocStatus.AggregateProgress
	}
	for _, entity := range data.Entities.EnumList {
		for _, item := range entity.ItemList {
			item.DocStatus = generateDocStatus(item)
		}
		entity.DocStatus = generateDocStatus(entity)
		count += entity.DocStatus.AggregateProgress
	}
	for _, entity := range data.Entities.TypeList {
		entity.DocStatus = generateDocStatus(entity)
		count += entity.DocStatus.AggregateProgress
	}
	data.Entities.Coverage = float32(count / total)
}
