// Package claude_agent_sdk provides process management for Claude Code CLI.
package claude_agent_sdk

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/guyskk/ccc/internal/logger"
)

// Process manages a subprocess (claude command).
type Process struct {
	cmd    *exec.Cmd
	ctx    context.Context
	cancel context.CancelFunc

	stdout io.ReadCloser
	stderr io.ReadCloser

	logger  logger.Logger
	mu      sync.Mutex
	started bool
	done    bool
}

// NewProcess creates a new Process from an exec.Cmd.
func NewProcess(cmd *exec.Cmd, logger logger.Logger) *Process {
	return &Process{
		cmd:    cmd,
		logger: logger,
	}
}

// Start starts the process.
func (p *Process) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return fmt.Errorf("process already started")
	}

	// Create pipes
	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	p.stdout = stdout

	stderr, err := p.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	p.stderr = stderr

	// Start the process
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	p.started = true
	p.logger.Debug("process started", logger.IntField("pid", p.cmd.Process.Pid))

	return nil
}

// Wait waits for the process to complete.
func (p *Process) Wait() error {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return fmt.Errorf("process not started")
	}
	p.mu.Unlock()

	err := p.cmd.Wait()

	p.mu.Lock()
	p.done = true
	p.mu.Unlock()

	if err != nil {
		p.logger.Error("process exited with error", logger.StringField("error", err.Error()))
		return err
	}

	p.logger.Debug("process completed successfully")
	return nil
}

// Kill terminates the process.
func (p *Process) Kill() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd.Process == nil {
		return fmt.Errorf("process not running")
	}

	p.logger.Debug("killing process", logger.IntField("pid", p.cmd.Process.Pid))

	if err := p.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}

	return nil
}

// Signal sends a signal to the process.
func (p *Process) Signal(sig syscall.Signal) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd.Process == nil {
		return fmt.Errorf("process not running")
	}

	p.logger.Debug("sending signal to process",
		logger.IntField("pid", p.cmd.Process.Pid),
		logger.StringField("signal", sig.String()))

	if err := p.cmd.Process.Signal(sig); err != nil {
		return fmt.Errorf("failed to send signal: %w", err)
	}

	return nil
}

// StdoutLine returns a channel that receives stdout lines.
func (p *Process) StdoutLine() <-chan string {
	ch := make(chan string)

	go func() {
		defer close(ch)

		scanner := bufio.NewScanner(p.stdout)
		for scanner.Scan() {
			ch <- scanner.Text()
		}

		if err := scanner.Err(); err != nil {
			p.logger.Error("error reading stdout", logger.StringField("error", err.Error()))
		}
	}()

	return ch
}

// StderrLine returns a channel that receives stderr lines.
func (p *Process) StderrLine() <-chan string {
	ch := make(chan string)

	go func() {
		defer close(ch)

		scanner := bufio.NewScanner(p.stderr)
		for scanner.Scan() {
			line := scanner.Text()
			p.logger.Debug("stderr", logger.StringField("line", line))
			ch <- line
		}

		if err := scanner.Err(); err != nil {
			p.logger.Error("error reading stderr", logger.StringField("error", err.Error()))
		}
	}()

	return ch
}

// Pid returns the process ID.
func (p *Process) Pid() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd.Process == nil {
		return -1
	}
	return p.cmd.Process.Pid
}

// IsDone returns true if the process has completed.
func (p *Process) IsDone() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.done
}

// WaitWithTimeout waits for the process with a timeout.
func (p *Process) WaitWithTimeout(timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- p.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		p.Kill()
		return fmt.Errorf("process timeout after %v", timeout)
	}
}

// CombinedOutput returns the combined stdout and stderr output.
func (p *Process) CombinedOutput() ([]byte, error) {
	if err := p.Start(); err != nil {
		return nil, err
	}
	defer p.Wait()

	var stdout, stderr []byte

	// Read stdout
	stdoutCh := make(chan []byte)
	go func() {
		data, _ := io.ReadAll(p.stdout)
		stdoutCh <- data
	}()

	// Read stderr
	stderrCh := make(chan []byte)
	go func() {
		data, _ := io.ReadAll(p.stderr)
		stderrCh <- data
	}()

	// Wait for both
	stdout = <-stdoutCh
	stderr = <-stderrCh

	output := append(stdout, stderr...)
	return output, nil
}
