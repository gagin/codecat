// cmd/codecat/walk.go
package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	gocodewalker "github.com/boyter/gocodewalker"
)

// generateConcatenatedCode walks directories, processes files, and generates the output.
func generateConcatenatedCode(
	cwd string,
	scanDirs []string,
	exts map[string]struct{},
	manualFilePaths []string,
	excludeBasenames []string,
	projectExcludePatterns []string,
	flagExcludePatterns []string,
	useGitignore bool,
	header, marker string,
	noScan bool,
	useLegacyFormat bool,
	xmlEscapeContent bool,
) (
	output string,
	includedFiles []FileInfo,
	emptyFiles []string,
	errorFiles map[string]error,
	totalSize int64,
	returnedErr error,
) {
	slog.Debug("generateConcatenatedCode received extensions map", "exts_keys", mapsKeys(exts))

	var outputBuilder strings.Builder
	if !useLegacyFormat { // XML Mode
		outputBuilder.WriteString("<codebase>\n")
		if header != "" {
			outputBuilder.WriteString("  <description><![CDATA[")
			outputBuilder.WriteString(strings.TrimSpace(header))
			outputBuilder.WriteString("]]]]><![CDATA[></description>\n")
		}
	} else { // Legacy Mode
		if header != "" {
			outputBuilder.WriteString(header)
		}
	}

	includedFiles = make([]FileInfo, 0)
	emptyFiles = make([]string, 0)
	errorFiles = make(map[string]error)
	processedAbsPaths := make(map[string]bool)
	totalSize = 0

	// --- Pre-validate and Combine Exclude Patterns ---
	validBasenameExcludes := make([]string, 0, len(excludeBasenames))
	for _, pattern := range excludeBasenames {
		if _, errMatch := filepath.Match(pattern, "a"); errMatch != nil {
			slog.Warn("Invalid global exclude basename pattern syntax, ignoring.",
				"pattern", pattern, "error", errMatch)
		} else {
			validBasenameExcludes = append(validBasenameExcludes, pattern)
		}
	}
	slog.Debug("Using validated basename exclude patterns", "patterns", validBasenameExcludes)

	cwdRelativeExcludePatterns := []string{}
	combinedCwdExcludes := append([]string{}, projectExcludePatterns...)
	combinedCwdExcludes = append(combinedCwdExcludes, flagExcludePatterns...)
	for _, pattern := range combinedCwdExcludes {
		source := tern(contains(flagExcludePatterns, pattern), "flag", "project")
		if _, errMatch := filepath.Match(pattern, "a"); errMatch != nil {
			slog.Warn("Invalid CWD-relative exclude pattern syntax, ignoring.",
				"pattern", pattern, "source", source, "error", errMatch)
			continue
		}
		cwdRelativeExcludePatterns = append(cwdRelativeExcludePatterns, pattern)
	}
	slog.Debug("Using combined CWD-relative exclude patterns", "patterns", cwdRelativeExcludePatterns)

	// --- Process Manually Specified Files (-f) ---
	// This runs first to mark paths as processed. The -f override happens in the excluder logic.
	processManualFiles(
		cwd,
		manualFilePaths,
		marker,
		&outputBuilder,
		processedAbsPaths, // This is intentionally processed first
		&includedFiles,
		&emptyFiles,
		errorFiles,
		&totalSize,
		useLegacyFormat,
		xmlEscapeContent,
	)

	// --- Perform Directory Scan ---
	shouldScan := !noScan && len(scanDirs) > 0
	if shouldScan {
		// Pass manual file paths to the excluder so it can grant them priority.
		excluder := NewDefaultExcluder(validBasenameExcludes, cwdRelativeExcludePatterns, manualFilePaths)

		if len(exts) == 0 && len(manualFilePaths) == 0 {
			slog.Warn("Scanning requested, but no extensions/manual files provided. Scan will find nothing.")
		}
		slog.Info("Starting file scan.", "scanDirs", scanDirs, "useGitignore", useGitignore)

		fileListQueue := make(chan *gocodewalker.File, 100)
		fileWalker := gocodewalker.NewFileWalker(cwd, fileListQueue)
		fileWalker.IgnoreGitIgnore = !useGitignore
		fileWalker.IgnoreIgnoreFile = !useGitignore

		var walkErr error
		var firstWalkError error
		processingDone := make(chan struct{})

		go func() {
			defer close(processingDone)
			walkerErrorHandler := func(e error) bool {
				slog.Warn("Error reported by file walker.", "scanDir", cwd, "error", e)
				if firstWalkError == nil {
					firstWalkError = e
				}
				return true
			}
			fileWalker.SetErrorHandler(walkerErrorHandler)
			walkErr = fileWalker.Start()
		}()

		for f := range fileListQueue {
			absPath := f.Location
			if processedAbsPaths[absPath] {
				continue
			}

			isInScanDir := false
			for _, dir := range scanDirs {
				if absPath == dir || strings.HasPrefix(absPath, dir+string(filepath.Separator)) {
					isInScanDir = true
					break
				}
			}
			if !isInScanDir {
				continue
			}

			baseName := filepath.Base(absPath)
			relPathCwd, _ := filepath.Rel(cwd, absPath)
			relPathCwd = filepath.ToSlash(relPathCwd)

			fileInfo, statErr := os.Stat(absPath)
			if statErr != nil {
				errorFiles[relPathCwd] = statErr
				processedAbsPaths[absPath] = true
				continue
			}

			isDir := fileInfo.IsDir()
			pathInfo := PathInfo{AbsPath: absPath, RelPathCwd: relPathCwd, BaseName: baseName, IsDir: isDir}
			excluded, reason, pattern := excluder.IsExcluded(pathInfo)
			if excluded {
				logMsg := tern(isDir, "Excluding directory and its contents.", "Excluding file.")
				slog.Log(nil, slog.LevelDebug, logMsg, "path", relPathCwd, "reason", reason, "pattern", pattern)
				processedAbsPaths[absPath] = true
				continue
			}

			if isDir {
				processedAbsPaths[absPath] = true
				continue
			}

			currentExt := strings.ToLower(filepath.Ext(baseName))
			_, extAllowed := exts[currentExt]
			if len(exts) > 0 && !extAllowed {
				processedAbsPaths[absPath] = true
				continue
			}

			content, errRead := os.ReadFile(absPath)
			if errRead != nil {
				errorFiles[relPathCwd] = errRead
			} else if len(content) == 0 {
				emptyFiles = append(emptyFiles, relPathCwd)
			} else {
				fileSize := fileInfo.Size()
				appendFileContent(&outputBuilder, marker, relPathCwd, content, useLegacyFormat, xmlEscapeContent)
				includedFiles = append(includedFiles, FileInfo{Path: relPathCwd, Size: fileSize, IsManual: false})
				totalSize += fileSize
			}
			processedAbsPaths[absPath] = true
		}
		<-processingDone

		finalWalkError := walkErr
		if finalWalkError == nil && firstWalkError != nil {
			finalWalkError = firstWalkError
		}
		if returnedErr == nil && finalWalkError != nil {
			returnedErr = fmt.Errorf("file walk operation failed for '%s': %w", cwd, finalWalkError)
		}

	} else if noScan {
		slog.Info("Skipping directory scan due to --no-scan flag.")
	}

	if !useLegacyFormat {
		outputBuilder.WriteString("</codebase>\n")
	}

	output = outputBuilder.String()
	return
}

func getUnfilteredFileList(cwd string, scanDirs []string) ([]string, error) {
	allFiles := make(map[string]struct{})
	for _, dir := range scanDirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				relPath, relErr := filepath.Rel(cwd, path)
				if relErr != nil {
					allFiles[filepath.ToSlash(path)] = struct{}{}
				} else {
					allFiles[filepath.ToSlash(relPath)] = struct{}{}
				}
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error during unfiltered walk of %s: %w", dir, err)
		}
	}
	result := make([]string, 0, len(allFiles))
	for path := range allFiles {
		result = append(result, path)
	}
	sort.Strings(result)
	return result, nil
}
