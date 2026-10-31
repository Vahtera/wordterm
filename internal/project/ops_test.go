package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStructureOperations(t *testing.T) {
	dir, err := Create(t.TempDir(), "N", "a", "")
	if err != nil {
		t.Fatal(err)
	}
	p, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(p.AddScene("Chapter1", -1, "Scene2"))
	must(p.AddChapter("Chapter2"))
	if p.AddScene("Chapter1", -1, "scene2") == nil {
		t.Fatal("case-insensitive duplicate accepted")
	}
	ci, si, err := p.MoveScene(0, 1, 1) // last scene down -> first of next chapter
	must(err)
	if ci != 1 || si != 0 {
		t.Fatalf("got %d,%d", ci, si)
	}
	if _, err := os.Stat(filepath.Join(dir, "Chapter2", "Scene2.txt")); err != nil {
		t.Fatal("file was not moved:", err)
	}
	ci, si, err = p.MoveScene(1, 0, -1) // first scene up -> last of previous chapter
	must(err)
	if ci != 0 || si != 1 {
		t.Fatalf("got %d,%d", ci, si)
	}
	_, _, err = p.MoveScene(0, 1, 1)
	must(err)
	must(p.RenameChapter("Chapter2", "Part2"))
	must(p.RenameScene("Part2", "Scene2", "Opening"))
	if _, err := os.Stat(filepath.Join(dir, "Part2", "Opening.txt")); err != nil {
		t.Fatal(err)
	}
	if p.DeleteChapter("Part2") == nil {
		t.Fatal("non-empty chapter was deleted")
	}
	must(p.DeleteScene("Part2", "Opening"))
	must(p.DeleteChapter("Part2"))
	q, err := Load(dir) // metadata on disk must match the filesystem
	must(err)
	if len(q.Meta.Chapters) != 1 || len(q.Meta.Chapters[0].Scenes) != 1 {
		t.Fatalf("unexpected structure: %+v", q.Meta.Chapters)
	}
}
