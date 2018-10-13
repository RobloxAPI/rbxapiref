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
	// Manifest is the path relative to Sub that points to the manifest file.
	Manifest string
}

func (settings *Settings) ReadFrom(r io.Reader) (n int64, err error) {
	dw := NewDecodeWrapper(r)
	err = json.NewDecoder(dw).Decode(settings)
	if err != nil {
		return dw.BytesRead(), errors.Wrap(err, "decode settings file")
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

func (settings *Settings) ReadFile(filename string) error {
	if filename == "" {
		filename = filepath.Join(configdir.LocalConfig(ToolName), SettingsFile)
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
	if filename == "" {
		filename = filepath.Join(configdir.LocalConfig(ToolName), SettingsFile)
	}
	file, err := os.Create(filename)
	if err != nil {
		return errors.Wrap(err, "create settings file")
	}
	defer file.Close()
	_, err = settings.WriteTo(file)
	return err
}
