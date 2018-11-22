package main

import (
	"fmt"
	"github.com/robloxapi/rbxapi/rbxapijson"
	"github.com/robloxapi/rbxapiref/fetch"
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
