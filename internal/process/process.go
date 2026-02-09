package process

import (
	"bufio"
	"io"
	"os/exec"
	"sync"
	"syscall"
)

// Line represents a single line of output from the process.
type Line struct {
	Text string
}

// Manager manages a child process.
type Manager struct {
	command string
	cmd     *exec.Cmd
	output  chan Line
	done    chan struct{}
	mu      sync.Mutex
	started bool
}

// NewManager creates a new Manager for the given command.
func NewManager(command string) *Manager {
	return &Manager{
		command: command,
		output:  make(chan Line, 100),
		done:    make(chan struct{}),
	}
}

// Start runs the command via sh -c and captures stdout/stderr.
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return nil
	}

	m.cmd = exec.Command("sh", "-c", m.command)

	// Create a process group for clean termination
	m.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := m.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := m.cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := m.cmd.Start(); err != nil {
		return err
	}

	m.started = true

	// Combine stdout and stderr
	var wg sync.WaitGroup
	wg.Add(2)

	readLines := func(r io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			select {
			case <-m.done:
				return
			case m.output <- Line{Text: scanner.Text()}:
			}
		}
	}

	go readLines(stdout)
	go readLines(stderr)

	// Close output channel when both readers finish
	go func() {
		wg.Wait()
		m.cmd.Wait()
		close(m.output)
	}()

	return nil
}

// Stop kills the process and all its children.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started || m.cmd == nil || m.cmd.Process == nil {
		return nil
	}

	// Signal done to stop readers
	select {
	case <-m.done:
		// Already closed
	default:
		close(m.done)
	}

	// Kill the entire process group
	pgid, err := syscall.Getpgid(m.cmd.Process.Pid)
	if err == nil {
		syscall.Kill(-pgid, syscall.SIGKILL)
	}

	return m.cmd.Wait()
}

// Output returns a channel of output lines.
func (m *Manager) Output() <-chan Line {
	return m.output
}
