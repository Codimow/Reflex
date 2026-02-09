package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Codimow/Reflex/internal/process"
	"github.com/Codimow/Reflex/internal/ui"
	"github.com/Codimow/Reflex/internal/watcher"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: reflex <command>")
		fmt.Println("Example: reflex \"npm run dev\"")
		os.Exit(1)
	}
	command := os.Args[1]

	extensions := []string{".js", ".ts", ".jsx", ".tsx", ".css", ".mdx", ".json"}
	p := tea.NewProgram(ui.New(), tea.WithAltScreen())

	// This goroutine is the main controller for the application.
	go func() {
		// 1. Initialize the file watcher
		watcherEvents, err := watcher.New(".", extensions)
		if err != nil {
			log.Fatalf("could not create file watcher: %v", err)
		}

		// 2. Start the initial process
		p.Send(ui.StatusUpdateMsg{Status: "Starting process..."})
		proc := startProcess(p, command)

		// 3. Start the main event loop
		for {
			select {
			case <-watcherEvents:
				// On file change, restart the process
				p.Send(ui.StatusUpdateMsg{Status: "Restarting process..."})
				if proc != nil {
					proc.Stop()
				}
				// A small delay allows the old process to die gracefully
				time.Sleep(250 * time.Millisecond)
				p.Send(ui.ClearLogsMsg{})
				proc = startProcess(p, command)
			}
		}
	}()

	// Run the UI. This is a blocking call.
	if _, err := p.Run(); err != nil {
		log.Fatalf("error running UI: %v", err)
	}
}

// startProcess is a helper function to create, start, and listen to a new process.
func startProcess(p *tea.Program, command string) *process.Manager {
	proc := process.NewManager(command)
	if err := proc.Start(); err != nil {
		// Can't use log.Fatal here as it would exit the whole program
		log.Printf("failed to start process: %v", err)
		return nil
	}

	p.Send(ui.StatusUpdateMsg{Status: "Running"})

	// This goroutine streams the process output to the UI
	go func() {
		for line := range proc.Output() {
			p.Send(ui.ProcessOutputLineMsg{Line: line.Text})
		}
	}()

	return proc
}
