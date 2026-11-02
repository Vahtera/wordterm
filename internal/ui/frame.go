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

func (p PaneContent) row(i, w int, fg, bg, titleFG string) string {
	if i == 0 {
		h := Pad(" "+p.Header, w)
		if p.Focused {
			h = invert.Render(h)
		}
		return Paint(h, titleFG, bg)
	}
	if i-1 < len(p.Lines) {
		return Paint(Pad(p.Lines[i-1], w), fg, bg)
	}
	return Paint(strings.Repeat(" ", w), fg, bg)
}

// Frame draws the two-pane window with rounded Unicode borders, in the theme's colors.
func Frame(l Layout, th Theme, title string, nav, edit PaneContent, status string) string {
	bar := func(n int) string { return strings.Repeat("─", n) }
	fr := func(s string) string { return Paint(s, th.FrameFG, th.NavBG) }
	head := " " + title + " "
	top := fr("╭─") + Paint(head, th.TitleFG, th.NavBG) + fr(bar(max(0, l.Width-3-ansi.StringWidth(head)))+"╮")

	var b strings.Builder
	b.WriteString(top + "\n")
	for i := 0; i < l.PaneH; i++ {
		b.WriteString(fr("│") + nav.row(i, l.NavW, th.NavFG, th.NavBG, th.TitleFG) + fr("│") +
			edit.row(i, l.EditW, th.EditorFG, th.EditorBG, th.TitleFG) + fr("│") + "\n")
	}
	b.WriteString(fr("├"+bar(l.NavW)+"┴"+bar(l.EditW)+"┤") + "\n")
	b.WriteString(fr("│") + Paint(Pad(" "+status, l.Width-2), th.NavFG, th.NavBG) + fr("│") + "\n")
	b.WriteString(fr("╰" + bar(l.Width-2) + "╯"))
	return b.String()
}

// TooSmall returns the centred "Terminal Size too Small" screen.
func TooSmall(w, h int) string {
	msg := ansi.Truncate("Terminal Size too Small", w, "")
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, msg)
}
