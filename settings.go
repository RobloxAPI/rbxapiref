package main

import (
	"github.com/robloxapi/rbxapiref/fetch"
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
	// Resources is the location of resource files.
	Resources string
	// Templates is the location of template files.
	Templates string
}

type SettingsOutput struct {
	// Root is the directory to which generated files will be written.
	Root string
	// Sub is a path that follows the output directory and precedes a
	// generated file path.
	Sub string
	// Resources is the path relative to the Base where generated resource
	// files will be written.
	Resources string
	// Manifest is the path relative to the base that points to the manifest
	// file.
	Manifest string
}
