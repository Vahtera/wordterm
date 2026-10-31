// Package navigator holds the tree selection/expansion state and renders the tree.
package navigator

import (
	"charm.land/lipgloss/v2"

	"wordterm/internal/project"
	"wordterm/internal/ui"
)

type Kind int

const (
	Root Kind = iota
	Chapter
	Scene
)

// Item identifies a tree row. Selection is kept by identity, so it survives edits.
type Item struct {
	Kind            Kind
	Chapter, Scene string
}

type Navigator struct {
	collapsed map[string]bool
	sel       Item
	idx, top  int
}

var selStyle = lipgloss.NewStyle().Reverse(true) // inverted colours, works on any theme

func New() *Navigator { return &Navigator{collapsed: map[string]bool{}} }

func (n *Navigator) items(m *project.Meta) []Item {
	items := []Item{{Kind: Root}}
	for _, c := range m.Chapters {
		items = append(items, Item{Kind: Chapter, Chapter: c.Name})
		if n.collapsed[c.Name] {
			continue
		}
		for _, s := range c.Scenes {
			items = append(items, Item{Scene, c.Name, s})
		}
	}
	return items
}

// fix re-resolves the selection after the tree changed and returns the visible rows.
func (n *Navigator) fix(m *project.Meta) []Item {
	items := n.items(m)
	i := -1
	for j, it := range items {
		if it == n.sel {
			i = j
		}
	}
	if i < 0 {
		i = min(n.idx, len(items)-1)
		n.sel = items[i]
	}
	n.idx = i
	return items
}

func (n *Navigator) Selected(m *project.Meta) Item { n.fix(m); return n.sel }
func (n *Navigator) Select(it Item)                { n.sel = it }
func (n *Navigator) Expand(chapter string)         { delete(n.collapsed, chapter) }
func (n *Navigator) Toggle(chapter string)         { n.collapsed[chapter] = !n.collapsed[chapter] }

// Rename keeps expansion and selection state when a chapter is renamed.
func (n *Navigator) Rename(old, name string) {
	if n.collapsed[old] {
		delete(n.collapsed, old)
		n.collapsed[name] = true
	}
	if n.sel.Chapter == old {
		n.sel.Chapter = name
	}
}

func (n *Navigator) Up(m *project.Meta) {
	items := n.fix(m)
	if n.idx > 0 {
		n.idx--
		n.sel = items[n.idx]
	}
}

func (n *Navigator) Down(m *project.Meta) {
	items := n.fix(m)
	if n.idx < len(items)-1 {
		n.idx++
		n.sel = items[n.idx]
	}
}

// Left collapses a chapter, or moves to the parent.
func (n *Navigator) Left(m *project.Meta) {
	n.fix(m)
	switch n.sel.Kind {
	case Scene:
		n.sel = Item{Kind: Chapter, Chapter: n.sel.Chapter}
	case Chapter:
		if n.collapsed[n.sel.Chapter] {
			n.sel = Item{Kind: Root}
		} else {
			n.collapsed[n.sel.Chapter] = true
		}
	}
}

// Right expands a chapter, or moves to the first child.
func (n *Navigator) Right(m *project.Meta) {
	items := n.fix(m)
	if n.idx+1 >= len(items) {
		return
	}
	next := items[n.idx+1]
	switch n.sel.Kind {
	case Root:
		n.sel = next
	case Chapter:
		if n.collapsed[n.sel.Chapter] {
			delete(n.collapsed, n.sel.Chapter)
		} else if next.Kind == Scene && next.Chapter == n.sel.Chapter {
			n.sel = next
		}
	}
}

func (n *Navigator) label(m *project.Meta, it Item, open project.SceneRef) string {
	ci := m.ChapterIndex(it.Chapter)
	lastC := ci == len(m.Chapters)-1
	switch it.Kind {
	case Root:
		return m.Name
	case Chapter:
		branch, mark := "├─ ", "▾ "
		if lastC {
			branch = "└─ "
		}
		if n.collapsed[it.Chapter] {
			mark = "▸ "
		}
		return branch + mark + it.Chapter
	}
	pad, branch := "│  ", "├─ "
	if lastC {
		pad = "   "
	}
	if m.SceneIndex(it.Chapter, it.Scene) == len(m.Chapters[ci].Scenes)-1 {
		branch = "└─ "
	}
	s := pad + branch + it.Scene
	if open == (project.SceneRef{Chapter: it.Chapter, Scene: it.Scene}) {
		s += " ◂"
	}
	return s
}

// Lines renders h rows of width w; the selected row is drawn inverted.
func (n *Navigator) Lines(m *project.Meta, open project.SceneRef, w, h int) []string {
	items := n.fix(m)
	if n.idx < n.top {
		n.top = n.idx
	}
	if n.idx >= n.top+h {
		n.top = n.idx - h + 1
	}
	n.top = max(n.top, 0)
	var out []string
	for i := n.top; i < len(items) && i < n.top+h; i++ {
		row := ui.Pad(" "+n.label(m, items[i], open), w)
		if i == n.idx {
			row = selStyle.Render(row)
		}
		out = append(out, row)
	}
	return out
}
