package project

import (
	"os"
	"path/filepath"
	"testing"
	"wordterm/internal/storage"
)

func TestProjectMetadataDefaults(t *testing.T) {
	p := NewProject("Example", "Author")
	if p.Name != "Example" {
		t.Fatalf("project name mismatch: got %q", p.Name)
	}
	if p.Author != "Author" {
		t.Fatalf("author mismatch: got %q", p.Author)
	}
	if len(p.Chapters) == 0 {
		t.Fatal("project should contain one default chapter")
	}
	if len(p.Chapters[0].Scenes) == 0 {
		t.Fatal("default chapter should contain one default scene")
	}
}

func TestWordCountUsesWhitespaceSplit(t *testing.T) {
	text := "Hello world\nThis is  a test"
	count := CountWords(text)
	if count != 6 {
		t.Fatalf("expected 6 words, got %d", count)
	}
}

func TestLoadProjectFromDirectory(t *testing.T) {
	dir := t.TempDir()
	chapterDir := filepath.Join(dir, "Chapter 1")
	if err := os.MkdirAll(chapterDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chapterDir, "Scene 1.txt"), []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	meta := `{"name":"Example","author":"Author","format_version":1,"last_opened_scene":{"chapter":"Chapter 1","scene":"Scene 1"},"chapters":[{"name":"Chapter 1","scenes":["Scene 1"]}]}`
	if err := os.WriteFile(filepath.Join(dir, MetadataFileName), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := LoadProject(dir)
	if err != nil {
		t.Fatalf("load project should succeed: %v", err)
	}
	if p.Name != "Example" {
		t.Fatalf("loaded name mismatch: got %q", p.Name)
	}
	if p.LastOpenedScene.Scene != "Scene 1" {
		t.Fatalf("last opened scene mismatch: got %q", p.LastOpenedScene.Scene)
	}
}

func TestCreateWorkingCopyCopiesProjectFiles(t *testing.T) {
	dir := t.TempDir()
	chapterDir := filepath.Join(dir, "Chapter 1")
	if err := os.MkdirAll(chapterDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chapterDir, "Scene 1.txt"), []byte("test content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, MetadataFileName), []byte(`{"name":"Example","author":"Author","format_version":1,"last_opened_scene":{"chapter":"Chapter 1","scene":"Scene 1"},"chapters":[{"name":"Chapter 1","scenes":["Scene 1"]}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	copyRoot, err := storage.CreateWorkingCopy(dir)
	if err != nil {
		t.Fatalf("failed to create working copy: %v", err)
	}
	if copyRoot == dir {
		t.Fatal("working copy should not be the same as original project root")
	}
	if _, err := os.Stat(filepath.Join(copyRoot, MetadataFileName)); err != nil {
		t.Fatalf("working copy should include metadata: %v", err)
	}
}
