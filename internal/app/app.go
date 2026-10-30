package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"wordterm/internal/editor"
	"wordterm/internal/project"
	"wordterm/internal/settings"
	"wordterm/internal/storage"

	tea "charm.land/bubbletea/v2"
)

type Model struct {
	project         *project.Project
	editor          *editor.Editor
	settings        settings.Settings
	activePane      string
	width           int
	height          int
	showMessage     string
	originalRoot    string
	workingRoot     string
	selectedChapter int
	selectedScene   int
	unsavedChanges  bool
}

func NewModel() *Model {
	root, err := os.MkdirTemp("", "wordterm-project-")
	if err != nil {
		panic(err)
	}
	p := project.NewProject("Example", "Author")
	if err := project.SaveProject(p, root); err != nil {
		panic(err)
	}
	workRoot, err := storage.CreateWorkingCopy(root)
	if err != nil {
		panic(err)
	}
	loaded, err := project.LoadProject(workRoot)
	if err != nil {
		panic(err)
	}
	m := &Model{
		project:         loaded,
		settings:        settings.Default(),
		activePane:      "editor",
		width:           100,
		height:          30,
		originalRoot:    root,
		workingRoot:     workRoot,
		selectedChapter: 0,
		selectedScene:   0,
		unsavedChanges:  false,
	}
	if err := m.openSelectedScene(); err != nil {
		panic(err)
	}
	return m
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
		case "up":
			if m.activePane == "navigator" {
				m.moveNavigator(-1)
			} else {
				m.editor.MoveUp()
			}
			return m, nil
		case "down":
			if m.activePane == "navigator" {
				m.moveNavigator(1)
			} else {
				m.editor.MoveDown()
			}
			return m, nil
		case "left":
			if m.activePane == "navigator" {
				m.activePane = "editor"
			} else {
				m.editor.MoveLeft()
			}
			return m, nil
		case "right":
			if m.activePane == "navigator" {
				m.activePane = "editor"
			} else {
				m.editor.MoveRight()
			}
			return m, nil
		case "home":
			m.editor.Home()
			return m, nil
		case "end":
			m.editor.End()
			return m, nil
		case "pgup":
			m.editor.PageUp()
			return m, nil
		case "pgdown":
			m.editor.PageDown()
			return m, nil
		case "enter":
			if m.activePane == "navigator" {
				m.activePane = "editor"
				return m, nil
			}
			m.editor.InsertString("\n")
			m.unsavedChanges = true
			return m, nil
		case "backspace":
			m.editor.Backspace()
			m.unsavedChanges = true
			return m, nil
		case "delete":
			m.editor.Delete()
			m.unsavedChanges = true
			return m, nil
		case "s":
			if err := m.saveProject(); err != nil {
				m.showMessage = err.Error()
				return m, nil
			}
			m.showMessage = "Saved"
			m.unsavedChanges = false
			return m, nil
		case "r":
			if err := m.reloadProject(); err != nil {
				m.showMessage = err.Error()
				return m, nil
			}
			m.showMessage = "Reloaded"
			return m, nil
		case "n":
			if m.activePane == "navigator" {
				m.createSceneInCurrentChapter()
			}
			return m, nil
		default:
			if m.activePane == "editor" && len(msg.String()) == 1 && msg.String() != "" {
				m.editor.InsertString(msg.String())
				m.unsavedChanges = true
			}
		}
	}
	return m, nil
}

func (m *Model) View() tea.View {
	if m.width < 50 || m.height < 12 {
		return tea.NewView(centerText("Terminal Size too Small", m.width, m.height))
	}

	navWidth := minInt(maxInt(14, m.width/3), m.width-26)
	editorWidth := m.width - navWidth - 7
	bodyHeight := maxInt(4, m.height-6)
	navigatorLines := m.renderNavigatorLines()
	editorLines := m.renderEditorLines(editorWidth - 2)

	lines := make([]string, 0, bodyHeight+5)
	lines = append(lines, borderTop(m.width))
	for i := 0; i < bodyHeight; i++ {
		navText := " "
		if i < len(navigatorLines) {
			navText = navigatorLines[i]
		}
		editorText := " "
		if i < len(editorLines) {
			editorText = editorLines[i]
		}
		row := "│ " + padRight(navText, navWidth-2) + " │ " + padRight(editorText, editorWidth-2) + " │"
		lines = append(lines, row)
	}
	lines = append(lines, separator(m.width))
	lines = append(lines, "│ "+padRight(m.statusBar(), m.width-4)+" │")
	lines = append(lines, bottomBorder(m.width))
	return tea.NewView(strings.Join(lines, "\n"))
}

func (m *Model) renderNavigatorLines() []string {
	if m.project == nil {
		return []string{"Project"}
	}
	lines := []string{m.project.Name}
	for i, chapter := range m.project.Chapters {
		prefix := "├─ "
		if i == len(m.project.Chapters)-1 {
			prefix = "└─ "
		}
		if i == m.selectedChapter {
			prefix = "├─ "
		}
		lines = append(lines, prefix+chapter.Name)
		for j, scene := range chapter.Scenes {
			branch := "│  "
			if i == len(m.project.Chapters)-1 {
				branch = "   "
			}
			marker := "├─ "
			if j == len(chapter.Scenes)-1 {
				marker = "└─ "
			}
			if i == m.selectedChapter && j == m.selectedScene {
				marker = "├─ "
			}
			if i == m.selectedChapter {
				lines = append(lines, branch+marker+scene)
			}
		}
	}
	return lines
}

func (m *Model) renderEditorLines(width int) []string {
	if m.editor == nil {
		return []string{"Start writing..."}
	}
	text := m.editor.Text()
	if strings.TrimSpace(text) == "" {
		return []string{"Start writing..."}
	}
	return wrapText(text, width)
}

func (m *Model) statusBar() string {
	if m.editor == nil {
		return "Words: Scene: 0 | Chapter: 0 | Project: 0"
	}
	projectWords := countProjectWords(m.project, m.workingRoot)
	chapterWords := 0
	if m.selectedChapter >= 0 && m.selectedChapter < len(m.project.Chapters) {
		chapterWords = countChapterWords(m.project.Chapters[m.selectedChapter], m.workingRoot)
	}
	sceneWords := m.editor.WordCount()
	return fmt.Sprintf("Words: Scene: %d | Chapter: %d | Project: %d", sceneWords, chapterWords, projectWords)
}

func (m *Model) openSelectedScene() error {
	if m.project == nil || len(m.project.Chapters) == 0 {
		return nil
	}
	if m.selectedChapter >= len(m.project.Chapters) {
		m.selectedChapter = 0
	}
	if len(m.project.Chapters[m.selectedChapter].Scenes) == 0 {
		m.editor = editor.NewEditor()
		return nil
	}
	if m.selectedScene >= len(m.project.Chapters[m.selectedChapter].Scenes) {
		m.selectedScene = len(m.project.Chapters[m.selectedChapter].Scenes) - 1
	}
	chapter := m.project.Chapters[m.selectedChapter]
	scene := chapter.Scenes[m.selectedScene]
	path := filepath.Join(m.workingRoot, chapter.Name, scene+".txt")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	e := editor.NewEditor()
	if err == nil {
		e.InsertString(string(data))
	}
	m.editor = e
	m.project.LastOpenedScene = project.LastOpenedScene{Chapter: chapter.Name, Scene: scene}
	return nil
}

func (m *Model) reloadProject() error {
	if m.originalRoot == "" {
		return nil
	}
	workRoot, err := storage.CreateWorkingCopy(m.originalRoot)
	if err != nil {
		return err
	}
	loaded, err := project.LoadProject(workRoot)
	if err != nil {
		return err
	}
	m.project = loaded
	m.workingRoot = workRoot
	m.selectedChapter = 0
	m.selectedScene = 0
	m.activePane = "editor"
	return m.openSelectedScene()
}

func (m *Model) saveProject() error {
	if m.project == nil {
		return fmt.Errorf("no active project")
	}
	if m.editor == nil {
		return fmt.Errorf("no active editor")
	}
	if m.selectedChapter >= len(m.project.Chapters) || m.selectedScene >= len(m.project.Chapters[m.selectedChapter].Scenes) {
		return fmt.Errorf("no active scene selected")
	}
	chapter := m.project.Chapters[m.selectedChapter]
	scene := chapter.Scenes[m.selectedScene]
	filePath := filepath.Join(m.workingRoot, chapter.Name, scene+".txt")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filePath, []byte(m.editor.Text()), 0o644); err != nil {
		return err
	}
	m.project.LastOpenedScene = project.LastOpenedScene{Chapter: chapter.Name, Scene: scene}
	if err := project.SaveProject(m.project, m.workingRoot); err != nil {
		return err
	}
	if m.originalRoot != "" {
		if err := storage.SyncWorkingCopyToOriginal(m.workingRoot, m.originalRoot); err != nil {
			return err
		}
	}
	return nil
}

func (m *Model) moveNavigator(delta int) {
	if m.project == nil || len(m.project.Chapters) == 0 {
		return
	}
	if len(m.project.Chapters[m.selectedChapter].Scenes) > 0 {
		m.selectedScene += delta
		if m.selectedScene < 0 {
			m.selectedScene = 0
		}
		if m.selectedScene >= len(m.project.Chapters[m.selectedChapter].Scenes) {
			m.selectedScene = len(m.project.Chapters[m.selectedChapter].Scenes) - 1
		}
		_ = m.openSelectedScene()
		return
	}
	m.selectedChapter += delta
	if m.selectedChapter < 0 {
		m.selectedChapter = 0
	}
	if m.selectedChapter >= len(m.project.Chapters) {
		m.selectedChapter = len(m.project.Chapters) - 1
	}
}

func (m *Model) createSceneInCurrentChapter() {
	if m.project == nil || len(m.project.Chapters) == 0 {
		return
	}
	chapter := &m.project.Chapters[m.selectedChapter]
	name := fmt.Sprintf("Scene %d", len(chapter.Scenes)+1)
	chapter.Scenes = append(chapter.Scenes, name)
	m.selectedScene = len(chapter.Scenes) - 1
	if err := os.MkdirAll(filepath.Join(m.workingRoot, chapter.Name), 0o755); err != nil {
		m.showMessage = err.Error()
		return
	}
	if err := os.WriteFile(filepath.Join(m.workingRoot, chapter.Name, name+".txt"), []byte(""), 0o644); err != nil {
		m.showMessage = err.Error()
		return
	}
	m.showMessage = "Scene created"
	m.unsavedChanges = true
	_ = m.openSelectedScene()
}

func wrapText(s string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	if strings.TrimSpace(s) == "" {
		return []string{""}
	}
	lines := []string{}
	current := ""
	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
			continue
		}
		if len([]rune(current))+1 > width {
			lines = append(lines, current)
			current = string(ch)
			continue
		}
		current += string(ch)
	}
	if current != "" || len(lines) == 0 {
		lines = append(lines, current)
	}
	return lines
}

func borderTop(width int) string {
	inside := maxInt(0, width-2)
	return "┌" + strings.Repeat("─", inside) + "┐"
}

func separator(width int) string {
	inside := maxInt(0, width-2)
	return "├" + strings.Repeat("─", inside) + "┤"
}

func bottomBorder(width int) string {
	inside := maxInt(0, width-2)
	return "└" + strings.Repeat("─", inside) + "┘"
}

func padRight(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(runes))
}

func centerText(msg string, width, height int) string {
	if width <= 0 {
		width = len(msg) + 4
	}
	padding := maxInt(0, (width-len([]rune(msg)))/2)
	line := strings.Repeat(" ", padding) + msg
	if height <= 1 {
		return line
	}
	verticalPad := maxInt(0, (height-1)/2)
	result := make([]string, 0, verticalPad+1)
	for i := 0; i < verticalPad; i++ {
		result = append(result, "")
	}
	result = append(result, line)
	return strings.Join(result, "\n")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func countProjectWords(p *project.Project, root string) int {
	if p == nil {
		return 0
	}
	count := 0
	for _, chapter := range p.Chapters {
		count += countChapterWords(chapter, root)
	}
	return count
}

func countChapterWords(chapter project.Chapter, root string) int {
	count := 0
	for _, scene := range chapter.Scenes {
		path := filepath.Join(root, chapter.Name, scene+".txt")
		if data, err := os.ReadFile(path); err == nil {
			count += project.CountWords(string(data))
		}
	}
	return count
}
