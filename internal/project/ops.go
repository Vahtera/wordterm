package project

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"wordterm/internal/storage"
)

// All operations here act on p.Root (the working copy). Each one changes the
// filesystem and metadata together and rolls both back if anything fails.

func (m *Meta) ChapterIndex(name string) int {
	for i, c := range m.Chapters {
		if c.Name == name {
			return i
		}
	}
	return -1
}

// SceneIndex finds a scene by its File key.
func (m *Meta) SceneIndex(chapter, file string) int {
	if ci := m.ChapterIndex(chapter); ci >= 0 {
		return slices.IndexFunc(m.Chapters[ci].Scenes, func(s Scene) bool { return s.File == file })
	}
	return -1
}

// SceneName returns the display name of the scene with the given File key.
func (m *Meta) SceneName(chapter, file string) string {
	if si := m.SceneIndex(chapter, file); si >= 0 {
		return m.Chapters[m.ChapterIndex(chapter)].Scenes[si].Name
	}
	return file
}

func (p *Project) snapshot() func() {
	chapters := make([]Chapter, len(p.Meta.Chapters))
	for i, c := range p.Meta.Chapters {
		chapters[i] = Chapter{c.Name, slices.Clone(c.Scenes)}
	}
	var last *SceneRef
	if p.Meta.LastOpened != nil {
		v := *p.Meta.LastOpened
		last = &v
	}
	name := p.Meta.Name
	return func() { p.Meta.Chapters, p.Meta.LastOpened, p.Meta.Name = chapters, last, name }
}

func (p *Project) chapterDir(name string) string { return filepath.Join(p.Root, name) }

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func (p *Project) chapterNameFree(name string, except int) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	for i, c := range p.Meta.Chapters {
		if i != except && strings.EqualFold(c.Name, name) {
			return fmt.Errorf("chapter %q already exists", c.Name)
		}
	}
	return nil
}

// sceneNameFree checks a new display name; display names stay unique per
// chapter when adding or renaming (only moves may create look-alikes).
func (p *Project) sceneNameFree(ci int, name string, except int) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	for i, s := range p.Meta.Chapters[ci].Scenes {
		if i != except && strings.EqualFold(s.Name, name) {
			return fmt.Errorf("scene %q already exists in %q", s.Name, p.Meta.Chapters[ci].Name)
		}
	}
	return nil
}

// freeFile returns a file stem for base that is unused in chapter ci, adding
// _2, _3, ... on a clash. self is the index of the scene being renamed (or -1).
func (p *Project) freeFile(ci int, base string, self int) string {
	c := p.Meta.Chapters[ci]
	for n := 1; ; n++ {
		cand := base
		if n > 1 {
			cand = fmt.Sprintf("%s_%d", base, n)
		}
		taken := false
		for i, s := range c.Scenes {
			if i != self && strings.EqualFold(s.File, cand) {
				taken = true
			}
		}
		if !taken && exists(SceneFile(p.Root, SceneRef{c.Name, cand})) &&
			!(self >= 0 && strings.EqualFold(c.Scenes[self].File, cand)) {
			taken = true
		}
		if !taken {
			return cand
		}
	}
}

func (p *Project) RenameProject(name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	restore := p.snapshot()
	p.Meta.Name = name
	if err := p.WriteMeta(); err != nil {
		restore()
		return err
	}
	return nil
}

func (p *Project) AddChapter(name string) error {
	if err := p.chapterNameFree(name, -1); err != nil {
		return err
	}
	dir := p.chapterDir(name)
	if err := os.Mkdir(dir, 0o755); err != nil {
		return err
	}
	restore := p.snapshot()
	p.Meta.Chapters = append(p.Meta.Chapters, Chapter{Name: name})
	if err := p.WriteMeta(); err != nil {
		restore()
		os.Remove(dir)
		return err
	}
	return nil
}

// AddScene creates an empty scene at index at (or at the end if at is out of range).
func (p *Project) AddScene(chapter string, at int, name string) (SceneRef, error) {
	ci := p.Meta.ChapterIndex(chapter)
	if ci < 0 {
		return SceneRef{}, fmt.Errorf("no chapter %q", chapter)
	}
	if err := p.sceneNameFree(ci, name, -1); err != nil {
		return SceneRef{}, err
	}
	ref := SceneRef{chapter, p.freeFile(ci, name, -1)}
	path := SceneFile(p.Root, ref)
	if err := storage.WriteFileAtomic(path, nil, 0o644); err != nil {
		return SceneRef{}, err
	}
	restore := p.snapshot()
	sc := p.Meta.Chapters[ci].Scenes
	if at < 0 || at > len(sc) {
		at = len(sc)
	}
	p.Meta.Chapters[ci].Scenes = slices.Insert(sc, at, Scene{Name: name, File: ref.Scene})
	if err := p.WriteMeta(); err != nil {
		restore()
		os.Remove(path)
		return SceneRef{}, err
	}
	return ref, nil
}

func (p *Project) RenameChapter(old, name string) error {
	ci := p.Meta.ChapterIndex(old)
	if ci < 0 {
		return fmt.Errorf("no chapter %q", old)
	}
	if old == name {
		return nil
	}
	if err := p.chapterNameFree(name, ci); err != nil {
		return err
	}
	from, to := p.chapterDir(old), p.chapterDir(name)
	if !strings.EqualFold(old, name) && exists(to) {
		return fmt.Errorf("%s already exists", to)
	}
	if err := os.Rename(from, to); err != nil {
		return err
	}
	restore := p.snapshot()
	p.Meta.Chapters[ci].Name = name
	if l := p.Meta.LastOpened; l != nil && l.Chapter == old {
		l.Chapter = name
	}
	if err := p.WriteMeta(); err != nil {
		restore()
		os.Rename(to, from)
		return err
	}
	return nil
}

// RenameScene renames the scene with File key file; its file is renamed to
// match (with a _2, _3... suffix if that stem is taken).
func (p *Project) RenameScene(chapter, file, name string) (SceneRef, error) {
	ci, si := p.Meta.ChapterIndex(chapter), p.Meta.SceneIndex(chapter, file)
	if si < 0 {
		return SceneRef{}, fmt.Errorf("no scene %q in %q", file, chapter)
	}
	old := SceneRef{chapter, file}
	sc := p.Meta.Chapters[ci].Scenes[si]
	if sc.Name == name {
		return old, nil
	}
	if err := p.sceneNameFree(ci, name, si); err != nil {
		return old, err
	}
	nw := SceneRef{chapter, p.freeFile(ci, name, si)}
	if nw != old {
		if err := os.Rename(SceneFile(p.Root, old), SceneFile(p.Root, nw)); err != nil {
			return old, err
		}
	}
	restore := p.snapshot()
	p.Meta.Chapters[ci].Scenes[si] = Scene{Name: name, File: nw.Scene}
	if l := p.Meta.LastOpened; l != nil && *l == old {
		*l = nw
	}
	if err := p.WriteMeta(); err != nil {
		restore()
		if nw != old {
			os.Rename(SceneFile(p.Root, nw), SceneFile(p.Root, old))
		}
		return old, err
	}
	return nw, nil
}

func (p *Project) DeleteScene(chapter, name string) error {
	ci, si := p.Meta.ChapterIndex(chapter), p.Meta.SceneIndex(chapter, name)
	if si < 0 {
		return fmt.Errorf("no scene %q in %q", name, chapter)
	}
	path := SceneFile(p.Root, SceneRef{chapter, name})
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	restore := p.snapshot()
	p.Meta.Chapters[ci].Scenes = slices.Delete(p.Meta.Chapters[ci].Scenes, si, si+1)
	if l := p.Meta.LastOpened; l != nil && l.Chapter == chapter && l.Scene == name {
		p.Meta.LastOpened = nil
	}
	if err := p.WriteMeta(); err != nil {
		restore()
		storage.WriteFileAtomic(path, data, 0o644)
		return err
	}
	return nil
}

// DeleteChapter only deletes chapters that contain no scenes.
func (p *Project) DeleteChapter(name string) error {
	ci := p.Meta.ChapterIndex(name)
	if ci < 0 {
		return fmt.Errorf("no chapter %q", name)
	}
	if len(p.Meta.Chapters[ci].Scenes) > 0 {
		return fmt.Errorf("chapter %q is not empty: delete its scenes first", name)
	}
	dir := p.chapterDir(name)
	if err := os.Remove(dir); err != nil { // fails if the folder holds anything
		return err
	}
	restore := p.snapshot()
	p.Meta.Chapters = slices.Delete(p.Meta.Chapters, ci, ci+1)
	if err := p.WriteMeta(); err != nil {
		restore()
		os.Mkdir(dir, 0o755)
		return err
	}
	return nil
}

// MoveChapter moves chapter ci by d (-1 up, +1 down) and returns its new index.
func (p *Project) MoveChapter(ci, d int) (int, error) {
	j := ci + d
	if j < 0 || j >= len(p.Meta.Chapters) {
		return ci, nil
	}
	restore := p.snapshot()
	c := p.Meta.Chapters
	c[ci], c[j] = c[j], c[ci]
	if err := p.WriteMeta(); err != nil {
		restore()
		return ci, err
	}
	return j, nil
}

// MoveScene moves a scene by d within its chapter. At a chapter edge it moves
// the file into the neighbouring chapter (last scene going up, first going
// down). It returns the scene's new position; unchanged position means no-op.
func (p *Project) MoveScene(ci, si, d int) (int, int, error) {
	c := &p.Meta.Chapters[ci]
	if j := si + d; j >= 0 && j < len(c.Scenes) {
		restore := p.snapshot()
		c.Scenes[si], c.Scenes[j] = c.Scenes[j], c.Scenes[si]
		if err := p.WriteMeta(); err != nil {
			restore()
			return ci, si, err
		}
		return ci, j, nil
	}
	ti := ci + d
	if ti < 0 || ti >= len(p.Meta.Chapters) {
		return ci, si, nil
	}
	target, sc := &p.Meta.Chapters[ti], c.Scenes[si]
	// A clash in the target chapter is allowed: the file gets a suffix
	// (Intro_2.txt) while the scene keeps its name in the metadata.
	newFile := p.freeFile(ti, sc.Name, -1)
	oldRef, newRef := SceneRef{c.Name, sc.File}, SceneRef{target.Name, newFile}
	if err := os.Rename(SceneFile(p.Root, oldRef), SceneFile(p.Root, newRef)); err != nil {
		return ci, si, err
	}
	restore := p.snapshot()
	c.Scenes = slices.Delete(c.Scenes, si, si+1)
	moved := Scene{Name: sc.Name, File: newFile}
	nsi := 0
	if d < 0 {
		target.Scenes = append(target.Scenes, moved)
		nsi = len(target.Scenes) - 1
	} else {
		target.Scenes = slices.Insert(target.Scenes, 0, moved)
	}
	if l := p.Meta.LastOpened; l != nil && *l == oldRef {
		*l = newRef
	}
	if err := p.WriteMeta(); err != nil {
		restore()
		os.Rename(SceneFile(p.Root, newRef), SceneFile(p.Root, oldRef))
		return ci, si, err
	}
	return ti, nsi, nil
}
