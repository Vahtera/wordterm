package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/Vahtera/wordterm/internal/storage"
)

const FormatVersion = 1

// SceneRef identifies a scene: its chapter name and the scene's File key.
type SceneRef struct {
	Chapter string `json:"chapter"`
	Scene   string `json:"scene"`
}

// Scene is one scene. Name is what the user sees; File is the .txt file stem
// inside the chapter folder (the scene's unique key within its chapter). They
// differ only when a move had to avoid a name clash, e.g. Name "Intro",
// File "Intro_2". In JSON a plain string means Name == File; otherwise an object.
type Scene struct {
	Name string
	File string
}

type sceneJSON struct {
	Name string `json:"name"`
	File string `json:"file"`
}

func (s Scene) MarshalJSON() ([]byte, error) {
	if s.File == "" || s.File == s.Name {
		return json.Marshal(s.Name)
	}
	return json.Marshal(sceneJSON{s.Name, s.File})
}

func (s *Scene) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		if err := json.Unmarshal(b, &s.Name); err != nil {
			return err
		}
		s.File = s.Name
		return nil
	}
	var o sceneJSON
	if err := json.Unmarshal(b, &o); err != nil {
		return err
	}
	s.Name, s.File = o.Name, o.File
	if s.File == "" {
		s.File = o.Name
	}
	return nil
}

type Chapter struct {
	Name   string  `json:"name"`
	Scenes []Scene `json:"scenes"`
}

type Meta struct {
	Version    int       `json:"format_version"`
	Name       string    `json:"name"`
	Author     string    `json:"author"`
	AuthorInfo string    `json:"author_info"`
	LastOpened *SceneRef `json:"last_opened_scene,omitempty"`
	Chapters   []Chapter `json:"chapters"`
}

// Validate checks names, uniqueness (case-insensitively, for Windows and
// macOS filesystems) and the format version.
func (m *Meta) Validate() error {
	if m.Version < 1 || m.Version > FormatVersion {
		return fmt.Errorf("unsupported project format version %d (supported: 1-%d)", m.Version, FormatVersion)
	}
	if err := ValidateName(m.Name); err != nil {
		return fmt.Errorf("project name: %w", err)
	}
	chapters := map[string]bool{}
	for _, c := range m.Chapters {
		if err := ValidateName(c.Name); err != nil {
			return fmt.Errorf("chapter: %w", err)
		}
		if k := strings.ToLower(c.Name); chapters[k] {
			return fmt.Errorf("duplicate chapter %q", c.Name)
		} else {
			chapters[k] = true
		}
		files := map[string]bool{}
		for _, s := range c.Scenes {
			if err := ValidateName(s.Name); err != nil {
				return fmt.Errorf("scene in %q: %w", c.Name, err)
			}
			if err := ValidateName(s.File); err != nil {
				return fmt.Errorf("scene file in %q: %w", c.Name, err)
			}
			if k := strings.ToLower(s.File); files[k] {
				return fmt.Errorf("duplicate scene file %q in chapter %q", s.File, c.Name)
			} else {
				files[k] = true
			}
		}
	}
	return nil
}

func (m *Meta) HasScene(r SceneRef) bool {
	for _, c := range m.Chapters {
		if c.Name == r.Chapter {
			for _, s := range c.Scenes {
				if s.File == r.Scene {
					return true
				}
			}
		}
	}
	return false
}

// Resolve returns the last opened scene, or the first scene if that no
// longer exists. ok is false when the project has no scenes.
func (m *Meta) Resolve() (ref SceneRef, ok bool) {
	if m.LastOpened != nil && m.HasScene(*m.LastOpened) {
		return *m.LastOpened, true
	}
	for _, c := range m.Chapters {
		if len(c.Scenes) > 0 {
			return SceneRef{c.Name, c.Scenes[0].File}, true
		}
	}
	return SceneRef{}, false
}

// Project is a loaded project rooted at Root (normally the working copy).
type Project struct {
	Root     string
	MetaPath string
	Meta     Meta
}

func SceneFile(root string, r SceneRef) string {
	return filepath.Join(root, r.Chapter, r.Scene+".txt")
}

func findMeta(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	var found []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			found = append(found, e.Name())
		}
	}
	switch len(found) {
	case 0:
		return "", errors.New("no project .json file found")
	case 1:
		return filepath.Join(root, found[0]), nil
	}
	return "", fmt.Errorf("several .json files found (%s); expected exactly one", strings.Join(found, ", "))
}

// IsProjectDir reports whether dir contains exactly one project .json file.
func IsProjectDir(dir string) bool {
	_, err := findMeta(dir)
	return err == nil
}

// Load reads and validates the project in root and checks that every
// chapter folder and scene file exists. Nothing is modified.
func Load(root string) (*Project, error) {
	path, err := findMeta(root)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	p := &Project{Root: root, MetaPath: path}
	if err := json.Unmarshal(data, &p.Meta); err != nil {
		return nil, fmt.Errorf("%s: invalid JSON: %w", filepath.Base(path), err)
	}
	if err := p.Meta.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	var missing []error
	for _, c := range p.Meta.Chapters {
		if st, err := os.Stat(filepath.Join(root, c.Name)); err != nil || !st.IsDir() {
			missing = append(missing, fmt.Errorf("chapter folder missing: %s", c.Name))
			continue
		}
		for _, s := range c.Scenes {
			if _, err := os.Stat(SceneFile(root, SceneRef{c.Name, s.File})); err != nil {
				missing = append(missing, fmt.Errorf("scene file missing: %s/%s.txt", c.Name, s.File))
			}
		}
	}
	if err := errors.Join(missing...); err != nil {
		return nil, err
	}
	return p, nil
}

// WriteMeta atomically writes the metadata file inside p.Root.
func (p *Project) WriteMeta() error {
	data, err := json.MarshalIndent(&p.Meta, "", "  ")
	if err != nil {
		return err
	}
	return storage.WriteFileAtomic(p.MetaPath, append(data, '\n'), 0o644)
}

// ReadScene returns the text of a scene (must be valid UTF-8).
func (p *Project) ReadScene(r SceneRef) (string, error) {
	data, err := os.ReadFile(SceneFile(p.Root, r))
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("%s/%s.txt is not valid UTF-8", r.Chapter, r.Scene)
	}
	return string(data), nil
}

// CountWords counts whitespace-separated words (Unicode whitespace).
func CountWords(s string) int { return len(strings.Fields(s)) }

// Counts returns word counts for the scene cur, its chapter and the project.
func (p *Project) Counts(cur SceneRef) (scene, chapter, total int, err error) {
	for _, c := range p.Meta.Chapters {
		cw := 0
		for _, s := range c.Scenes {
			txt, err := p.ReadScene(SceneRef{c.Name, s.File})
			if err != nil {
				return 0, 0, 0, err
			}
			n := CountWords(txt)
			cw += n
			total += n
			if c.Name == cur.Chapter && s.File == cur.Scene {
				scene = n
			}
		}
		if c.Name == cur.Chapter {
			chapter = cw
		}
	}
	return
}

// Create makes a new project directory parent/name with one chapter and
// one empty scene. It refuses to touch an existing directory.
func Create(parent, name, author, authorInfo string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	if st, err := os.Stat(parent); err != nil || !st.IsDir() {
		return "", fmt.Errorf("destination %q is not an existing directory", parent)
	}
	dir := filepath.Join(parent, name)
	if err := os.Mkdir(dir, 0o755); err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("%s already exists; not overwriting", dir)
		}
		return "", err
	}
	fail := func(err error) (string, error) { os.RemoveAll(dir); return "", err }
	first := SceneRef{"Chapter1", "Scene1"}
	p := &Project{Root: dir, MetaPath: filepath.Join(dir, name+".json"), Meta: Meta{
		Version: FormatVersion, Name: name, Author: author, AuthorInfo: authorInfo,
		LastOpened: &first, Chapters: []Chapter{{Name: first.Chapter, Scenes: []Scene{{Name: first.Scene, File: first.Scene}}}},
	}}
	if err := os.Mkdir(filepath.Join(dir, first.Chapter), 0o755); err != nil {
		return fail(err)
	}
	if err := storage.WriteFileAtomic(SceneFile(dir, first), nil, 0o644); err != nil {
		return fail(err)
	}
	if err := p.WriteMeta(); err != nil {
		return fail(err)
	}
	return dir, nil
}
