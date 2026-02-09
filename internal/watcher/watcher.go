package watcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// Event represents a single file system event.
type Event struct {
	Path string // The path to the file that changed.
}

// ignoredDirs contains directory names that should be skipped during watching.
// These are typically generated/dependency directories that cause spurious restarts.
var ignoredDirs = map[string]bool{
	"node_modules": true,
	".next":        true,
	".git":         true,
	"dist":         true,
	"build":        true,
	".cache":       true,
}

// shouldIgnoreFile returns true if the file path should be ignored from triggering restarts.
// This filters out lock files and other generated files that tools frequently modify.
func shouldIgnoreFile(path string) bool {
	base := filepath.Base(path)

	// Ignore lock files: package-lock.json, yarn.lock, pnpm-lock.yaml, etc.
	if strings.HasSuffix(base, "-lock.json") || strings.HasSuffix(base, ".lock") {
		return true
	}

	return false
}

// New creates a new file system watcher and returns a channel of events.
// It watches the given root path recursively for files with the specified extensions.
func New(rootPath string, extensions []string) (<-chan Event, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	eventChan := make(chan Event)

	// Walk the initial directory tree and add all subdirectories to the watcher.
	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip ignored directories (node_modules, .next, .git, dist, build, .cache)
			if ignoredDirs[info.Name()] {
				return filepath.SkipDir
			}
			return watcher.Add(path)
		}
		return nil
	})

	if err != nil {
		watcher.Close()
		return nil, err
	}

	// Goroutine to handle events from fsnotify and filter them.
	go func() {
		defer watcher.Close()
		defer close(eventChan)

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op.Has(fsnotify.Write) || event.Op.Has(fsnotify.Create) {
					// Skip files that should be ignored (lock files, etc.)
					if shouldIgnoreFile(event.Name) {
						continue
					}

					isTarget := false
					for _, ext := range extensions {
						if strings.HasSuffix(event.Name, ext) {
							isTarget = true
							break
						}
					}
					if isTarget {
						eventChan <- Event{Path: event.Name}
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("watcher error: %v", err)
			}
		}
	}()

	return eventChan, nil
}
