package project

import (
	"encoding/json"
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
	if _, err := p.AddScene("Chapter1", -1, "Scene2"); err != nil {
		t.Fatal(err)
	}
	must(p.AddChapter("Chapter2"))
	if _, err := p.AddScene("Chapter1", -1, "scene2"); err == nil {
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
	if _, err := p.RenameScene("Part2", "Scene2", "Opening"); err != nil {
		t.Fatal(err)
	}
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

func TestMoveWithNameClash(t *testing.T) {
	dir, _ := Create(t.TempDir(), "N", "a", "")
	p, _ := Load(dir)
	if err := p.AddChapter("Chapter2"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.AddScene("Chapter2", -1, "Scene1"); err != nil {
		t.Fatal(err)
	}
	ci, si, err := p.MoveScene(0, 0, 1) // Scene1 -> Chapter2, which already has a Scene1
	if err != nil || ci != 1 || si != 0 {
		t.Fatalf("got %d,%d,%v", ci, si, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Chapter2", "Scene1_2.txt")); err != nil {
		t.Fatal("clashing file should be stored as Scene1_2.txt:", err)
	}
	q, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := q.Meta.Chapters[1].Scenes[0]; got != (Scene{"Scene1", "Scene1_2"}) {
		t.Fatalf("got %+v", got)
	}
	if q.Meta.Chapters[1].Scenes[1] != (Scene{"Scene1", "Scene1"}) {
		t.Fatalf("original scene changed: %+v", q.Meta.Chapters[1].Scenes)
	}
}

func TestSceneJSONForms(t *testing.T) {
	var c Chapter
	if err := json.Unmarshal([]byte(`{"name":"C","scenes":["a",{"name":"b","file":"b_2"}]}`), &c); err != nil {
		t.Fatal(err)
	}
	if c.Scenes[0] != (Scene{"a", "a"}) || c.Scenes[1] != (Scene{"b", "b_2"}) {
		t.Fatalf("%+v", c.Scenes)
	}
	out, _ := json.Marshal(c)
	if string(out) != `{"name":"C","scenes":["a",{"name":"b","file":"b_2"}]}` {
		t.Fatalf("%s", out)
	}
}
