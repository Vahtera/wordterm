package app

import (
	"fmt"
	"strings"

	"wordterm/internal/editor"
	"wordterm/internal/project"
	"wordterm/internal/settings"

	tea "charm.land/bubbletea/v2"
)

type Model struct {
	project     *project.Project
	editor      *editor.Editor
	settings    settings.Settings
	activePane  string
	width       int
	height      int
	showMessage string
}

func NewModel() *Model {
	p := project.NewProject("Example", "Author")
	e := editor.NewEditor()
	e.InsertString("This is a sample scene for Wordterm.\nIt is a simple writing app.")
	return &Model{
		project:    p,
		editor:     e,
		settings:   settings.Default(),
		activePane: "editor",
		width:      100,
		height:     30,
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab", "ctrl+tab":
			if m.activePane == "editor" {
				m.activePane = "navigator"
			} else {
				m.activePane = "editor"
			}
			return m, nil
		case "left":
			if m.activePane == "navigator" {
				m.showMessage = "Navigator focus"
			}
			return m, nil
		case "right":
			if m.activePane == "editor" {
				m.showMessage = "Editor focus"
			}
			return m, nil
		case "backspace":
			m.editor.Backspace()
			return m, nil
		case "delete":
			m.editor.Delete()
			return m, nil
		case "enter":
			m.editor.InsertString("\n")
			return m, nil
		default:
			if len(msg.String()) == 1 && msg.String() != "" {
				m.editor.InsertString(msg.String())
			}
		}
	}
	return m, nil
}

func (m *Model) View() tea.View {
	if m.width < 60 || m.height < 12 {
		return tea.NewView(centerText("Terminal Size too Small", m.width, m.height))
	}

	projectName := m.project.Name
	navigator := "Project\n└─ Chapter 1\n   ├─ Scene 1\n   └─ Scene 2"
	editorText := m.editor.Text()
	if editorText == "" {
		editorText = "Start writing..."
	}
	selectedIndicator := " "
	if m.activePane == "editor" {
		selectedIndicator = "[Editor]"
	} else {
		selectedIndicator = "[Navigator]"
	}

	status := fmt.Sprintf("Words: %d | Scene: %d | Chapter: %d | Project: %d", m.editor.WordCount(), m.editor.WordCount(), m.editor.WordCount(), m.editor.WordCount())
	body := strings.Builder{}
	body.WriteString(fmt.Sprintf("┌─ Wordterm ───────────────────────────────────────────────┐\n"))
	body.WriteString(fmt.Sprintf("│ %-42s │\n", projectName))
	body.WriteString(fmt.Sprintf("│ %-42s │\n", "  "+navigator))
	body.WriteString(fmt.Sprintf("├─────────────────┴──────────────────────────────────────┤\n"))
	body.WriteString(fmt.Sprintf("│ %s %-48s │\n", selectedIndicator, truncate(editorText, 48)))
	body.WriteString(fmt.Sprintf("│ %-52s │\n", ""))
	body.WriteString(fmt.Sprintf("├─────────────────────────────────────────────────────────┤\n"))
	body.WriteString(fmt.Sprintf("│ %-53s │\n", status))
	body.WriteString(fmt.Sprintf("└─────────────────────────────────────────────────────────┘"))
	return tea.NewView(body.String())
}

func truncate(s string, width int) string {
	if len([]rune(s)) <= width {
		return s
	}
	return string([]rune(s)[:width-1]) + "…"
}

func centerText(msg string, width, height int) string {
	if width <= 0 {
		width = len(msg) + 4
	}
	padding := max(0, (width-len(msg))/2)
	line := strings.Repeat(" ", padding) + msg
	if height <= 1 {
		return line
	}
	verticalPad := max(0, (height-1)/2)
	return strings.Join(append(make([]string, verticalPad), line), "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
