// cmd/codecat/exclusion.go
package main

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
)

// PathInfo holds information about a path being considered for exclusion.
type PathInfo struct {
	AbsPath    string // Absolute path on the filesystem
	RelPathCwd string // Path relative to CWD, using slashes
	BaseName   string // Final component of the path
	IsDir      bool   // Is the path a directory?
}

// Excluder defines the interface for checking if a path should be excluded.
type Excluder interface {
	IsExcluded(info PathInfo) (excluded bool, reason string, pattern string)
}

// DefaultExcluder implements the Excluder interface using basename and CWD-relative rules.
type DefaultExcluder struct {
	basenamePatterns       []string
	cwdRelativePatterns    []string
	manualFileSet          map[string]struct{} // Set of CWD-relative paths for -f files
	excludedDirRelPathsCwd map[string]string   // CWD-relative path -> causing pattern
	mu                     sync.RWMutex
}

// NewDefaultExcluder creates and initializes a DefaultExcluder.
func NewDefaultExcluder(basenamePatterns, cwdRelativePatterns, manualFiles []string) *DefaultExcluder {
	manualSet := make(map[string]struct{}, len(manualFiles))
	for _, f := range manualFiles {
		manualSet[f] = struct{}{}
	}

	return &DefaultExcluder{
		basenamePatterns:       basenamePatterns,
		cwdRelativePatterns:    cwdRelativePatterns,
		manualFileSet:          manualSet,
		excludedDirRelPathsCwd: make(map[string]string),
	}
}

// IsExcluded implements the Excluder interface with ancestor checking.
func (e *DefaultExcluder) IsExcluded(info PathInfo) (excluded bool, reason string, pattern string) {
	// --- HIGHEST PRIORITY: Check if the file was specified with -f ---
	// Since `processManualFiles` runs first, this check prevents the scanner from
	// double-processing or applying its own exclusion logic to -f files.
	// This doesn't stop gocodewalker's gitignore, but that's okay because
	// processManualFiles has already added the content.
	if _, isManual := e.manualFileSet[info.RelPathCwd]; isManual {
		slog.Debug("Exclusion check: path is a manual file, skipping further checks.", "path", info.RelPathCwd)
		// We return true to "exclude" it from the *scanner's* processing,
		// because it has already been handled.
		return true, "manual file override", ""
	}

	// --- ANCESTOR CHECKS ---
	pathParts := strings.Split(filepath.ToSlash(info.RelPathCwd), "/")
	if len(pathParts) > 1 {
		for _, part := range pathParts[:len(pathParts)-1] {
			if match, p := matchesGlob(part, e.basenamePatterns); match {
				slog.Debug("Exclusion check: path excluded due to ancestor basename match", "path", info.RelPathCwd, "ancestor", part, "pattern", p)
				return true, fmt.Sprintf("ancestor %s basename match", part), p
			}
		}
	}

	currentParent := info.RelPathCwd
	for {
		currentParent = filepath.Dir(currentParent)
		if currentParent == "." || currentParent == "" || currentParent == "/" {
			break
		}
		if match, p := matchesGlob(currentParent, e.cwdRelativePatterns); match {
			slog.Debug("Exclusion check: path excluded due to ancestor CWD exact/glob match", "path", info.RelPathCwd, "ancestorDir", currentParent, "pattern", p)
			return true, fmt.Sprintf("ancestor %s CWD match", currentParent), p
		}
		for _, patt := range e.cwdRelativePatterns {
			cleanPattern := strings.TrimRight(patt, `\/`)
			if cleanPattern != "" && strings.HasPrefix(currentParent, cleanPattern+"/") {
				slog.Debug("Exclusion check: path excluded due to ancestor CWD prefix match", "path", info.RelPathCwd, "ancestorDir", currentParent, "pattern", patt)
				return true, fmt.Sprintf("ancestor %s CWD prefix match", currentParent), patt
			}
		}
	}

	// --- CURRENT ITEM CHECKS (if not excluded by an ancestor) ---
	if match, p := matchesGlob(info.BaseName, e.basenamePatterns); match {
		slog.Debug("Exclusion check: item excluded by basename", "path", info.RelPathCwd, "basename", info.BaseName, "pattern", p)
		if info.IsDir {
			e.mu.Lock()
			if _, exists := e.excludedDirRelPathsCwd[info.RelPathCwd]; !exists {
				e.excludedDirRelPathsCwd[info.RelPathCwd] = p
			}
			e.mu.Unlock()
		}
		return true, "basename match", p
	}

	for _, p := range e.cwdRelativePatterns {
		match, _ := filepath.Match(p, info.RelPathCwd)
		if !match && info.IsDir && strings.HasSuffix(p, "/") {
			match, _ = filepath.Match(strings.TrimRight(p, "/"), info.RelPathCwd)
		}
		if match {
			slog.Debug("Exclusion check: item excluded by CWD-relative pattern", "path", info.RelPathCwd, "pattern", p)
			if info.IsDir {
				e.mu.Lock()
				if _, exists := e.excludedDirRelPathsCwd[info.RelPathCwd]; !exists {
					e.excludedDirRelPathsCwd[info.RelPathCwd] = p
				}
				e.mu.Unlock()
			}
			return true, "CWD-relative match", p
		}
	}

	slog.Debug("Exclusion check: path not excluded", "path", info.RelPathCwd)
	return false, "", ""
}
