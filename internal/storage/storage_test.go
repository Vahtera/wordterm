package storage_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Vahtera/wordterm/internal/project"
	"github.com/Vahtera/wordterm/internal/storage"
)

func setup(t *testing.T) (orig string, s *storage.Session) {
	t.Helper()
	tmp := t.TempDir()
	for _, k := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(k, tmp)
	}
	dir, err := project.Create(t.TempDir(), "Novel", "Me", "")
	if err != nil {
		t.Fatal(err)
	}
	s, err = storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return dir, s
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestOriginalUntouchedUntilSave(t *testing.T) {
	orig, s := setup(t)
	scene := filepath.Join("Chapter1", "Scene1.txt")
	if err := os.WriteFile(filepath.Join(s.Root, scene), []byte("hello wörld"), 0o644); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(orig, scene)) != "" {
		t.Fatal("original changed before Save")
	}
	if has, _ := s.HasChanges(); !has {
		t.Fatal("HasChanges should be true")
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(orig, scene)) != "hello wörld" {
		t.Fatal("Save did not update original")
	}
	if has, _ := s.HasChanges(); has {
		t.Fatal("HasChanges should be false after Save")
	}
}

func TestSaveRemovesDeletedSceneAndReloadDiscards(t *testing.T) {
	orig, s := setup(t)
	os.Remove(filepath.Join(s.Root, "Chapter1", "Scene1.txt"))
	os.WriteFile(filepath.Join(s.Root, "Chapter1", "Other.txt"), []byte("x"), 0o644)
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(orig, "Chapter1", "Scene1.txt")); !os.IsNotExist(err) {
		t.Fatal("deleted scene still in original")
	}
	os.WriteFile(filepath.Join(s.Root, "Chapter1", "Other.txt"), []byte("changed"), 0o644)
	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(s.Root, "Chapter1", "Other.txt")) != "x" {
		t.Fatal("Reload did not restore original content")
	}
}

func TestRecovery(t *testing.T) {
	_, s := setup(t)
	if recs, _ := storage.FindRecoverable(); len(recs) != 0 {
		t.Fatal("live session must not be recoverable")
	}
	data, _ := json.Marshal(map[string]any{"original": s.Original, "pid": 999999999, "started": time.Now()})
	os.WriteFile(filepath.Join(s.Dir, "session.json"), data, 0o600)
	recs, err := storage.FindRecoverable()
	if err != nil || len(recs) != 1 {
		t.Fatalf("recs=%v err=%v", recs, err)
	}
	if _, err := storage.Adopt(recs[0]); err != nil {
		t.Fatal(err)
	}
	if recs, _ := storage.FindRecoverable(); len(recs) != 0 {
		t.Fatal("adopted session must not be recoverable")
	}
}

func TestOutsideChangeIsNotOverwrittenSilently(t *testing.T) {
	orig, s := setup(t)
	scene := filepath.Join("Chapter1", "Scene1.txt")
	os.WriteFile(filepath.Join(orig, scene), []byte("edited elsewhere"), 0o644)
	os.WriteFile(filepath.Join(s.Root, scene), []byte("my words"), 0o644)
	if err := s.Save(); !errors.Is(err, storage.ErrOriginalChanged) {
		t.Fatalf("Save err = %v, want ErrOriginalChanged", err)
	}
	if read(t, filepath.Join(orig, scene)) != "edited elsewhere" {
		t.Fatal("original was overwritten without confirmation")
	}
	if err := s.ForceSave(); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(orig, scene)) != "my words" {
		t.Fatal("ForceSave did not write")
	}
	if err := s.Save(); err != nil {
		t.Fatalf("after ForceSave a plain Save must work: %v", err)
	}
}
