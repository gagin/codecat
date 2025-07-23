// cmd/codecat/preflight.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// preflightChecks validates that all specified directories and files exist before processing.
func preflightChecks(scanDirs []string, manualFiles []string, cwd string) error {
	var errorMessages []string

	// Check scan directories
	for _, dir := range scanDirs {
		info, err := os.Stat(dir)
		if err != nil {
			relDir, relErr := filepath.Rel(cwd, dir)
			if relErr != nil {
				relDir = dir // fallback to absolute if rel fails
			}
			if os.IsNotExist(err) {
				errorMessages = append(errorMessages, fmt.Sprintf("scan directory not found: %s", relDir))
			} else {
				errorMessages = append(errorMessages, fmt.Sprintf("cannot stat scan directory %s: %v", relDir, err))
			}
			continue
		}
		if !info.IsDir() {
			relDir, _ := filepath.Rel(cwd, dir)
			errorMessages = append(errorMessages, fmt.Sprintf("scan path is not a directory: %s", relDir))
		}
	}

	// Check manual files
	for _, file := range manualFiles {
		absPath := file
		if !filepath.IsAbs(file) {
			absPath = filepath.Join(cwd, file)
		}
		info, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				errorMessages = append(errorMessages, fmt.Sprintf("manual file not found: %s", file))
			} else {
				errorMessages = append(errorMessages, fmt.Sprintf("cannot stat manual file %s: %v", file, err))
			}
			continue
		}
		if info.IsDir() {
			errorMessages = append(errorMessages, fmt.Sprintf("manual path is a directory, not a file: %s", file))
		}
	}

	if len(errorMessages) > 0 {
		return fmt.Errorf("pre-flight checks failed:\n- %s", strings.Join(errorMessages, "\n- "))
	}

	return nil
}
