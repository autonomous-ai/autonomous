// Package system provides OS and system-level utilities: process execution,
// file and directory helpers, and temporary file/dir creation with cleanup.
package system

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// Run runs the named program with the given arguments and context.
// It returns combined stdout and stderr. If the context is cancelled or times out,
// the process is killed.
func Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s %v: %w", name, args, err)
	}
	return out, nil
}

// ChmodRecursive walks root and sets directory permissions to dirMode and file
// permissions to fileMode. Symlinks are not followed.
func ChmodRecursive(root string, dirMode, fileMode os.FileMode) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return os.Chmod(path, dirMode)
		}
		return os.Chmod(path, fileMode)
	})
}

// SpawnBackground starts a process that is fully detached from the parent.
// The process survives if the caller exits. Stdout/stderr are discarded.
func SpawnBackground(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawn %s %v: %w", name, args, err)
	}
	// Release so the caller doesn't wait on it.
	_ = cmd.Process.Release()
	return nil
}

// RestartService runs systemctl restart for the given service name.
func RestartService(ctx context.Context, service string) error {
	_, err := Run(ctx, "systemctl", "restart", service)
	return err
}

// Poweroff runs systemctl poweroff. Requires appropriate privileges.
func Poweroff(ctx context.Context) error {
	_, err := Run(ctx, "systemctl", "poweroff")
	return err
}
