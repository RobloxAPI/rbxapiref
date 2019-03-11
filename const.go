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

const (
	ClassPath           = "class"
	EnumPath            = "enum"
	TypePath            = "type"
	FileExt             = ".html"
	MemberAnchorPrefix  = "member-"
	SectionAnchorPrefix = "section-"
	MainTitle           = "Roblox API Reference"
	TitleSep            = "-"
)

var DefaultSettings = &Settings{
	Input: SettingsInput{
		Resources: "resources",
		Templates: "templates",
	},
	Output: SettingsOutput{
		Root:         ".",
		Sub:          "ref",
		Resources:    "res",
		DocResources: "docres",
		Manifest:     "manifest",
	},
	Configs: map[string]fetch.Config{
		"Archive": {
			Builds:             fetch.NewLocations(ArchiveURL + "builds.json"),
			Latest:             fetch.NewLocations(ArchiveURL + "latest.json"),
			APIDump:            fetch.NewLocations(ArchiveURL + "data/api-dump/json/$HASH.json"),
			ReflectionMetadata: fetch.NewLocations(ArchiveURL + "data/reflection-metadata/xml/$HASH.xml"),
			ExplorerIcons: fetch.NewLocations(
				CDNURL+"$HASH-content-textures2.zip#ClassImages.PNG",
				CDNURL+"$HASH-RobloxStudio.zip#RobloxStudioBeta.exe",
			),
		},
		"Production": {
			Builds:             fetch.NewLocations(CDNURL + "DeployHistory.txt"),
			Latest:             fetch.NewLocations(CDNURL + "versionQTStudio"),
			APIDump:            fetch.NewLocations(CDNURL + "$HASH-API-Dump.json"),
			ReflectionMetadata: fetch.NewLocations(CDNURL + "$HASH-RobloxStudio.zip#ReflectionMetadata.xml"),
			ExplorerIcons: fetch.NewLocations(
				CDNURL+"$HASH-content-textures2.zip#ClassImages.PNG",
				CDNURL+"$HASH-RobloxStudio.zip#RobloxStudioBeta.exe",
			),
		},
	},
	UseConfigs: []string{
		"Archive",
		"Production",
	},
}
