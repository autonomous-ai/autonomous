package system

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
)

// LookPath searches for the named executable in the PATH. It returns the full path
// or an error if not found.
func LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// FileExists reports whether path exists (file or directory).
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// MkdirAll creates the directory and any parents with the given permission.
func MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// WriteFile writes data to path with the given permission.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// CreateTempFile creates a new temporary file in dir with the given pattern.
// It returns the path and a cleanup function that removes the file. The caller
// must close the file if it was opened.
func CreateTempFile(dir, pattern string) (path string, cleanup func(), err error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", nil, err
	}
	path = f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", nil, err
	}
	cleanup = func() { _ = os.Remove(path) }
	return path, cleanup, nil
}

// CreateTempDir creates a new temporary directory in dir with the given pattern.
// It returns the path and a cleanup function that removes the directory and its contents.
func CreateTempDir(dir, pattern string) (path string, cleanup func(), err error) {
	path, err = os.MkdirTemp(dir, pattern)
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { _ = os.RemoveAll(path) }
	return path, cleanup, nil
}

// DownloadToTemp downloads url via client to a temporary file and returns its path
// and a cleanup function. The client's context is not used; use the provided ctx
// for the request.
func DownloadToTemp(ctx context.Context, client *http.Client, url string) (path string, cleanup func(), err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("download %s: status %s", url, resp.Status)
	}
	f, err := os.CreateTemp("", "download-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp file: %w", err)
	}
	path = f.Name()
	cleanup = func() { _ = os.Remove(path) }
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		cleanup()
		return "", nil, fmt.Errorf("write temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("close temp file: %w", err)
	}
	return path, cleanup, nil
}
