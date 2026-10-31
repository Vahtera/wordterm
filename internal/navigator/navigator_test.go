package navigator

import (
	"testing"

	"wordterm/internal/project"
)

func TestKeyboardTree(t *testing.T) {
	m := &project.Meta{Name: "P", Chapters: []project.Chapter{{Name: "A", Scenes: []project.Scene{{Name: "a1", File: "a1"}, {Name: "a2", File: "a2"}}}, {Name: "B", Scenes: []project.Scene{{Name: "b1", File: "b1"}}}}}
	n := New()
	if got := len(n.items(m)); got != 6 {
		t.Fatalf("items = %d", got)
	}
	n.Down(m) // A
	n.Left(m) // collapse A
	if got := len(n.items(m)); got != 4 {
		t.Fatalf("collapsed items = %d", got)
	}
	n.Right(m) // expand
	n.Right(m) // first child
	if n.sel != (Item{Scene, "A", "a1"}) {
		t.Fatalf("sel = %+v", n.sel)
	}
	n.Left(m) // parent
	if n.sel != (Item{Kind: Chapter, Chapter: "A"}) {
		t.Fatalf("sel = %+v", n.sel)
	}
	n.Up(m)
	if n.sel.Kind != Root {
		t.Fatalf("sel = %+v", n.sel)
	}
}

func TestSelectionSurvivesDeletion(t *testing.T) {
	m := &project.Meta{Name: "P", Chapters: []project.Chapter{{Name: "A", Scenes: []project.Scene{{Name: "a1", File: "a1"}, {Name: "a2", File: "a2"}}}}}
	n := New()
	n.Select(Item{Scene, "A", "a2"})
	n.Selected(m)
	m.Chapters[0].Scenes = []project.Scene{{Name: "a1", File: "a1"}}
	if it := n.Selected(m); it != (Item{Scene, "A", "a1"}) {
		t.Fatalf("sel = %+v", it)
	}
}
