package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Pad truncates s to w display columns and pads it with spaces.
func Pad(s string, w int) string {
	s = ansi.Truncate(s, w, "")
	if gap := w - ansi.StringWidth(s); gap > 0 {
		s += strings.Repeat(" ", gap)
	}
	return s
}

// PaneContent is the text shown in one pane. Header is drawn on the first
// row; Focused renders the header inverted.
type PaneContent struct {
	Header  string
	Lines   []string
	Focused bool
}

var invert = lipgloss.NewStyle().Reverse(true)

func (p PaneContent) row(i, w int) string {
	if i == 0 {
		h := Pad(" "+p.Header, w)
		if p.Focused {
			return invert.Render(h)
		}
		return h
	}
	if i-1 < len(p.Lines) {
		return Pad(p.Lines[i-1], w)
	}
	return strings.Repeat(" ", w)
}

// Frame draws the two-pane window with rounded Unicode borders.
func Frame(l Layout, title string, nav, edit PaneContent, status string) string {
	bar := func(n int) string { return strings.Repeat("─", n) }
	t := "─ " + title + " "
	top := "╭" + t + bar(max(0, l.Width-2-ansi.StringWidth(t))) + "╮"

	var b strings.Builder
	b.WriteString(top + "\n")
	for i := 0; i < l.PaneH; i++ {
		b.WriteString("│" + nav.row(i, l.NavW) + "│" + edit.row(i, l.EditW) + "│\n")
	}
	b.WriteString("├" + bar(l.NavW) + "┴" + bar(l.EditW) + "┤\n")
	b.WriteString("│" + Pad(" "+status, l.Width-2) + "│\n")
	b.WriteString("╰" + bar(l.Width-2) + "╯")
	return b.String()
}

// TooSmall returns the centred "Terminal Size too Small" screen.
func TooSmall(w, h int) string {
	msg := ansi.Truncate("Terminal Size too Small", w, "")
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, msg)
}
