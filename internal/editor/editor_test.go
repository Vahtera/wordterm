package editor

import "testing"

func TestEditorBufferInsertAndDelete(t *testing.T) {
	e := NewEditor()
	e.InsertString("hello")
	if e.Text() != "hello" {
		t.Fatalf("unexpected text: %q", e.Text())
	}
	e.SetCursor(1)
	e.Backspace()
	if e.Text() != "ello" {
		t.Fatalf("backspace should remove the rune before the cursor: %q", e.Text())
	}
	if e.Cursor() != 0 {
		t.Fatalf("cursor should be 0 after backspace, got %d", e.Cursor())
	}
}

func TestEditorBufferWordCount(t *testing.T) {
	e := NewEditor()
	e.InsertString("one two\nthree")
	if e.WordCount() != 3 {
		t.Fatalf("expected 3 words, got %d", e.WordCount())
	}
}
