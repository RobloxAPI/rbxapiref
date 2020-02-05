// +build go1.13

package main

import (
	"os"
)

func userConfigDir() (string, error) {
	return os.UserConfigDir()
}
