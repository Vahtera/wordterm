package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// CreateWorkingCopy creates a new temporary project mirror for the active session.
func CreateWorkingCopy(projectRoot string) (string, error) {
	base := filepath.Join(os.TempDir(), "wordterm")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	copyRoot := filepath.Join(base, fmt.Sprintf("%s-%d", filepath.Base(projectRoot), os.Getpid()))
	if err := os.MkdirAll(copyRoot, 0o755); err != nil {
		return "", err
	}
	return copyRoot, nil
}

// EnsureDirectory creates a directory path when needed.
func EnsureDirectory(path string) error {
	return os.MkdirAll(path, 0o755)
}
