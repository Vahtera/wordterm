package editor

import (
	"strings"
	"unicode"
)

// Editor holds a simple plain-text buffer with cursor state.
type Editor struct {
	buf    []rune
	cursor int
}

func NewEditor() *Editor {
	return &Editor{buf: make([]rune, 0), cursor: 0}
}

func (e *Editor) InsertString(s string) {
	if s == "" {
		return
	}
	insert := []rune(s)
	if e.cursor > len(e.buf) {
		e.cursor = len(e.buf)
	}
	prefix := append([]rune(nil), e.buf[:e.cursor]...)
	suffix := append([]rune(nil), e.buf[e.cursor:]...)
	e.buf = append(prefix, append(insert, suffix...)...)
	e.cursor += len(insert)
}

func (e *Editor) InsertRune(r rune) {
	if e.cursor > len(e.buf) {
		e.cursor = len(e.buf)
	}
	prefix := append([]rune(nil), e.buf[:e.cursor]...)
	suffix := append([]rune(nil), e.buf[e.cursor:]...)
	e.buf = append(prefix, append([]rune{r}, suffix...)...)
	e.cursor++
}

func (e *Editor) Backspace() {
	if e.cursor == 0 {
		return
	}
	e.cursor--
	e.buf = append(e.buf[:e.cursor], e.buf[e.cursor+1:]...)
}

func (e *Editor) Delete() {
	if e.cursor >= len(e.buf) {
		return
	}
	e.buf = append(e.buf[:e.cursor], e.buf[e.cursor+1:]...)
}

func (e *Editor) SetCursor(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos > len(e.buf) {
		pos = len(e.buf)
	}
	e.cursor = pos
}

func (e *Editor) Cursor() int {
	return e.cursor
}

func (e *Editor) Text() string {
	return string(e.buf)
}

func (e *Editor) SetText(text string) {
	e.buf = []rune(text)
	if e.cursor > len(e.buf) {
		e.cursor = len(e.buf)
	}
}

func (e *Editor) WordCount() int {
	return len(strings.FieldsFunc(string(e.buf), func(r rune) bool {
		return unicode.IsSpace(r)
	}))
}
