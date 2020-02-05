package main

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/robloxapi/rbxapiref/builds"
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
	// Build specifies build settings.
	Build builds.Settings
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
		Build struct {
			Configs       map[string]fetch.Config
			UseConfigs    []string
			DisableRewind *bool
		}
	}
	err = json.NewDecoder(dw).Decode(&jsettings)
	if err != nil {
		return dw.BytesRead(), errors.Wrap(err, "decode settings file")
	}

	wd, _ := os.Getwd()

	mergeString := func(dst, src *string, path bool) {
		if src != nil && *src != "" {
			*dst = *src
		}
		if path && !filepath.IsAbs(*dst) {
			*dst = filepath.Join(wd, *dst)
		}
	}
	mergeBool := func(dst, src *bool) {
		if src != nil && *src {
			*dst = *src
		}
	}
	mergeString(&settings.Input.Resources, jsettings.Input.Resources, true)
	mergeString(&settings.Input.Templates, jsettings.Input.Templates, true)
	mergeString(&settings.Input.Documents, jsettings.Input.Documents, true)
	mergeString(&settings.Input.DocResources, jsettings.Input.DocResources, true)
	mergeBool(&settings.Input.UseGit, jsettings.Input.UseGit)
	mergeBool(&settings.Build.DisableRewind, jsettings.Build.DisableRewind)
	mergeString(&settings.Output.Root, jsettings.Output.Root, true)
	mergeString(&settings.Output.Sub, jsettings.Output.Sub, false)
	mergeString(&settings.Output.Manifest, jsettings.Output.Manifest, false)
	mergeString(&settings.Output.Resources, jsettings.Output.Resources, false)
	mergeString(&settings.Output.DocResources, jsettings.Output.DocResources, false)
	mergeString(&settings.Output.Host, jsettings.Output.Host, false)
	for k, v := range jsettings.Build.Configs {
		settings.Build.Configs[k] = v
	}
	if len(jsettings.Build.UseConfigs) > 0 {
		settings.Build.UseConfigs = append(settings.Build.UseConfigs[:0], jsettings.Build.UseConfigs...)
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

func (settings *Settings) filename(name string) (string, error) {
	// User-defined.
	if name != "" {
		return name, nil
	}

	// Portable, if present.
	name = SettingsFile
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		return name, nil
	}

	// Local config.
	config, err := userConfigDir()
	if err != nil {
		return "", err
	}
	name = filepath.Join(config, ToolName, SettingsFile)
	return name, nil
}

func (settings *Settings) ReadFile(filename string) error {
	filename, err := settings.filename(filename)
	if err != nil {
		return errors.Wrap(err, "settings file name")
	}
	file, err := os.Open(filename)
	if err != nil {
		return errors.Wrap(err, "open settings file")
	}
	defer file.Close()
	_, err = settings.ReadFrom(file)
	return err
}

func (settings *Settings) WriteFile(filename string) error {
	filename, err := settings.filename(filename)
	if err != nil {
		return errors.Wrap(err, "settings file name")
	}
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
	c.Build.Configs = make(map[string]fetch.Config, len(settings.Build.Configs))
	for k, v := range settings.Build.Configs {
		c.Build.Configs[k] = v
	}
	c.Build.UseConfigs = make([]string, len(settings.Build.UseConfigs))
	copy(c.Build.UseConfigs, settings.Build.UseConfigs)
	return &c
}
