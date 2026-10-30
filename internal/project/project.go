package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const MetadataFileName = "project.json"

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

// LoadProject reads metadata and validates the project from disk.
func LoadProject(projectDir string) (*Project, error) {
	if projectDir == "" {
		return nil, fmt.Errorf("project directory is required")
	}
	metaPath := filepath.Join(projectDir, MetadataFileName)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("read project metadata: %w", err)
	}

	p := &Project{Root: projectDir, WorkingCopyRoot: projectDir}
	if err := json.Unmarshal(data, p); err != nil {
		return nil, fmt.Errorf("parse project metadata: %w", err)
	}
	if p.Name == "" {
		return nil, fmt.Errorf("project name is required")
	}
	if len(p.Chapters) == 0 {
		return nil, fmt.Errorf("project contains no chapters")
	}
	p.Root = projectDir
	p.WorkingCopyRoot = projectDir
	for i := range p.Chapters {
		if p.Chapters[i].Name == "" {
			return nil, fmt.Errorf("chapter name is required")
		}
		p.Chapters[i].Path = filepath.Join(projectDir, p.Chapters[i].Name)
		for _, sceneName := range p.Chapters[i].Scenes {
			if sceneName == "" {
				continue
			}
			scenePath := filepath.Join(projectDir, p.Chapters[i].Name, sceneName+".txt")
			if _, err := os.Stat(scenePath); err != nil {
				return nil, fmt.Errorf("missing scene file %q: %w", sceneName, err)
			}
		}
	}
	p.EnsureLastOpenedScene()
	return p, nil
}

// Save writes the project metadata and scene files to the configured root directory.
func (p *Project) Save() error {
	if p == nil {
		return fmt.Errorf("project is nil")
	}
	if p.Root == "" {
		return fmt.Errorf("project root is not set")
	}
	return SaveProject(p, p.Root)
}

// SaveProject writes a complete project snapshot to the target directory.
func SaveProject(p *Project, dir string) error {
	if p == nil {
		return fmt.Errorf("project is nil")
	}
	if dir == "" {
		return fmt.Errorf("directory is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create project directory: %w", err)
	}
	p.Root = dir
	p.WorkingCopyRoot = dir
	if len(p.Chapters) == 0 {
		p.Chapters = []Chapter{{Name: "Chapter 1", Scenes: []string{"Scene 1"}, Path: filepath.Join(dir, "Chapter 1")}}
	}
	for i := range p.Chapters {
		if p.Chapters[i].Name == "" {
			return fmt.Errorf("chapter name is required")
		}
		chapterDir := filepath.Join(dir, p.Chapters[i].Name)
		if err := os.MkdirAll(chapterDir, 0o755); err != nil {
			return fmt.Errorf("create chapter directory: %w", err)
		}
		p.Chapters[i].Path = chapterDir
		for _, sceneName := range p.Chapters[i].Scenes {
			if sceneName == "" {
				continue
			}
			scenePath := filepath.Join(chapterDir, sceneName+".txt")
			if _, err := os.Stat(scenePath); err != nil {
				if err := os.WriteFile(scenePath, []byte(""), 0o644); err != nil {
					return fmt.Errorf("create scene file %q: %w", sceneName, err)
				}
			}
		}
	}
	p.EnsureLastOpenedScene()
	metaData, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal project metadata: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, MetadataFileName), metaData, 0o644); err != nil {
		return fmt.Errorf("write project metadata: %w", err)
	}
	return nil
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
