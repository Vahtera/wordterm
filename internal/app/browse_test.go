package app

import (
	"os"
	"path/filepath"
	"testing"

	"wordterm/internal/project"
)

func TestBrowseListsFoldersAndMarksProjects(t *testing.T) {
	base := t.TempDir()
	if _, err := project.Create(base, "Novel", "a", ""); err != nil {
		t.Fatal(err)
	}
	os.Mkdir(filepath.Join(base, "plain"), 0o755)
	os.Mkdir(filepath.Join(base, ".hidden"), 0o755)
	os.WriteFile(filepath.Join(base, "file.txt"), nil, 0o644)

	b := &browseState{}
	if err := b.cd(base, ""); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, e := range b.entries {
		got[e.name] = e.project
	}
	if p, ok := got["Novel"]; !ok || !p {
		t.Fatalf("Novel should be listed as a project: %v", got)
	}
	if p, ok := got["plain"]; !ok || p {
		t.Fatalf("plain should be listed as an ordinary folder: %v", got)
	}
	if _, ok := got[".hidden"]; ok {
		t.Fatal("hidden folders must not be listed")
	}
	if _, ok := got["file.txt"]; ok {
		t.Fatal("files must not be listed")
	}
	// going up from a folder selects the folder we came from
	if err := b.cd(filepath.Join(base, "plain"), ""); err != nil {
		t.Fatal(err)
	}
	if err := b.cd(base, "plain"); err != nil {
		t.Fatal(err)
	}
	if b.entries[b.idx].name != "plain" {
		t.Fatalf("selected %q", b.entries[b.idx].name)
	}
}
