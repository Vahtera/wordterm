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

func (e *Editor) MoveLeft() {
	if e.cursor > 0 {
		e.cursor--
	}
}

func (e *Editor) MoveRight() {
	if e.cursor < len(e.buf) {
		e.cursor++
	}
}

func (e *Editor) Home() {
	lineStart := e.cursor
	for lineStart > 0 && e.buf[lineStart-1] != '\n' {
		lineStart--
	}
	e.cursor = lineStart
}

func (e *Editor) End() {
	lineEnd := e.cursor
	for lineEnd < len(e.buf) && e.buf[lineEnd] != '\n' {
		lineEnd++
	}
	e.cursor = lineEnd
}

func (e *Editor) MoveUp() {
	starts := e.lineStarts()
	if len(starts) <= 1 {
		return
	}
	currentLine := e.lineIndexForCursor(e.cursor)
	if currentLine == 0 {
		return
	}
	currentCol := e.cursor - starts[currentLine]
	prevStart := starts[currentLine-1]
	prevEnd := prevStart
	for prevEnd < len(e.buf) && e.buf[prevEnd] != '\n' {
		prevEnd++
	}
	if prevEnd-prevStart < currentCol {
		currentCol = prevEnd - prevStart
	}
	e.cursor = prevStart + currentCol
}

func (e *Editor) MoveDown() {
	starts := e.lineStarts()
	if len(starts) <= 1 {
		return
	}
	currentLine := e.lineIndexForCursor(e.cursor)
	if currentLine >= len(starts)-1 {
		return
	}
	currentCol := e.cursor - starts[currentLine]
	nextStart := starts[currentLine+1]
	nextEnd := nextStart
	for nextEnd < len(e.buf) && e.buf[nextEnd] != '\n' {
		nextEnd++
	}
	if nextEnd-nextStart < currentCol {
		currentCol = nextEnd - nextStart
	}
	e.cursor = nextStart + currentCol
}

func (e *Editor) PageUp() {
	// Keep it simple and coarse-grained for a usable editor.
	for i := 0; i < 10; i++ {
		e.MoveUp()
	}
}

func (e *Editor) PageDown() {
	for i := 0; i < 10; i++ {
		e.MoveDown()
	}
}

func (e *Editor) lineStarts() []int {
	starts := []int{0}
	for i, r := range e.buf {
		if r == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func (e *Editor) lineIndexForCursor(pos int) int {
	starts := e.lineStarts()
	if pos >= len(e.buf) {
		pos = len(e.buf)
	}
	idx := 0
	for i := 1; i < len(starts); i++ {
		if pos >= starts[i] {
			idx = i
			continue
		}
		break
	}
	return idx
}

func (e *Editor) WordCount() int {
	return len(strings.FieldsFunc(string(e.buf), func(r rune) bool {
		return unicode.IsSpace(r)
	}))
}
