package settings

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/robloxapi/rbxapiref/builds"
	"github.com/robloxapi/rbxapiref/fetch"
	"github.com/robloxapi/rbxapiref/internal/binio"
)

type Settings struct {
	// Input specifies input settings.
	Input Input
	// Output specifies output settings.
	Output Output
	// Build specifies build settings.
	Build builds.Settings
}

type Input struct {
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

func (settings *Settings) ReadFrom(r io.Reader) (n int64, err error) {
	dw := binio.NewReader(r)
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
		return dw.BytesRead(), fmt.Errorf("decode settings file: %w", err)
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

	return dw.End()
}

func (settings *Settings) WriteTo(w io.Writer) (n int64, err error) {
	ew := binio.NewWriter(w)
	je := json.NewEncoder(ew)
	je.SetEscapeHTML(true)
	je.SetIndent("", "\t")
	err = je.Encode(settings)
	if err != nil {
		return ew.BytesWritten(), fmt.Errorf("encode settings file: %w", err)
	}
	return ew.End()
}

func (settings *Settings) filename(name string) (string, error) {
	// User-defined.
	if name != "" {
		return name, nil
	}

	// Portable, if present.
	name = FileName
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		return name, nil
	}

	// Local config.
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	name = filepath.Join(config, ToolName, FileName)
	return name, nil
}

func (settings *Settings) ReadFile(filename string) error {
	filename, err := settings.filename(filename)
	if err != nil {
		return fmt.Errorf("settings file name: %w", err)
	}
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open settings file: %w", err)
	}
	defer file.Close()
	_, err = settings.ReadFrom(file)
	return err
}

func (settings *Settings) WriteFile(filename string) error {
	filename, err := settings.filename(filename)
	if err != nil {
		return fmt.Errorf("settings file name: %w", err)
	}
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create settings file: %w", err)
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
