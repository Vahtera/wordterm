package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Session is one editing session: a temporary working copy of Original.
// Root is the working copy and the only tree the app may touch.
type Session struct {
	Original string
	Dir      string // session directory under the OS temp dir
	Root     string // Dir/project
	started  time.Time
	ephemeral bool
	baseline  map[string]string // the original as we last saw it (nil = unknown, e.g. recovered)
}

// ErrOriginalChanged means the original project differs from what this
// session last read or wrote, so a plain Save could overwrite outside changes.
var ErrOriginalChanged = errors.New("the original project changed since this working copy was made")

type info struct {
	Original string    `json:"original"`
	PID      int       `json:"pid"`
	Started  time.Time `json:"started"`
}

const infoFile = "session.json"

func baseDir() string { return filepath.Join(os.TempDir(), "wordterm") }

// Open creates a working copy of the project in original. original is only read.
func Open(original string) (*Session, error) {
	st, err := os.Stat(original)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", original)
	}
	abs, err := filepath.Abs(original)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(baseDir(), 0o700); err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(baseDir(), "session-*")
	if err != nil {
		return nil, err
	}
	s := &Session{Original: abs, Dir: dir, Root: filepath.Join(dir, "project"), started: time.Now()}
	if err := CopyTree(abs, s.Root); err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("creating working copy: %w", err)
	}
	snap, err := snapshot(abs)
	if err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("reading original: %w", err)
	}
	s.baseline = snap
	if err := s.writeInfo(); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	return s, nil
}

// Ephemeral marks the session as never worth recovering (used for the built-in help project).
func (s *Session) Ephemeral() {
	s.ephemeral = true
	os.Remove(filepath.Join(s.Dir, infoFile))
}

func (s *Session) writeInfo() error {
	if s.ephemeral {
		return nil
	}
	data, err := json.MarshalIndent(info{s.Original, os.Getpid(), s.started}, "", "  ")
	if err != nil {
		return err
	}
	return WriteFileAtomic(filepath.Join(s.Dir, infoFile), data, 0o600)
}

// SetOriginal points the session at a new project directory (after Save As).
func (s *Session) SetOriginal(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	s.Original = abs
	if snap, err := snapshot(abs); err == nil {
		s.baseline = snap
	}
	return s.writeInfo()
}

// Save synchronises the working copy into the original project.
// On failure the working copy is untouched.
// It refuses with ErrOriginalChanged if the original was modified outside
// Wordterm; use ForceSave after the user confirms.
func (s *Session) Save() error {
	cur, err := snapshot(s.Original)
	if err != nil {
		return err
	}
	if s.baseline == nil || !equalSnap(cur, s.baseline) {
		return ErrOriginalChanged
	}
	return s.ForceSave()
}

// ForceSave synchronises without checking for outside changes.
func (s *Session) ForceSave() error {
	if err := SyncTree(s.Root, s.Original); err != nil {
		return err
	}
	snap, err := snapshot(s.Original)
	if err != nil {
		return err
	}
	s.baseline = snap
	return nil
}

// HasChanges reports whether the working copy differs from the original.
func (s *Session) HasChanges() (bool, error) {
	eq, err := TreesEqual(s.Root, s.Original)
	return !eq, err
}

// Reload replaces the working copy with a fresh copy of the original.
// The old working copy is only removed once the new one is in place.
func (s *Session) Reload() error {
	fresh, old := s.Root+".new", s.Root+".old"
	os.RemoveAll(fresh)
	os.RemoveAll(old)
	if err := CopyTree(s.Original, fresh); err != nil {
		os.RemoveAll(fresh)
		return err
	}
	if err := os.Rename(s.Root, old); err != nil {
		os.RemoveAll(fresh)
		return err
	}
	if err := os.Rename(fresh, s.Root); err != nil {
		os.Rename(old, s.Root)
		return err
	}
	os.RemoveAll(old)
	if snap, err := snapshot(s.Original); err == nil {
		s.baseline = snap
	}
	return nil
}

// Close deletes the working copy. Call it only on a normal exit when the
// copy is no longer needed.
func (s *Session) Close() error { return os.RemoveAll(s.Dir) }

// Recoverable is a working copy left behind by a process that no longer runs.
type Recoverable struct {
	Dir, Root, Original string
	Started             time.Time
}

// FindRecoverable lists working copies whose owning process is gone.
func FindRecoverable() ([]Recoverable, error) {
	entries, err := os.ReadDir(baseDir())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Recoverable
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(baseDir(), e.Name())
		data, err := os.ReadFile(filepath.Join(dir, infoFile))
		if err != nil {
			continue
		}
		var in info
		if json.Unmarshal(data, &in) != nil || pidAlive(in.PID) {
			continue
		}
		out = append(out, Recoverable{dir, filepath.Join(dir, "project"), in.Original, in.Started})
	}
	return out, nil
}

// Adopt takes over a recoverable working copy for the current process.
func Adopt(r Recoverable) (*Session, error) {
	s := &Session{Original: r.Original, Dir: r.Dir, Root: r.Root, started: r.Started}
	return s, s.writeInfo()
}

// Discard permanently deletes a recoverable working copy.
func Discard(r Recoverable) error { return os.RemoveAll(r.Dir) }
