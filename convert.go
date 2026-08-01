package main

import (
	"fmt"
	"path/filepath"
)

// Convert parses the EML file at inputPath and returns the resulting Markdown.
//
// For now this is a stub that simply reports that the file has been converted.
func Convert(inputPath string) string {
	return fmt.Sprintf("File %s has been converted", filepath.Base(inputPath))
}
