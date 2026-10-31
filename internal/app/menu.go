package app

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Vahtera/wordterm/internal/settings"
	"github.com/Vahtera/wordterm/internal/ui"
)

var (
	invert   = lipgloss.NewStyle().Reverse(true)
	boxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
)

func onOff(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}

// keyName is the name used for key bindings; shifted "+" is normalised.
func keyName(msg tea.KeyPressMsg) string {
	k := msg.String()
	if k == "shift+=" || msg.Key().Text == "+" {
		return "+"
	}
	if msg.Key().Text == "-" {
		return "-"
	}
	return k
}

type menuState struct{ tab, item int }

type menuItem struct {
	label string
	sep   bool
	run   func(m *Model) tea.Cmd
}

type menuTab struct {
	name  string
	items []menuItem
}

func (m Model) menus() []menuTab {
	do := func(f func(m *Model)) func(*Model) tea.Cmd {
		return func(m *Model) tea.Cmd { f(m); return nil }
	}
	return []menuTab{
		{"File", []menuItem{
			{label: "New Project...", run: (*Model).newProject},
			{label: "Open Project...", run: (*Model).openProject},
			{label: "Save", run: do(func(m *Model) { m.askSave() })},
			{label: "Save As...", run: (*Model).saveAs},
			{sep: true},
			{label: "Quit", run: (*Model).requestQuit},
		}},
		{"Settings", []menuItem{
			{label: "Editor Size", run: do(func(m *Model) { m.adjust = adjEditor })},
			{label: "Write Size", run: do(func(m *Model) { m.adjust = adjWrite })},
			{label: "Screenwriter Mode (" + m.cfg.Keys["screenwriter"] + ")", run: do(func(m *Model) { m.toggleScreenwriter() })},
			{label: "Rebind Keys", run: do(func(m *Model) { m.rebind = &rebindState{} })},
			{label: "Word Wrap: " + onOff(m.cfg.WordWrap), run: do(func(m *Model) { m.toggleWrap() })},
			{label: "Autosave: " + onOff(m.cfg.Autosave), run: do(func(m *Model) { m.toggleAutosave() })},
			{label: "Default Save Path...", run: (*Model).askSavePath},
			{label: "Author Name...", run: (*Model).askAuthor},
		}},
		{"Help", []menuItem{
			{label: "About...", run: do(func(m *Model) { m.overlay = ovAbout })},
			{label: "Keys", run: do(func(m *Model) { m.overlay = ovHelp })},
		}},
	}
}

func (m *Model) menuKey(k string) tea.Cmd {
	tabs, mn := m.menus(), m.menu
	items := tabs[mn.tab].items
	step := func(d int) {
		for i := 0; i < len(items); i++ {
			mn.item = (mn.item + d + len(items)) % len(items)
			if !items[mn.item].sep {
				return
			}
		}
	}
	switch k {
	case "esc":
		m.menu = nil
	case "left":
		mn.tab, mn.item = (mn.tab+len(tabs)-1)%len(tabs), 0
	case "right":
		mn.tab, mn.item = (mn.tab+1)%len(tabs), 0
	case "up":
		step(-1)
	case "down":
		step(1)
	case "enter":
		it := items[mn.item]
		m.menu = nil
		if it.run != nil {
			return it.run(m)
		}
	}
	return nil
}

func (m Model) menuBox() string {
	tabs, mn := m.menus(), m.menu
	w, hw := 0, 0
	for _, t := range tabs {
		hw += len(t.name) + 2
	}
	w = hw
	for _, it := range tabs[mn.tab].items {
		w = max(w, len([]rune(it.label))+2)
	}
	var head strings.Builder
	for i, t := range tabs {
		s := " " + t.name + " "
		if i == mn.tab {
			s = invert.Render(s)
		}
		head.WriteString(s)
	}
	head.WriteString(strings.Repeat(" ", w-hw))
	lines := []string{head.String()}
	for i, it := range tabs[mn.tab].items {
		if it.sep {
			lines = append(lines, strings.Repeat("─", w))
			continue
		}
		row := ui.Pad(" "+it.label, w)
		if i == mn.item {
			row = invert.Render(row)
		}
		lines = append(lines, row)
	}
	return boxStyle.Render(strings.Join(lines, "\n"))
}

func (m *Model) saveSettings() {
	if err := m.cfg.Save(); err != nil {
		m.msg = "Cannot save settings: " + err.Error()
	}
}

func (m *Model) toggleWrap() {
	m.cfg.WordWrap = !m.cfg.WordWrap
	m.saveSettings()
	m.syncSize()
}

func (m *Model) toggleAutosave() {
	m.cfg.Autosave = !m.cfg.Autosave
	m.lastAuto = time.Now()
	m.saveSettings()
	if m.msg == "" {
		if m.cfg.Autosave {
			m.msg = fmt.Sprintf("Autosave ON (every %d s)", m.cfg.AutosaveInterval)
		} else {
			m.msg = "Autosave OFF"
		}
	}
}

// dispatch runs the action bound to key k, otherwise passes the key to the focused pane.
func (m *Model) dispatch(msg tea.KeyPressMsg, k string) tea.Cmd {
	if id := m.cfg.ActionFor(k); id != "" && (m.focus == FocusNavigator || !settings.NavOnly(id)) {
		return m.doAction(id)
	}
	if m.focus == FocusNavigator {
		m.navKey(msg)
	} else if m.hasScene() {
		m.editorKey(msg)
	}
	return nil
}

func (m *Model) doAction(id string) tea.Cmd {
	switch id {
	case "help":
		m.overlay = ovHelp
	case "screenwriter":
		m.toggleScreenwriter()
	case "switch_pane":
		if m.screenwriter {
			break // the Navigator is hidden
		}
		if m.focus == FocusNavigator {
			m.focus = FocusEditor
		} else {
			m.focus = FocusNavigator
		}
	case "save":
		m.askSave()
	case "reload":
		m.requestReload()
	case "quit":
		return m.requestQuit()
	case "rename":
		m.renameItem()
	case "move_up":
		m.moveItem(-1)
	case "move_down":
		m.moveItem(1)
	case "add":
		m.addItem()
	case "delete":
		m.deleteItem()
	}
	return nil
}
