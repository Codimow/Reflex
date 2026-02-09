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

// Manager runs and manages a child process.
type Manager struct {
	command   string
	cmd       *exec.Cmd
	outChan   chan Line
	wg        sync.WaitGroup
	mu        sync.Mutex
	isRunning bool
}

// NewManager creates a new process manager for the given command string.
func NewManager(command string) *Manager {
	return &Manager{
		command: command,
		outChan: make(chan Line),
	}
}

// Output returns the channel that receives lines of output from the process.
func (m *Manager) Output() <-chan Line {
	return m.outChan
}

// Start executes the command and begins streaming its output.
func (m *Manager) Start() error {
	m.mu.Lock()
	if m.isRunning {
		m.mu.Unlock()
		return nil // Already running
	}

	m.cmd = exec.Command("sh", "-c", m.command)
	m.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := m.cmd.StdoutPipe()
	if err != nil {
		m.mu.Unlock()
		return err
	}
	stderr, err := m.cmd.StderrPipe()
	if err != nil {
		m.mu.Unlock()
		return err
	}

	if err := m.cmd.Start(); err != nil {
		m.mu.Unlock()
		return err
	}

	m.isRunning = true
	m.mu.Unlock()

	m.wg.Add(2)
	go m.streamOutput(stdout, &m.wg)
	go m.streamOutput(stderr, &m.wg)

	go func() {
		m.wg.Wait()
		m.cmd.Wait()
		m.mu.Lock()
		m.isRunning = false
		m.mu.Unlock()
		close(m.outChan)
	}()

	return nil
}

// Stop terminates the running process.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning || m.cmd == nil || m.cmd.Process == nil {
		return nil // Not running
	}

	return syscall.Kill(-m.cmd.Process.Pid, syscall.SIGKILL)
}

func (m *Manager) streamOutput(r io.Reader, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		m.outChan <- Line{Text: scanner.Text()}
	}
}
