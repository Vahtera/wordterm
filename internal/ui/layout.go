// Package ui holds pure layout and rendering helpers (no application state).
package ui

const (
	MinNavWidth    = 8
	MinEditorWidth = 20
	MinPaneHeight  = 3

	// DefaultNavWidth is the Navigator width when none is configured.
	DefaultNavWidth = 16

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

// Compute divides the terminal. prefNav is the desired Navigator width (0 =
// DefaultNavWidth); the Editor gets everything that is left. The Navigator
// width is clamped so neither pane drops below its minimum.
func Compute(w, h, prefNav int) Layout {
	l := Layout{Width: w, Height: h}
	mw, mh := MinSize()
	if w < mw || h < mh {
		return l
	}
	avail := w - chromeW
	n := prefNav
	if n <= 0 {
		n = DefaultNavWidth
	}
	n = max(n, MinNavWidth)
	n = min(n, avail-MinEditorWidth)
	l.NavW, l.EditW = n, avail-n
	l.PaneH = h - chromeH
	l.OK = true
	return l
}
