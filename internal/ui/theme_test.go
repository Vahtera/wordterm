package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestPaint(t *testing.T) {
	if got := Paint("abc", "", ""); got != "abc" {
		t.Fatalf("an empty theme must not change the text, got %q", got)
	}
	got := Paint("a\x1b[7mb\x1b[mc", "#ff0000", "#000000")
	base := "\x1b[38;2;255;0;0;48;2;0;0;0m"
	if !strings.HasPrefix(got, base) || !strings.Contains(got, "\x1b[m"+base+"c") || !strings.HasSuffix(got, "\x1b[m") {
		t.Fatalf("colors must be re-applied after a reset: %q", got)
	}
}

func TestBoxIsRectangular(t *testing.T) {
	box := Theme{FrameFG: "#ffffff", OverlayBG: "#000000"}.Box("one\ntwo three")
	for _, l := range strings.Split(box, "\n") {
		if w := ansi.StringWidth(l); w != 13 {
			t.Fatalf("line %q has width %d, want 13", l, w)
		}
	}
}
