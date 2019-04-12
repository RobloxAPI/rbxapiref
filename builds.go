package main

import (
	"fmt"
	"github.com/anaminus/but"
	"github.com/pkg/errors"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"github.com/robloxapi/rbxapi/rbxapijson/diff"
	"github.com/robloxapi/rbxapiref/fetch"
	"sort"
	"time"
)

type Build struct {
	Config string
	Info   BuildInfo
	API    *rbxapijson.Root
}

type BuildInfo struct {
	Hash    string
	Date    time.Time
	Version fetch.Version
}

func (a BuildInfo) Equal(b BuildInfo) bool {
	if a.Hash != b.Hash {
		return false
	}
	if a.Version != b.Version {
		return false
	}
	if !a.Date.Equal(b.Date) {
		return false
	}
	return true
}

func (m BuildInfo) String() string {
	return fmt.Sprintf("%s; %s; %s", m.Hash, m.Date, m.Version)
}

func FetchBuilds(settings Settings) (builds []Build, err error) {
	client := &fetch.Client{CacheMode: fetch.CacheNone}
	for _, cfg := range settings.UseConfigs {
		client.Config = settings.Configs[cfg]
		bs, err := client.Builds()
		if err != nil {
			return nil, errors.WithMessage(err, "fetch build")
		}
		for _, b := range bs {
			builds = append(builds, Build{Config: cfg, Info: BuildInfo(b)})
		}
	}

	// Collapse adjacent builds of equal versions.
	b := builds[:0]
	for _, build := range builds {
		if len(b) == 0 || build.Info.Version != b[len(b)-1].Info.Version {
			b = append(b, build)
		}
	}
	for i := len(b); i < len(builds); i++ {
		builds[i] = Build{}
	}
	builds = b

	// Rewind to current live build.
	if live, err := client.Live(); err != nil {
		but.Logf("fetch live build: %v\n", err)
	} else if live.Hash != "" {
		for i := len(builds) - 1; i >= 0; i-- {
			if builds[i].Info.Hash == live.Hash {
				builds = builds[:i+1]
				break
			}
		}
	}

	sort.Slice(builds, func(i, j int) bool {
		return builds[i].Info.Date.Before(builds[j].Info.Date)
	})
	return builds, nil
}

func MergeBuilds(settings Settings, cached []Patch, builds []Build) (patches []Patch, err error) {
	client := &fetch.Client{CacheMode: fetch.CacheTemp}
	var latest *Build
loop:
	for _, build := range builds {
		for _, patch := range cached {
			if !build.Info.Equal(patch.Info) {
				// Not relevant; skip.
				continue
			}
			// Current build has a cached version.
			if latest == nil {
				if patch.Prev != nil {
					// Cached build is now the first, but was not originally;
					// actions are stale.
					but.Log("STALE", patch.Info)
					break
				}
			} else {
				if patch.Prev == nil {
					// Cached build was not originally the first, but now is;
					// actions are stale.
					but.Log("STALE", patch.Info)
					break
				}
				if !latest.Info.Equal(*patch.Prev) {
					// Latest build does not match previous build; actions are
					// stale.
					but.Log("STALE", patch.Info)
					break
				}
			}
			// Cached actions are still fresh; set them directly.
			patches = append(patches, patch)
			latest = &Build{Info: patch.Info, Config: patch.Config}
			continue loop
		}
		but.Log("NEW", build.Info)
		client.Config = settings.Configs[build.Config]
		root, err := client.APIDump(build.Info.Hash)
		if but.IfErrorf(err, "%s: fetch build %s", build.Config, build.Info.Hash) {
			continue
		}
		build.API = root
		var actions []Action
		if latest == nil {
			// First build; compare with nothing.
			actions = WrapActions((&diff.Diff{Prev: nil, Next: build.API}).Diff())
		} else {
			if latest.API == nil {
				// Previous build was cached; fetch its data to compare with
				// current build.
				client.Config = settings.Configs[latest.Config]
				root, err := client.APIDump(latest.Info.Hash)
				if but.IfErrorf(err, "%s: fetch build %s", latest.Config, latest.Info.Hash) {
					continue
				}
				latest.API = root
			}
			actions = WrapActions((&diff.Diff{Prev: latest.API, Next: build.API}).Diff())
		}
		patch := Patch{Stale: true, Info: build.Info, Config: build.Config, Actions: actions}
		if latest != nil {
			prev := latest.Info
			patch.Prev = &prev
		}
		patches = append(patches, patch)
		b := build
		latest = &b
	}

	// Set action indices.
	for i, patch := range patches {
		for j := range patch.Actions {
			patches[i].Actions[j].Index = j
		}
	}

	return patches, nil
}
