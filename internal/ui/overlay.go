package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Overlay draws box (multi-line, may contain styles) over base at column x, row y.
func Overlay(base, box string, x, y int) string {
	lines := strings.Split(base, "\n")
	total := lipgloss.Width(lines[0])
	for i, b := range strings.Split(box, "\n") {
		r := y + i
		if r < 0 || r >= len(lines) {
			continue
		}
		b = ansi.Truncate(b, max(total-x, 0), "")
		line := lines[r]
		lines[r] = Pad(line, x) + "\x1b[0m" + b + "\x1b[0m" + ansi.TruncateLeft(line, x+ansi.StringWidth(b), "")
	}
	return strings.Join(lines, "\n")
}
