package project

import "testing"

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
