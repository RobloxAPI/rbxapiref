package main

import (
	"github.com/robloxapi/rbxapiref/fetch"
)

const (
	ToolName     = "rbxapiref"
	SettingsFile = "settings.json"
)

const (
	ArchiveURL = "https://raw.githubusercontent.com/RobloxAPI/archive/master/"
	CDNURL     = "https://setup.rbxcdn.com/"
	DevHubURL  = "developer.roblox.com/api-reference"
)

var DefaultSettings = &Settings{
	Input: SettingsInput{
		Resources: "resources",
		Templates: "templates",
	},
	Output: SettingsOutput{
		Root:      ".",
		Sub:       "ref",
		Resources: "res",
		Manifest:  "manifest",
	},
	Configs: map[string]fetch.Config{
		"Archive": {
			Builds:             fetch.NewLocation(ArchiveURL + "builds.json"),
			Latest:             fetch.NewLocation(ArchiveURL + "latest.json"),
			APIDump:            fetch.NewLocation(ArchiveURL + "data/api-dump/json/$HASH.json"),
			ReflectionMetadata: fetch.NewLocation(ArchiveURL + "data/reflection-metadata/xml/$HASH.xml"),
			ExplorerIcons:      fetch.NewLocation(CDNURL + "$HASH-RobloxStudio.zip#RobloxStudioBeta.exe"),
		},
		"Production": {
			Builds:             fetch.NewLocation(CDNURL + "DeployHistory.txt"),
			Latest:             fetch.NewLocation(CDNURL + "versionQTStudio"),
			APIDump:            fetch.NewLocation(CDNURL + "$HASH-API-Dump.json"),
			ReflectionMetadata: fetch.NewLocation(CDNURL + "$HASH-RobloxStudio.zip#ReflectionMetadata.xml"),
			ExplorerIcons:      fetch.NewLocation(CDNURL + "$HASH-RobloxStudio.zip#RobloxStudioBeta.exe"),
		},
	},
	UseConfigs: []string{
		"Archive",
		"Production",
	},
}
