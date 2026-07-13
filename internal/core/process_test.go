package core

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestProcessManagerPersistsOwnershipAcrossManagers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mihomo-tui only manages Unix processes")
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "fake-mihomo")
	script := "#!/bin/sh\ntrap 'exit 0' INT TERM\nwhile :; do sleep 1; done\n"
	if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}

	opts := ProcessOptions{
		BinaryPath:        binary,
		ConfigPath:        filepath.Join(dir, "config.yaml"),
		DataDir:           filepath.Join(dir, "data"),
		PIDPath:           filepath.Join(dir, "mihomo.pid"),
		LogPath:           filepath.Join(dir, "mihomo.log"),
		ControllerAddress: "",
	}

	first := NewProcessManager(opts)
	if err := first.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = first.Stop() })
	if got := first.Status(); got != StatusRunning {
		command, commandErr := processCommand(first.cmd.Process.Pid)
		t.Fatalf("first Status() = %s, want %s; command=%q err=%v", got, StatusRunning, command, commandErr)
	}

	second := NewProcessManager(opts)
	if got := second.Status(); got != StatusRunning {
		t.Fatalf("second Status() = %s, want %s", got, StatusRunning)
	}
	if err := second.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if got := second.Status(); got != StatusStopped {
		t.Fatalf("Status() after Stop = %s, want %s", got, StatusStopped)
	}
	if _, err := os.Stat(opts.PIDPath); !os.IsNotExist(err) {
		t.Fatalf("PID file still exists after Stop: %v", err)
	}
}

func TestProcessManagerReportsStartFailure(t *testing.T) {
	dir := t.TempDir()
	manager := NewProcessManager(ProcessOptions{
		BinaryPath:        filepath.Join(dir, "missing-mihomo"),
		ConfigPath:        filepath.Join(dir, "config.yaml"),
		DataDir:           filepath.Join(dir, "data"),
		PIDPath:           filepath.Join(dir, "mihomo.pid"),
		LogPath:           filepath.Join(dir, "mihomo.log"),
		ControllerAddress: "127.0.0.1:1",
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Start(ctx); err == nil {
		t.Fatal("Start() succeeded with a missing binary")
	}
	if got := manager.Status(); got != StatusFailed {
		t.Fatalf("Status() = %s, want %s", got, StatusFailed)
	}
}

func TestProcessManagerRejectsReusedPID(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "mihomo.pid")
	if err := writePID(pidPath, os.Getpid()); err != nil {
		t.Fatal(err)
	}
	manager := NewProcessManager(ProcessOptions{
		BinaryPath: "/not/the/test/process/mihomo",
		ConfigPath: filepath.Join(dir, "config.yaml"),
		PIDPath:    pidPath,
	})
	if got := manager.Status(); got != StatusStopped {
		t.Fatalf("Status() = %s, want %s for reused PID", got, StatusStopped)
	}
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Fatalf("stale PID file was not removed: %v", err)
	}
}
