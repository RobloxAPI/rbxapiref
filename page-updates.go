package main

import (
	"os"
	"path/filepath"
	"strconv"
)

func GenerateUpdatesPage(data *Data) error {
	type args struct {
		Patches []Patch
		Year    int
		Years   []int
	}
	page := Page{
		Template: "updates",
		Title:    "Updates",
		Styles:   []Resource{{Name: "updates.css", Embed: true, ID: "updates-style"}},
		Scripts:  []Resource{{Name: "updates.js", Embed: true}},
	}

	src := data.Patches
	if len(src) == 0 {
		f, err := os.Create(data.FilePath("updates"))
		if err != nil {
			return err
		}
		page.Data = args{}
		GeneratePage(data, f, page)
		f.Close()
		return err
	}
	src = src[1:]
	patches := make([]Patch, len(src))
	for i := len(src) / 2; i >= 0; i-- {
		j := len(src) - 1 - i
		patches[i], patches[j] = src[j], src[i]
	}

	maxYear := patches[0].Info.Date.Year()
	minYear := patches[len(patches)-1].Info.Date.Year()
	patchesByYear := make([][]Patch, maxYear-minYear+1)
	years := make([]int, maxYear-minYear+1)
	for i := range years {
		years[i] = maxYear - i
	}
	{
		i := 0
		current := maxYear
		for j, patch := range patches {
			year := patch.Info.Date.Year()
			if year < current {
				if j > i {
					patchesByYear[maxYear-current] = patches[i:j]
				}
				current = year
				i = j
			}
		}
		if len(patches) > i {
			patchesByYear[maxYear-current] = patches[i:]
		}
	}
	{
		i := len(patches)
		epoch := patches[0].Info.Date.AddDate(0, -3, 0)
		for j, patch := range patches {
			if patch.Info.Date.Before(epoch) {
				i = j - 1
				break
			}
		}
		f, err := os.Create(data.FilePath("updates"))
		if err != nil {
			return err
		}
		page.Title = "Recent Updates"
		page.Data = args{patches[:i], 0, years}
		err = GeneratePage(data, f, page)
		f.Close()
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(data.FilePath("updates", "0")), 0666); err != nil {
		return err
	}
	for i, patches := range patchesByYear {
		year := maxYear - i
		f, err := os.Create(data.FilePath("updates", strconv.Itoa(year)))
		if err != nil {
			return err
		}
		page.Title = "Updates in " + strconv.Itoa(year)
		page.Data = args{patches, year, years}
		err = GeneratePage(data, f, page)
		f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
