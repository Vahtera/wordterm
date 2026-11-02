package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Theme holds the eight screen colors as "#rrggbb"; "" means the terminal's own
// color, so the zero Theme draws exactly as an unthemed terminal program.
type Theme struct {
	EditorBG, NavBG, OverlayBG             string
	NavFG, EditorFG, OverlayFG             string
	TitleFG, FrameFG                       string
}

func parseHex(h string) (r, g, b uint64, ok bool) {
	if len(h) != 7 || h[0] != '#' {
		return 0, 0, 0, false
	}
	v, err := strconv.ParseUint(h[1:], 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return v >> 16 & 0xff, v >> 8 & 0xff, v & 0xff, true
}

func sgr(fg, bg string) string {
	var p []string
	if r, g, b, ok := parseHex(fg); ok {
		p = append(p, fmt.Sprintf("38;2;%d;%d;%d", r, g, b))
	}
	if r, g, b, ok := parseHex(bg); ok {
		p = append(p, fmt.Sprintf("48;2;%d;%d;%d", r, g, b))
	}
	if len(p) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(p, ";") + "m"
}

// Paint draws s in the given colors. Colors are re-applied after every style
// reset inside s (cursor, inverted rows), so they last to the end of s.
func Paint(s, fg, bg string) string {
	base := sgr(fg, bg)
	if base == "" {
		return s
	}
	s = strings.ReplaceAll(s, "\x1b[m", "\x1b[m"+base)
	s = strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+base)
	return base + s + "\x1b[m"
}

func (t Theme) EditorPaint(s string) string  { return Paint(s, t.EditorFG, t.EditorBG) }
func (t Theme) OverlayPaint(s string) string { return Paint(s, t.OverlayFG, t.OverlayBG) }

// Box draws content (several lines) in a rounded frame with the overlay colors.
func (t Theme) Box(content string) string {
	lines := strings.Split(content, "\n")
	w := 0
	for _, l := range lines {
		w = max(w, ansi.StringWidth(l))
	}
	frame := func(s string) string { return Paint(s, t.FrameFG, t.OverlayBG) }
	out := []string{frame("╭" + strings.Repeat("─", w+2) + "╮")}
	for _, l := range lines {
		out = append(out, frame("│")+t.OverlayPaint(Pad(" "+l, w+1)+" ")+frame("│"))
	}
	return strings.Join(append(out, frame("╰"+strings.Repeat("─", w+2)+"╯")), "\n")
}
