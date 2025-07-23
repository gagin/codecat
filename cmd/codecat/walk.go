// cmd/codecat/walk.go
package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
			outputBuilder.WriteString("]]></description>\n")
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
	processManualFiles(
		cwd,
		manualFilePaths,
		marker,
		&outputBuilder,
		processedAbsPaths,
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
		excluder := NewDefaultExcluder(validBasenameExcludes, cwdRelativeExcludePatterns)

		if len(exts) == 0 && len(manualFilePaths) == 0 {
			slog.Warn("Scanning requested, but no extensions/manual files provided. Scan will find nothing.")
		}
		slog.Info("Starting file scan.", "scanDirs", scanDirs, "useGitignore", useGitignore)

		// Pre-flight checks in main() ensure scanDirs are valid directories.
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

			if processedAbsPaths[absPath] {
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
				processedAbsPaths[absPath] = true
				continue
			}
			if len(content) == 0 {
				emptyFiles = append(emptyFiles, relPathCwd)
				processedAbsPaths[absPath] = true
				continue
			}
			fileSize := fileInfo.Size()
			appendFileContent(&outputBuilder, marker, relPathCwd, content, useLegacyFormat, xmlEscapeContent)
			includedFiles = append(includedFiles, FileInfo{Path: relPathCwd, Size: fileSize, IsManual: false})
			totalSize += fileSize
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

		if returnedErr == nil {
			slog.Info("File scan completed.")
		} else {
			slog.Error("File scan finished with errors.", "first_error", returnedErr)
		}
	} else if noScan {
		slog.Info("Skipping directory scan due to --no-scan flag.")
	} else if len(scanDirs) == 0 {
		slog.Info("Skipping directory scan as no scan directories were provided or determined.")
	}

	if !useLegacyFormat { // XML Mode
		outputBuilder.WriteString("</codebase>\n")
	}

	output = outputBuilder.String()
	return
}
