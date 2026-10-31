// Package storage holds filesystem helpers and the temporary working copy.
// It knows nothing about the project format.
package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WriteFileAtomic writes data to a temp file in the same directory, syncs
// it and renames it over path, so readers never see a half-written file.
func WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".wordterm-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	fail := func(err error) error {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return fail(err)
	}
	if err := tmp.Sync(); err != nil {
		return fail(err)
	}
	if err := tmp.Close(); err != nil {
		return fail(err)
	}
	if err := os.Chmod(name, perm); err != nil {
		return fail(err)
	}
	if err := os.Rename(name, path); err != nil {
		return fail(err)
	}
	return nil
}

// CopyTree copies directories and regular files from src into dst.
func CopyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case d.Type().IsRegular():
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			return os.WriteFile(target, data, 0o644)
		default:
			return fmt.Errorf("unsupported file type: %s", p)
		}
	})
}

func isTemp(name string) bool {
	return strings.HasPrefix(name, ".wordterm-") && strings.HasSuffix(name, ".tmp")
}

// listTree returns slash-separated relative paths of files and directories.
func listTree(root string) (files, dirs []string, err error) {
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil || rel == "." {
			return err
		}
		rel = filepath.ToSlash(rel)
		switch {
		case d.IsDir():
			dirs = append(dirs, rel)
		case !isTemp(d.Name()):
			files = append(files, rel)
		}
		return nil
	})
	return
}

// SyncTree makes orig match work. It only reads work. New and changed
// files are written first (atomically); afterwards stale .txt files and
// the directories left empty by that are removed. Other files that
// happen to live in orig are never deleted.
func SyncTree(work, orig string) error {
	files, dirs, err := listTree(work)
	if err != nil {
		return err
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(orig, filepath.FromSlash(d)), 0o755); err != nil {
			return err
		}
	}
	keep := map[string]bool{}
	for _, f := range files {
		keep[f] = true
		data, err := os.ReadFile(filepath.Join(work, filepath.FromSlash(f)))
		if err != nil {
			return err
		}
		dst := filepath.Join(orig, filepath.FromSlash(f))
		if cur, err := os.ReadFile(dst); err == nil && bytes.Equal(cur, data) {
			continue
		}
		if err := WriteFileAtomic(dst, data, 0o644); err != nil {
			return fmt.Errorf("saving %s: %w", f, err)
		}
	}
	ofiles, odirs, err := listTree(orig)
	if err != nil {
		return err
	}
	for _, f := range ofiles {
		if !keep[f] && strings.HasSuffix(f, ".txt") {
			if err := os.Remove(filepath.Join(orig, filepath.FromSlash(f))); err != nil {
				return err
			}
		}
	}
	want := map[string]bool{}
	for _, d := range dirs {
		want[d] = true
	}
	sort.Slice(odirs, func(i, j int) bool { return len(odirs[i]) > len(odirs[j]) })
	for _, d := range odirs {
		if !want[d] {
			os.Remove(filepath.Join(orig, filepath.FromSlash(d))) // fails harmlessly if not empty
		}
	}
	return nil
}

func snapshot(root string) (map[string]string, error) {
	files, dirs, err := listTree(root)
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	for _, d := range dirs {
		m[d+"/"] = ""
	}
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		m[f] = hex.EncodeToString(sum[:])
	}
	return m, nil
}

// TreesEqual reports whether two directory trees have identical structure and contents.
func TreesEqual(a, b string) (bool, error) {
	sa, err := snapshot(a)
	if err != nil {
		return false, err
	}
	sb, err := snapshot(b)
	if err != nil {
		return false, err
	}
	if len(sa) != len(sb) {
		return false, nil
	}
	for k, v := range sa {
		if w, ok := sb[k]; !ok || w != v {
			return false, nil
		}
	}
	return true, nil
}
