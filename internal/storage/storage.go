package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CreateWorkingCopy creates a temporary project mirror for the active session.
func CreateWorkingCopy(projectRoot string) (string, error) {
	if projectRoot == "" {
		return "", fmt.Errorf("project root is required")
	}
	base := filepath.Join(os.TempDir(), "wordterm")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	copyRoot, err := os.MkdirTemp(base, filepath.Base(projectRoot)+"-")
	if err != nil {
		return "", err
	}
	if err := copyDirectory(projectRoot, copyRoot); err != nil {
		_ = os.RemoveAll(copyRoot)
		return "", err
	}
	return copyRoot, nil
}

func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer out.Close()
		if _, err := io.Copy(out, in); err != nil {
			return err
		}
		return nil
	})
}

// SyncWorkingCopyToOriginal copies the working copy back to the original project directory.
func SyncWorkingCopyToOriginal(workingCopyRoot, originalRoot string) error {
	if workingCopyRoot == "" || originalRoot == "" {
		return fmt.Errorf("working copy and original project root are required")
	}
	if err := os.MkdirAll(originalRoot, 0o755); err != nil {
		return err
	}
	return copyDirectory(workingCopyRoot, originalRoot)
}

// EnsureDirectory creates a directory path when needed.
func EnsureDirectory(path string) error {
	return os.MkdirAll(path, 0o755)
}
