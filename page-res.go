package main

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func GenerateResPage(data *Data) error {
	fis, err := ioutil.ReadDir(data.Settings.Input.Resources)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(data.FilePath("resource"), 0666); err != nil {
		return err
	}
	for _, fi := range fis {
		if fi.IsDir() {
			continue
		}
		src, err := os.Open(filepath.Join(data.Settings.Input.Resources, fi.Name()))
		if err != nil {
			return err
		}
		dst, err := os.Create(data.FilePath("resource", fi.Name()))
		if err != nil {
			src.Close()
			return err
		}
		_, err = io.Copy(dst, src)
		dst.Close()
		src.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
