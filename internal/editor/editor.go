// Package editor is a plain-text buffer with word wrap, grapheme-aware
// cursor movement and undo/redo. The text lives in memory; the app writes
// it to the working copy (never on every keystroke).
package editor

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/rivo/uniseg"
)

const maxUndo = 2000

type pos struct{ r, c int } // row, column (in grapheme clusters)

// edit replaces [at,delEnd) (text del) by ins, which ends at insEnd.
type edit struct {
	at, delEnd, insEnd, before pos
	del, ins                   string
}

type line struct {
	cl     []string // one entry per grapheme cluster
	starts []int    // wrap cache: first cluster of each visual row
	w      int      // width the cache was computed for
	wrap   bool     // wrap mode the cache was computed for
}

type Editor struct {
	lines       []*line
	cur         pos
	wantX       int
	w, h, top   int
	wrap        bool // word wrap on; off = horizontal scrolling
	left        int  // first visible display column when wrap is off
	undo, redo  []edit
	dirty       bool
}

var cursorStyle = lipgloss.NewStyle().Reverse(true)

func clusters(s string) []string {
	var out []string
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		out = append(out, g.Str())
	}
	return out
}

func width(c string) int {
	if c == "\t" {
		return 4
	}
	return uniseg.StringWidth(c)
}

// New creates an editor holding text (CRLF is normalised to LF).
func New(text string) *Editor {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	e := &Editor{w: 80, h: 24, wrap: true}
	for _, p := range strings.Split(text, "\n") {
		e.lines = append(e.lines, &line{cl: clusters(p)})
	}
	return e
}

func (e *Editor) Text() string { return e.textOf(pos{}, e.end()) }

func (e *Editor) end() pos {
	r := len(e.lines) - 1
	return pos{r, len(e.lines[r].cl)}
}

func (e *Editor) Dirty() bool      { return e.dirty }
func (e *Editor) ClearDirty()      { e.dirty = false }
func (e *Editor) WordCount() int   { return len(strings.Fields(e.Text())) }
func (e *Editor) SetSize(w, h int) { e.w, e.h = max(w, 1), max(h, 1); e.fixScroll() }

// SetWrap switches word wrap on or off (off scrolls long lines horizontally).
func (e *Editor) SetWrap(on bool) { e.wrap, e.left = on, 0; e.fixScroll() }

// wrapLine returns the first cluster index of every visual row when the
// line is word-wrapped at w columns. Spaces may hang past the edge.
func wrapLine(cl []string, w int) []int {
	starts := []int{0}
	x, lastSpace := 0, -1
	for i, c := range cl {
		cw := width(c)
		if c != " " && x+cw > w && i > starts[len(starts)-1] {
			brk := i
			if lastSpace > starts[len(starts)-1] {
				brk = lastSpace
			}
			starts = append(starts, brk)
			x = 0
			for _, p := range cl[brk:i] {
				x += width(p)
			}
			lastSpace = -1
		}
		x += cw
		if c == " " {
			lastSpace = i + 1
		}
	}
	return starts
}

func (e *Editor) segs(r int) []int {
	l := e.lines[r]
	if l.starts == nil || l.w != e.w || l.wrap != e.wrap {
		if e.wrap {
			l.starts = wrapLine(l.cl, e.w)
		} else {
			l.starts = []int{0}
		}
		l.w, l.wrap = e.w, e.wrap
	}
	return l.starts
}

func (e *Editor) segEnd(r, s int) int {
	if st := e.segs(r); s+1 < len(st) {
		return st[s+1]
	}
	return len(e.lines[r].cl)
}

func (e *Editor) segOf(r, c int) int {
	s := 0
	for i, st := range e.segs(r) {
		if st <= c {
			s = i
		}
	}
	return s
}

func (e *Editor) xOf(r, c int) int {
	x := 0
	for _, p := range e.lines[r].cl[e.segs(r)[e.segOf(r, c)]:c] {
		x += width(p)
	}
	return x
}

// colAt returns the column in visual row (r,s) closest to display column x.
func (e *Editor) colAt(r, s, x int) int {
	cl, st := e.lines[r].cl, e.segs(r)
	start, end := st[s], e.segEnd(r, s)
	acc, col, found := 0, end, false
	for i := start; i < end; i++ {
		if w := width(cl[i]); acc+w > x {
			col, found = i, true
			break
		} else {
			acc += w
		}
	}
	if !found && s+1 < len(st) && end > start {
		col = end - 1 // never land on the next row's first cluster
	}
	return col
}

func (e *Editor) vrow() int {
	n := 0
	for r := 0; r < e.cur.r; r++ {
		n += len(e.segs(r))
	}
	return n + e.segOf(e.cur.r, e.cur.c)
}

func (e *Editor) fixScroll() {
	v := e.vrow()
	if v < e.top {
		e.top = v
	}
	if v >= e.top+e.h {
		e.top = v - e.h + 1
	}
	e.top = max(e.top, 0)
	if e.wrap {
		e.left = 0
		return
	}
	if cx := e.xOf(e.cur.r, e.cur.c); cx < e.left {
		e.left = cx
	} else if cx+1 > e.left+e.w {
		e.left = cx + 1 - e.w
	}
}

func (e *Editor) settle() {
	e.wantX = e.xOf(e.cur.r, e.cur.c)
	e.fixScroll()
}

// Lines returns the visible rows (at most h). The cursor is drawn inverted when focused.
func (e *Editor) Lines(focused bool) []string {
	var out []string
	v := 0
	for r := 0; r < len(e.lines) && len(out) < e.h; r++ {
		st := e.segs(r)
		if v+len(st) <= e.top {
			v += len(st)
			continue
		}
		for s := range st {
			if v >= e.top && len(out) < e.h {
				out = append(out, e.renderRow(r, s, focused))
			}
			v++
		}
	}
	return out
}

func (e *Editor) renderRow(r, s int, focused bool) string {
	cl := e.lines[r].cl
	start, end := e.segs(r)[s], e.segEnd(r, s)
	var sb strings.Builder
	if !e.wrap {
		x, clipped := 0, false
		for i := start; i < end && !clipped; i++ {
			c, cw := cl[i], width(cl[i])
			switch {
			case x < e.left:
			case x-e.left+cw > e.w:
				clipped = true
			default:
				if c == "\t" {
					c = "    "
				}
				if focused && e.cur == (pos{r, i}) {
					c = cursorStyle.Render(c)
				}
				sb.WriteString(c)
			}
			x += cw
		}
		if focused && !clipped && e.cur == (pos{r, end}) && x-e.left < e.w {
			sb.WriteString(cursorStyle.Render(" "))
		}
		return sb.String()
	}
	for i := start; i < end; i++ {
		c := cl[i]
		if c == "\t" {
			c = "    "
		}
		if focused && e.cur == (pos{r, i}) {
			c = cursorStyle.Render(c)
		}
		sb.WriteString(c)
	}
	if focused && e.cur == (pos{r, end}) && s == len(e.segs(r))-1 {
		sb.WriteString(cursorStyle.Render(" "))
	}
	return sb.String()
}

// ---- cursor movement ----

func (e *Editor) Left() {
	switch {
	case e.cur.c > 0:
		e.cur.c--
	case e.cur.r > 0:
		e.cur.r--
		e.cur.c = len(e.lines[e.cur.r].cl)
	}
	e.settle()
}

func (e *Editor) Right() {
	switch {
	case e.cur.c < len(e.lines[e.cur.r].cl):
		e.cur.c++
	case e.cur.r < len(e.lines)-1:
		e.cur.r++
		e.cur.c = 0
	}
	e.settle()
}

func (e *Editor) moveVert(d int) {
	r, s := e.cur.r, e.segOf(e.cur.r, e.cur.c)
	if d < 0 {
		switch {
		case s > 0:
			s--
		case r > 0:
			r--
			s = len(e.segs(r)) - 1
		default:
			e.cur.c = 0
			e.fixScroll()
			return
		}
	} else {
		switch {
		case s+1 < len(e.segs(r)):
			s++
		case r+1 < len(e.lines):
			r, s = r+1, 0
		default:
			e.cur.c = len(e.lines[r].cl)
			e.fixScroll()
			return
		}
	}
	e.cur = pos{r, e.colAt(r, s, e.wantX)}
	e.fixScroll()
}

func (e *Editor) Up()       { e.moveVert(-1) }
func (e *Editor) Down()     { e.moveVert(1) }
func (e *Editor) PageUp()   { e.page(-1) }
func (e *Editor) PageDown() { e.page(1) }

func (e *Editor) page(d int) {
	for i := 0; i < e.h; i++ {
		e.moveVert(d)
	}
}

func (e *Editor) Home() {
	e.cur.c = e.segs(e.cur.r)[e.segOf(e.cur.r, e.cur.c)]
	e.settle()
}

func (e *Editor) End() {
	s := e.segOf(e.cur.r, e.cur.c)
	end := e.segEnd(e.cur.r, s)
	if s+1 < len(e.segs(e.cur.r)) {
		end--
	}
	e.cur.c = end
	e.settle()
}

// ---- editing ----

func sanitize(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || (r >= 32 && r != 127) {
			return r
		}
		return -1
	}, s)
}

// Insert inserts text (typed or pasted) at the cursor.
func (e *Editor) Insert(s string) {
	if s = sanitize(s); s != "" {
		e.do(edit{at: e.cur, ins: s, before: e.cur})
	}
}

func (e *Editor) Backspace() {
	from := e.cur
	switch {
	case e.cur.c > 0:
		from.c--
	case e.cur.r > 0:
		from = pos{e.cur.r - 1, len(e.lines[e.cur.r-1].cl)}
	default:
		return
	}
	e.do(edit{at: from, del: e.textOf(from, e.cur), delEnd: e.cur, before: e.cur})
}

func (e *Editor) Delete() {
	to := e.cur
	switch {
	case e.cur.c < len(e.lines[e.cur.r].cl):
		to.c++
	case e.cur.r < len(e.lines)-1:
		to = pos{e.cur.r + 1, 0}
	default:
		return
	}
	e.do(edit{at: e.cur, del: e.textOf(e.cur, to), delEnd: to, before: e.cur})
}

func (e *Editor) do(ed edit) {
	e.run(&ed)
	e.redo = nil
	if n := len(e.undo); n > 0 {
		last := &e.undo[n-1]
		typing := last.del == "" && ed.del == "" && last.insEnd == ed.at &&
			!strings.Contains(last.ins+ed.ins, "\n") && !strings.HasSuffix(last.ins, " ")
		backspacing := last.ins == "" && ed.ins == "" && ed.delEnd == last.at &&
			!strings.Contains(last.del+ed.del, "\n")
		switch {
		case typing:
			last.ins += ed.ins
			last.insEnd = ed.insEnd
			return
		case backspacing:
			last.del = ed.del + last.del
			last.at = ed.at
			return
		}
	}
	e.undo = append(e.undo, ed)
	if len(e.undo) > maxUndo {
		e.undo = e.undo[1:]
	}
}

// run applies ed forwards and leaves the cursor after it.
func (e *Editor) run(ed *edit) {
	if ed.del != "" {
		e.removeRange(ed.at, ed.delEnd)
	}
	ed.insEnd = ed.at
	if ed.ins != "" {
		ed.insEnd = e.insertAt(ed.at, ed.ins)
	}
	e.cur, e.dirty = ed.insEnd, true
	e.settle()
}

func (e *Editor) Undo() {
	n := len(e.undo)
	if n == 0 {
		return
	}
	ed := e.undo[n-1]
	e.undo = e.undo[:n-1]
	if ed.insEnd != ed.at {
		e.removeRange(ed.at, ed.insEnd)
	}
	if ed.del != "" {
		e.insertAt(ed.at, ed.del)
	}
	e.redo = append(e.redo, ed)
	e.cur, e.dirty = ed.before, true
	e.settle()
}

func (e *Editor) Redo() {
	n := len(e.redo)
	if n == 0 {
		return
	}
	ed := e.redo[n-1]
	e.redo = e.redo[:n-1]
	e.run(&ed)
	e.undo = append(e.undo, ed)
}

func (e *Editor) textOf(a, b pos) string {
	var sb strings.Builder
	for r := a.r; r <= b.r; r++ {
		cl := e.lines[r].cl
		from, to := 0, len(cl)
		if r == a.r {
			from = a.c
		}
		if r == b.r {
			to = b.c
		}
		sb.WriteString(strings.Join(cl[from:to], ""))
		if r < b.r {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func (e *Editor) removeRange(a, b pos) {
	la, lb := e.lines[a.r], e.lines[b.r]
	nc := make([]string, 0, a.c+len(lb.cl)-b.c)
	nc = append(nc, la.cl[:a.c]...)
	nc = append(nc, lb.cl[b.c:]...)
	la.cl, la.starts = nc, nil
	if b.r > a.r {
		e.lines = append(e.lines[:a.r+1], e.lines[b.r+1:]...)
	}
}

func (e *Editor) insertAt(at pos, s string) pos {
	parts := strings.Split(s, "\n")
	l := e.lines[at.r]
	tail := append([]string(nil), l.cl[at.c:]...)
	first := append(append([]string(nil), l.cl[:at.c]...), clusters(parts[0])...)
	l.starts = nil
	if len(parts) == 1 {
		end := pos{at.r, len(first)}
		l.cl = append(first, tail...)
		return end
	}
	l.cl = first
	var nl []*line
	for _, p := range parts[1 : len(parts)-1] {
		nl = append(nl, &line{cl: clusters(p)})
	}
	lastCl := clusters(parts[len(parts)-1])
	end := pos{at.r + len(parts) - 1, len(lastCl)}
	nl = append(nl, &line{cl: append(lastCl, tail...)})
	e.lines = append(e.lines[:at.r+1], append(nl, e.lines[at.r+1:]...)...)
	return end
}
