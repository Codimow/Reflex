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
	p := tea.NewProgram(ui.NewModel(), tea.WithAltScreen())

	go func() {
		watcherEvents, err := watcher.New(".", extensions)
		if err != nil {
			log.Fatalf("could not create file watcher: %v", err)
		}

		p.Send(ui.StatusUpdateMsg{Status: "Starting process..."})
		proc := startProcess(p, command)

		// --- Main Event Loop ---
		for {
			select {
			case <-watcherEvents:
				p.Send(ui.StatusUpdateMsg{Status: "Restarting process..."})
				if proc != nil {
					proc.Stop()
				}
				// Add a small delay to allow the process to terminate cleanly
				time.Sleep(250 * time.Millisecond)
				p.Send(ui.ClearLogsMsg{})
				proc = startProcess(p, command)
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		log.Fatalf("error running program: %v", err)
	}
}

func startProcess(p *tea.Program, command string) *process.Manager {
	proc := process.NewManager(command)
	if err := proc.Start(); err != nil {
		log.Printf("failed to start process: %v", err)
		return nil
	}

	p.Send(ui.StatusUpdateMsg{Status: "Running"})

	go func() {
		for line := range proc.Output() {
			p.Send(ui.ProcessOutputLineMsg{Line: line.Text})
		}
	}()

	return proc
}
