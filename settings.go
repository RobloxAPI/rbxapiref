package main

import (
	"encoding/json"
	"github.com/kirsle/configdir"
	"github.com/pkg/errors"
	"github.com/robloxapi/rbxapiref/fetch"
	"io"
	"os"
	"path/filepath"
)

type Settings struct {
	// Input specifies input settings.
	Input SettingsInput
	// Output specifies output settings.
	Output SettingsOutput
	// Configs maps an identifying name to a fetch configuration.
	Configs map[string]fetch.Config
	// UseConfigs specifies the logical concatenation of the fetch configs
	// defined in the Configs setting. Builds from these configs are read
	// sequentially.
	UseConfigs []string
}

type SettingsInput struct {
	// Resources is the directory containing resource files.
	Resources string
	// Templates is the directory containing template files.
	Templates string
	// Documents is the directory containing document files.
	Documents string
	// DocResources is the directory containing document resource files.
	DocResources string
	// UseGit sets whether document parsing is aware of git. If so, only
	// committed content will be used. That is, untracked files are ignored, and
	// only committed modifications to a file are used.
	UseGit bool
}

type SettingsOutput struct {
	// Root is the directory to which generated files will be written.
	Root string
	// Sub is a path that follows the output directory and precedes a
	// generated file path.
	Sub string
	// Resources is the path relative to Sub where generated resource files
	// will be written.
	Resources string
	// DocResources is the path relative to Sub where document resource files
	// will be written.
	DocResources string
	// Manifest is the path relative to Sub that points to the manifest file.
	Manifest string

	// Host is the host part of the absolute URL of the site.
	Host string
}

func (settings *Settings) ReadFrom(r io.Reader) (n int64, err error) {
	dw := NewDecodeWrapper(r)
	var jsettings struct {
		Input struct {
			Resources    *string
			Templates    *string
			Documents    *string
			DocResources *string
			UseGit       *bool
		}
		Output struct {
			Root         *string
			Sub          *string
			Resources    *string
			DocResources *string
			Manifest     *string
			Host         *string
		}
		Configs    map[string]fetch.Config
		UseConfigs []string
	}
	err = json.NewDecoder(dw).Decode(&jsettings)
	if err != nil {
		return dw.BytesRead(), errors.Wrap(err, "decode settings file")
	}

	wd, _ := os.Getwd()

	merge := func(dst, src *string, path bool) {
		if src != nil && *src != "" {
			*dst = *src
		}
		if path {
			*dst = filepath.Join(wd, *dst)
		}
	}
	merge(&settings.Input.Resources, jsettings.Input.Resources, true)
	merge(&settings.Input.Templates, jsettings.Input.Templates, true)
	merge(&settings.Input.Documents, jsettings.Input.Documents, true)
	merge(&settings.Input.DocResources, jsettings.Input.DocResources, true)
	if jsettings.Input.UseGit != nil && *jsettings.Input.UseGit {
		settings.Input.UseGit = *jsettings.Input.UseGit
	}
	merge(&settings.Output.Root, jsettings.Output.Root, true)
	merge(&settings.Output.Sub, jsettings.Output.Sub, false)
	merge(&settings.Output.Manifest, jsettings.Output.Manifest, true)
	merge(&settings.Output.Resources, jsettings.Output.Resources, true)
	merge(&settings.Output.DocResources, jsettings.Output.DocResources, true)
	merge(&settings.Output.Host, jsettings.Output.Host, false)
	for k, v := range jsettings.Configs {
		settings.Configs[k] = v
	}
	if len(jsettings.UseConfigs) > 0 {
		settings.UseConfigs = append(settings.UseConfigs[:0], jsettings.UseConfigs...)
	}

	return dw.Result()
}

func (settings *Settings) WriteTo(w io.Writer) (n int64, err error) {
	ew := NewEncodeWrapper(w)
	je := json.NewEncoder(ew)
	je.SetEscapeHTML(true)
	je.SetIndent("", "\t")
	err = je.Encode(settings)
	if err != nil {
		return ew.BytesRead(), errors.Wrap(err, "encode settings file")
	}
	return ew.Result()
}

func (settings *Settings) filename(name string) string {
	// User-defined.
	if name != "" {
		return name
	}

	// Portable, if present.
	name = SettingsFile
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		return name
	}

	// Local config.
	name = filepath.Join(configdir.LocalConfig(ToolName), SettingsFile)
	return name
}

func (settings *Settings) ReadFile(filename string) error {
	filename = settings.filename(filename)
	file, err := os.Open(filename)
	if err != nil {
		return errors.Wrap(err, "open settings file")
	}
	defer file.Close()
	_, err = settings.ReadFrom(file)
	return err
}

func (settings *Settings) WriteFile(filename string) error {
	filename = settings.filename(filename)
	file, err := os.Create(filename)
	if err != nil {
		return errors.Wrap(err, "create settings file")
	}
	defer file.Close()
	_, err = settings.WriteTo(file)
	return err
}

func (settings *Settings) Copy() *Settings {
	c := *settings
	c.Configs = make(map[string]fetch.Config, len(settings.Configs))
	for k, v := range settings.Configs {
		c.Configs[k] = v
	}
	c.UseConfigs = make([]string, len(settings.UseConfigs))
	copy(c.UseConfigs, settings.UseConfigs)
	return &c
}
