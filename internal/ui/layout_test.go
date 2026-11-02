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

func TestNavigatorWidthIsFixedAndEditorTakesTheRest(t *testing.T) {
	if l := Compute(100, 24, 0); l.NavW != DefaultNavWidth || l.EditW != 100-3-DefaultNavWidth {
		t.Fatalf("default: %+v", l)
	}
	if l := Compute(100, 24, 30); l.NavW != 30 || l.EditW != 67 {
		t.Fatalf("30 wide navigator: %+v", l)
	}
	if l := Compute(150, 24, 30); l.NavW != 30 { // a wider window only widens the editor
		t.Fatalf("resize: %+v", l)
	}
	if l := Compute(31, 7, 0); l.NavW != MinNavWidth || l.EditW != MinEditorWidth {
		t.Fatalf("minimum window: %+v", l)
	}
}
