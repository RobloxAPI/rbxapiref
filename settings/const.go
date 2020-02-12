package settings

import (
	"net/url"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/styles"
	"github.com/robloxapi/rbxapiref/builds"
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

func mustParseURL(rawurl string) url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		panic(err)
	}
	return *u
}

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
	Build: builds.Settings{
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
				Live: []fetch.Location{
					fetch.Location{
						Format: ".json",
						URL:    mustParseURL("https://versioncompatibility.api.roblox.com/GetCurrentClientVersionUpload/?apiKey=76e5a40c-3ae1-4028-9f10-7c62520bd94f&binaryType=WindowsStudio"),
					},
					fetch.Location{
						Format: ".json",
						URL:    mustParseURL("https://versioncompatibility.api.roblox.com/GetCurrentClientVersionUpload/?apiKey=76e5a40c-3ae1-4028-9f10-7c62520bd94f&binaryType=WindowsStudio64"),
					},
				},
			},
		},
		UseConfigs: []string{
			"Archive",
			"Production",
		},
	},
}

// Roblox studio light colors:
// Background Color                #FFFFFF
// Built-in Function Color         #00007F
// Comment Color                   #007F00
// Error Color                     #FF0000
// Find Selection Background Color #F6B93F
// Keyword Color                   #00007F
// Matching Word Background Color  #E2E6D6
// Number Color                    #007F7F
// Operator Color                  #7F7F00
// Preprocessor Color              #7F0000
// Selection Background Color      #6EA1F1
// Selection Color                 #FFFFFF
// String Color                    #7F007F
// Text Color                      #000000
// Warning Color                   #0000FF

// StyleRobloxLight is a light theme for code syntax highlighting.
var StyleRobloxLight = styles.Register(chroma.MustNewStyle("roblox-light", chroma.StyleEntries{
	chroma.Background:       "bg:#FFFFFF",
	chroma.LineHighlight:    "bg:#E2E6D6",
	chroma.LineNumbersTable: "#7F7F7F",
	chroma.LineNumbers:      "#7F7F7F",
	chroma.Error:            "#FF0000",
	chroma.Keyword:          "#00007F bold",
	chroma.Name:             "#000000",
	chroma.LiteralString:    "#7F007F",
	chroma.LiteralNumber:    "#007F7F",
	chroma.Operator:         "#7F7F00",
	chroma.OperatorWord:     "#00007F bold",
	chroma.Punctuation:      "#7F7F00",
	chroma.Comment:          "#007F00",
	chroma.CommentPreproc:   "#7F0000",
}))

// Roblox studio dark colors:
// Background Color                #252525
// Built-in Function Color         #84D6F7
// Comment Color                   #666666
// Error Color                     #FF0000
// Find Selection Background Color #FFF550
// Keyword Color                   #F86D7C
// Matching Word Background Color  #555555
// Number Color                    #FFC600
// Operator Color                  #CCCCCC
// Preprocessor Color              #66FFCC
// Selection Background Color      #2A2A2A
// Selection Color                 #999999
// String Color                    #ADF195
// Text Color                      #CCCCCC
// Warning Color                   #FF7315

// StyleRobloxDark is a dark theme for code syntax highlighting.
var StyleRobloxDark = styles.Register(chroma.MustNewStyle("roblox-dark", chroma.StyleEntries{
	chroma.Background:       "bg:#252525",
	chroma.LineHighlight:    "bg:#555555",
	chroma.LineNumbersTable: "#B4B4B4",
	chroma.LineNumbers:      "#B4B4B4",
	chroma.Error:            "#FF0000",
	chroma.Keyword:          "#F86D7C bold",
	chroma.Name:             "#CCCCCC",
	chroma.LiteralString:    "#ADF195",
	chroma.LiteralNumber:    "#FFC600",
	chroma.Operator:         "#CCCCCC",
	chroma.OperatorWord:     "#F86D7C bold",
	chroma.Punctuation:      "#CCCCCC",
	chroma.Comment:          "#666666",
	chroma.CommentPreproc:   "#66FFCC",
}))
