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
			// Ignore node_modules and .git directories
			if info.Name() == "node_modules" || info.Name() == ".git" {
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
