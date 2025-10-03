//go:build linux

package main

import (
	"os"
	"strings"
)
// isHiddenUnix checks hidden attribute on Unix-like systems
func isHiddenDir(path string, info os.FileInfo) (bool, error) {
	// On Unix, files starting with . are considered hidden
	return strings.HasPrefix(info.Name(), "."), nil
}
