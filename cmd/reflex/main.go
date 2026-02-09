// Package main is the entry point for Reflex, a file-watching process runner.
//
// Architecture Overview:
// ======================
//
// Reflex follows an event-driven architecture with three main components:
//
//   ┌─────────────┐     Events      ┌─────────────────┐     Messages    ┌─────────┐
//   │   Watcher   │ ──────────────► │   Controller    │ ──────────────► │   UI    │
//   │ (fsnotify)  │                 │  (main loop)    │                 │ (TUI)   │
//   └─────────────┘                 └────────┬────────┘                 └─────────┘
//                                            │
//                                            │ manages
//                                            ▼
//                                   ┌─────────────────┐
//                                   │ Process Manager │
//                                   │  (child proc)   │
//                                   └─────────────────┘
//
// Flow:
// 1. Watcher monitors the filesystem and emits events on file changes
// 2. Controller receives events and orchestrates process restarts
// 3. Process Manager handles the child process lifecycle (start/stop/output)
// 4. UI displays status and streams process output to the terminal
//
// Shutdown:
// - SIGINT/SIGTERM triggers graceful shutdown via context cancellation
// - Controller stops the child process before exiting
// - All goroutines clean up via context or channel closure

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Codimow/Reflex/internal/process"
	"github.com/Codimow/Reflex/internal/ui"
	"github.com/Codimow/Reflex/internal/watcher"
	tea "github.com/charmbracelet/bubbletea"
)

// Default file extensions to watch.
// These cover common web development file types.
// Note: .json is intentionally excluded because build tools (Next.js, npm, etc.)
// frequently modify package.json, lock files, and other JSON configs, causing
// unwanted restarts.
var defaultExtensions = []string{
	".js", ".ts", ".jsx", ".tsx",
	".css", ".scss", ".sass",
	".mdx", ".md",
	".html", ".vue", ".svelte",
}

// restartDebounce is the delay between detecting a file change and restarting
// the process. This prevents rapid restarts when multiple files change at once
// (e.g., during a git checkout or editor save-all).
const restartDebounce = 250 * time.Millisecond

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run is the main application logic, separated for cleaner error handling.
func run() error {
	// Parse command line arguments
	command, err := parseArgs()
	if err != nil {
		return err
	}

	// Create a root context that cancels on SIGINT or SIGTERM.
	// This enables graceful shutdown when the user presses Ctrl+C.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Initialize the Bubbletea UI program with alternate screen mode
	// (preserves the user's terminal history on exit)
	program := tea.NewProgram(ui.New(), tea.WithAltScreen())

	// WaitGroup to coordinate goroutine shutdown
	var wg sync.WaitGroup

	// Channel to propagate fatal errors from goroutines
	errChan := make(chan error, 1)

	// Start the controller goroutine that orchestrates watcher → process → UI
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := runController(ctx, program, command); err != nil {
			// Send error to main goroutine (non-blocking)
			select {
			case errChan <- err:
			default:
			}
			// Quit the UI on controller error
			program.Send(tea.Quit())
		}
	}()

	// Run the UI (blocking). This returns when:
	// - User presses 'q' or Ctrl+C in the UI
	// - program.Quit() is called
	// - An error occurs
	_, uiErr := program.Run()

	// Cancel context to signal all goroutines to stop
	cancel()

	// Wait for controller to finish cleanup
	wg.Wait()

	// Check for controller errors first (more informative)
	select {
	case err := <-errChan:
		return err
	default:
	}

	return uiErr
}

// parseArgs validates and returns the command to run.
func parseArgs() (string, error) {
	if len(os.Args) < 2 {
		return "", fmt.Errorf("usage: reflex <command>\n\nExample:\n  reflex \"npm run dev\"\n  reflex \"go run .\"")
	}
	return os.Args[1], nil
}

// runController is the main event loop that coordinates the watcher,
// process manager, and UI. It runs until the context is cancelled.
func runController(ctx context.Context, program *tea.Program, command string) error {
	// Initialize the file watcher
	watcherEvents, err := watcher.New(".", defaultExtensions)
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Track the current process (may be nil if not running)
	var currentProc *process.Manager

	// Ensure we always clean up the process on exit
	defer func() {
		if currentProc != nil {
			program.Send(ui.StatusUpdateMsg{Status: "Stopping..."})
			currentProc.Stop()
		}
	}()

	// Start the initial process
	program.Send(ui.StatusUpdateMsg{Status: "Starting process..."})
	currentProc = startProcess(ctx, program, command)

	// Main event loop: wait for file changes or shutdown signal
	for {
		select {
		case <-ctx.Done():
			// Graceful shutdown requested (Ctrl+C or SIGTERM)
			log.Println("Shutdown signal received, cleaning up...")
			return nil

		case event, ok := <-watcherEvents:
			if !ok {
				// Watcher channel closed (shouldn't happen normally)
				return fmt.Errorf("file watcher closed unexpectedly")
			}

			// File change detected — restart the process
			log.Printf("File changed: %s", event.Path)
			program.Send(ui.StatusUpdateMsg{Status: "Restarting..."})

			// Stop the current process if running
			if currentProc != nil {
				currentProc.Stop()
				currentProc = nil
			}

			// Debounce: wait a bit for more changes to settle
			// This prevents rapid restarts during batch file operations
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(restartDebounce):
			}

			// Clear logs and start fresh
			program.Send(ui.ClearLogsMsg{})
			currentProc = startProcess(ctx, program, command)
		}
	}
}

// startProcess creates and starts a new child process, streaming its output
// to the UI. Returns the process manager (or nil on failure).
func startProcess(ctx context.Context, program *tea.Program, command string) *process.Manager {
	proc := process.NewManager(command)

	if err := proc.Start(); err != nil {
		log.Printf("Failed to start process: %v", err)
		program.Send(ui.StatusUpdateMsg{Status: "Error: failed to start"})
		program.Send(ui.ProcessOutputLineMsg{Line: fmt.Sprintf("Error: %v", err)})
		return nil
	}

	program.Send(ui.StatusUpdateMsg{Status: "Running"})

	// Stream process output to the UI in a separate goroutine.
	// This goroutine exits when:
	// - The process exits (output channel closes)
	// - The context is cancelled (we stop reading)
	go func() {
		for {
			select {
			case <-ctx.Done():
				// Shutdown requested, stop streaming
				return

			case line, ok := <-proc.Output():
				if !ok {
					// Process exited, output channel closed
					program.Send(ui.StatusUpdateMsg{Status: "Process exited"})
					return
				}
				program.Send(ui.ProcessOutputLineMsg{Line: line.Text})
			}
		}
	}()

	return proc
}
