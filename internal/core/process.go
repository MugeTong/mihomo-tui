package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	startupTimeout = 10 * time.Second
	stopTimeout    = 5 * time.Second
)

type ProcessOptions struct {
	BinaryPath        string
	ConfigPath        string
	DataDir           string
	PIDPath           string
	LogPath           string
	ControllerAddress string
}

type ProcessManager struct {
	mu     sync.Mutex
	opts   ProcessOptions
	status Status
	cmd    *exec.Cmd
}

func NewProcessManager(opts ProcessOptions) *ProcessManager {
	return &ProcessManager{opts: opts, status: StatusStopped}
}

func (m *ProcessManager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status == StatusStarting || m.status == StatusStopping {
		return m.status
	}
	pid, err := readPID(m.opts.PIDPath)
	if err == nil && m.managedProcessAlive(pid) {
		m.status = StatusRunning
		return m.status
	}
	if err == nil {
		_ = os.Remove(m.opts.PIDPath)
	}
	if (err == nil || errors.Is(err, os.ErrNotExist)) && m.status != StatusFailed {
		m.status = StatusStopped
	}
	return m.status
}

func (m *ProcessManager) Start(ctx context.Context) error {
	if m.Status() == StatusRunning {
		return nil
	}

	m.mu.Lock()
	if m.status == StatusStarting {
		m.mu.Unlock()
		return fmt.Errorf("mihomo is already starting")
	}
	m.status = StatusStarting
	m.mu.Unlock()

	if err := m.start(ctx); err != nil {
		m.setFailed()
		return err
	}
	return nil
}

func (m *ProcessManager) start(ctx context.Context) error {
	if strings.TrimSpace(m.opts.BinaryPath) == "" {
		return fmt.Errorf("mihomo binary path is empty")
	}
	if strings.TrimSpace(m.opts.ConfigPath) == "" {
		return fmt.Errorf("mihomo config path is empty")
	}
	if err := os.MkdirAll(m.opts.DataDir, 0o700); err != nil {
		return fmt.Errorf("create mihomo data directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(m.opts.PIDPath), 0o700); err != nil {
		return fmt.Errorf("create mihomo PID directory: %w", err)
	}

	logFile, err := os.OpenFile(m.opts.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open mihomo log: %w", err)
	}
	cmd := exec.Command(m.opts.BinaryPath, "-d", m.opts.DataDir, "-f", m.opts.ConfigPath)
	// Keep the managed core alive when the TUI (and potentially its terminal
	// session) exits after q. The PID file lets a later TUI instance take over.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("start mihomo: %w", err)
	}
	logFile.Close()

	if err := writePID(m.opts.PIDPath, cmd.Process.Pid); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}

	m.mu.Lock()
	m.cmd = cmd
	m.mu.Unlock()
	go m.wait(cmd)

	readyCtx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()
	if err := waitForController(readyCtx, m.opts.ControllerAddress, cmd.Process.Pid); err != nil {
		_ = m.stopPID(cmd.Process.Pid)
		return fmt.Errorf("wait for mihomo controller: %w; see %s", err, m.opts.LogPath)
	}

	m.mu.Lock()
	if m.cmd == cmd {
		m.status = StatusRunning
	}
	m.mu.Unlock()
	return nil
}

func (m *ProcessManager) Stop() error {
	m.mu.Lock()
	if m.status == StatusStopping {
		m.mu.Unlock()
		return nil
	}
	m.status = StatusStopping
	m.mu.Unlock()

	pid, err := readPID(m.opts.PIDPath)
	if errors.Is(err, os.ErrNotExist) {
		m.setStopped()
		return nil
	}
	if err != nil {
		m.setFailed()
		return err
	}
	if !m.managedProcessAlive(pid) {
		_ = os.Remove(m.opts.PIDPath)
		m.setStopped()
		return nil
	}
	if err := m.stopPID(pid); err != nil {
		m.setFailed()
		return err
	}
	_ = os.Remove(m.opts.PIDPath)
	m.setStopped()
	return nil
}

func (m *ProcessManager) Restart(ctx context.Context) error {
	if err := m.Stop(); err != nil {
		return err
	}
	return m.Start(ctx)
}

func (m *ProcessManager) stopPID(pid int) error {
	if !m.managedProcessAlive(pid) {
		return nil
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find mihomo process: %w", err)
	}
	if err := process.Signal(os.Interrupt); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("interrupt mihomo: %w", err)
	}
	deadline := time.Now().Add(stopTimeout)
	for m.managedProcessAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if !m.managedProcessAlive(pid) {
		return nil
	}
	if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("kill mihomo after timeout: %w", err)
	}
	return nil
}

func (m *ProcessManager) managedProcessAlive(pid int) bool {
	if !processAlive(pid) {
		return false
	}
	command, err := processCommand(pid)
	if err != nil {
		return false
	}
	binaryPath := strings.TrimSpace(m.opts.BinaryPath)
	configPath := strings.TrimSpace(m.opts.ConfigPath)
	return binaryPath != "" && configPath != "" &&
		strings.Contains(command, binaryPath) && strings.Contains(command, configPath)
}

func processCommand(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("invalid PID %d", pid)
	}
	output, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	if err != nil {
		return "", fmt.Errorf("inspect process %d: %w", pid, err)
	}
	command := strings.TrimSpace(string(output))
	if command == "" {
		return "", fmt.Errorf("process %d has no command", pid)
	}
	return command, nil
}

func (m *ProcessManager) wait(cmd *exec.Cmd) {
	_ = cmd.Wait()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd != cmd {
		return
	}
	m.cmd = nil
	if pid, err := readPID(m.opts.PIDPath); err == nil && pid == cmd.Process.Pid {
		_ = os.Remove(m.opts.PIDPath)
	}
	if m.status != StatusStopping {
		m.status = StatusStopped
	}
}

func (m *ProcessManager) setFailed() {
	m.mu.Lock()
	m.status = StatusFailed
	m.mu.Unlock()
}

func (m *ProcessManager) setStopped() {
	m.mu.Lock()
	m.status = StatusStopped
	m.cmd = nil
	m.mu.Unlock()
}

func waitForController(ctx context.Context, address string, pid int) error {
	if strings.TrimSpace(address) == "" {
		return nil
	}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		connection, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err == nil {
			connection.Close()
			return nil
		}
		if !processAlive(pid) {
			return fmt.Errorf("mihomo exited before the controller became ready")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func readPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("parse mihomo PID file %s", path)
	}
	return pid, nil
}

func writePID(path string, pid int) error {
	if err := os.WriteFile(path, []byte(strconv.Itoa(pid)+"\n"), 0o600); err != nil {
		return fmt.Errorf("write mihomo PID file: %w", err)
	}
	return nil
}
