package project

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

// Chapter represents a writing chapter and preserves ordering in JSON metadata.
type Chapter struct {
	Name   string   `json:"name"`
	Scenes []string `json:"scenes"`
	Path   string   `json:"path,omitempty"`
}

// LastOpenedScene stores the last scene selection for reopening the project.
type LastOpenedScene struct {
	Chapter string `json:"chapter"`
	Scene   string `json:"scene"`
}

// Project is the application-level project representation.
type Project struct {
	Name            string          `json:"name"`
	Author          string          `json:"author"`
	AuthorInfo      string          `json:"author_info,omitempty"`
	FormatVersion   int             `json:"format_version"`
	LastOpenedScene LastOpenedScene `json:"last_opened_scene"`
	Chapters        []Chapter       `json:"chapters"`
	Root            string          `json:"-"`
	WorkingCopyRoot string          `json:"-"`
}

// NewProject creates the initial project skeleton.
func NewProject(name, author string) *Project {
	chapterName := "Chapter 1"
	if name == "" {
		name = "New Project"
	}
	if author == "" {
		author = "Anonymous"
	}
	return &Project{
		Name:          name,
		Author:        author,
		FormatVersion: 1,
		Chapters: []Chapter{
			{Name: chapterName, Scenes: []string{"Scene 1"}, Path: filepath.Join("Chapter 1")},
		},
		LastOpenedScene: LastOpenedScene{Chapter: chapterName, Scene: "Scene 1"},
	}
}

// CountWords returns the number of whitespace-delimited words in the provided text.
func CountWords(text string) int {
	return len(strings.FieldsFunc(text, func(r rune) bool {
		return unicode.IsSpace(r)
	}))
}

// ScenePath resolves the path to a scene file under the working project root.
func (p *Project) ScenePath(chapterName, sceneName string) string {
	root := p.WorkingCopyRoot
	if root == "" {
		root = p.Root
	}
	if root == "" {
		return filepath.Join(chapterName, sceneName+".txt")
	}
	return filepath.Join(root, chapterName, sceneName+".txt")
}

// ChapterPath resolves the chapter directory inside the current project root.
func (p *Project) ChapterPath(chapterName string) string {
	root := p.WorkingCopyRoot
	if root == "" {
		root = p.Root
	}
	if root == "" {
		return chapterName
	}
	return filepath.Join(root, chapterName)
}

// EnsureLastOpenedScene makes sure the saved selection points to an existing scene or falls back to the first available one.
func (p *Project) EnsureLastOpenedScene() {
	if len(p.Chapters) == 0 {
		return
	}
	for _, chapter := range p.Chapters {
		if chapter.Name == p.LastOpenedScene.Chapter {
			for _, scene := range chapter.Scenes {
				if scene == p.LastOpenedScene.Scene {
					return
				}
			}
		}
	}
	firstChapter := p.Chapters[0]
	p.LastOpenedScene = LastOpenedScene{Chapter: firstChapter.Name, Scene: firstChapter.Scenes[0]}
}

// ProjectSummary returns a short, human-readable summary for the UI.
func (p *Project) ProjectSummary() string {
	if p == nil {
		return "No project"
	}
	if p.Name == "" {
		return "Unnamed project"
	}
	return fmt.Sprintf("%s (%s)", p.Name, p.Author)
}
