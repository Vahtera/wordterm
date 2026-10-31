package editor

import (
	"reflect"
	"testing"
)

func TestInsertAndText(t *testing.T) {
	e := New("")
	e.Insert("héllo")
	e.Insert("\nwörld")
	if e.Text() != "héllo\nwörld" || e.cur != (pos{1, 5}) {
		t.Fatalf("text=%q cur=%v", e.Text(), e.cur)
	}
}

func TestBackspaceJoinsLines(t *testing.T) {
	e := New("ab\ncd")
	e.cur = pos{1, 0}
	e.Backspace()
	if e.Text() != "abcd" || e.cur != (pos{0, 2}) {
		t.Fatalf("text=%q cur=%v", e.Text(), e.cur)
	}
	e.Undo()
	if e.Text() != "ab\ncd" || e.cur != (pos{1, 0}) {
		t.Fatalf("after undo text=%q cur=%v", e.Text(), e.cur)
	}
}

func TestUndoRedo(t *testing.T) {
	e := New("")
	e.Insert("hello world")
	e.Backspace()
	e.Undo()
	if e.Text() != "hello world" {
		t.Fatalf("got %q", e.Text())
	}
	e.Undo()
	if e.Text() != "" {
		t.Fatalf("got %q", e.Text())
	}
	e.Redo()
	if e.Text() != "hello world" {
		t.Fatalf("got %q", e.Text())
	}
}

func TestTypingUndoesByWord(t *testing.T) {
	e := New("")
	for _, r := range "ab cd" {
		e.Insert(string(r))
	}
	e.Undo()
	if e.Text() != "ab " {
		t.Fatalf("got %q", e.Text())
	}
}

func TestWrap(t *testing.T) {
	got := wrapLine(clusters("hello world foo"), 8)
	if !reflect.DeepEqual(got, []int{0, 6, 12}) {
		t.Fatalf("got %v", got)
	}
	if got := wrapLine(clusters("日日日"), 4); !reflect.DeepEqual(got, []int{0, 2}) {
		t.Fatalf("wide chars: got %v", got)
	}
}

func TestClusters(t *testing.T) {
	if n := len(clusters("e\u0301x")); n != 2 {
		t.Fatalf("got %d clusters, want 2", n)
	}
}

func TestVerticalMoveOverWrap(t *testing.T) {
	e := New("hello world foo")
	e.SetSize(8, 5)
	e.moveVert(1)
	if e.cur.c != 6 {
		t.Fatalf("got %d", e.cur.c)
	}
	e.moveVert(1)
	if e.cur.c != 12 {
		t.Fatalf("got %d", e.cur.c)
	}
	e.moveVert(-1)
	if e.cur.c != 6 {
		t.Fatalf("got %d", e.cur.c)
	}
}

func TestCentered(t *testing.T) {
	e := New("a\nb\nc")
	e.SetSize(10, 5)
	e.cur = pos{1, 0}
	got := e.Centered(false)
	want := []string{"", "a", "b", "c", ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}
