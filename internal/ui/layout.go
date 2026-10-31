// Package ui holds pure layout and rendering helpers (no application state).
package ui

const (
	MinNavWidth    = 8
	MinEditorWidth = 20
	MinPaneHeight  = 3

	// Fixed chrome: 3 vertical border columns (left, divider, right) and
	// 4 rows (top border, divider row, status bar, bottom border).
	chromeW = 3
	chromeH = 4
)

// Layout describes how the terminal is divided. OK is false when the
// terminal is too small to be usable.
type Layout struct {
	Width, Height int
	NavW, EditW   int
	PaneH         int
	OK            bool
}

// MinSize returns the smallest usable terminal size, borders included.
func MinSize() (w, h int) {
	return MinNavWidth + MinEditorWidth + chromeW, MinPaneHeight + chromeH
}

// Compute divides the terminal. prefEditor is the desired editor width;
// pass 0 for the default (70% of the available width). The editor is
// clamped so neither pane drops below its minimum.
func Compute(w, h, prefEditor int) Layout {
	l := Layout{Width: w, Height: h}
	mw, mh := MinSize()
	if w < mw || h < mh {
		return l
	}
	avail := w - chromeW
	e := prefEditor
	if e <= 0 {
		e = avail * 70 / 100
	}
	if e < MinEditorWidth {
		e = MinEditorWidth
	}
	if e > avail-MinNavWidth {
		e = avail - MinNavWidth
	}
	l.EditW, l.NavW = e, avail-e
	l.PaneH = h - chromeH
	l.OK = true
	return l
}
