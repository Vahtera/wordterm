package ui

import "testing"

func TestMinSize(t *testing.T) {
	w, h := MinSize()
	if w != 31 || h != 7 {
		t.Fatalf("MinSize = %d,%d; want 31,7", w, h)
	}
	if Compute(w-1, h, 0).OK || Compute(w, h-1, 0).OK || !Compute(w, h, 0).OK {
		t.Fatal("minimum size boundary wrong")
	}
}

func TestComputeAddsUp(t *testing.T) {
	for w := 31; w < 250; w += 7 {
		for _, pref := range []int{0, 5, 40, 500} {
			l := Compute(w, 24, pref)
			if !l.OK || l.NavW+l.EditW+chromeW != w {
				t.Fatalf("w=%d pref=%d: %+v", w, pref, l)
			}
			if l.NavW < MinNavWidth || l.EditW < MinEditorWidth {
				t.Fatalf("below minimum: %+v", l)
			}
		}
	}
}
