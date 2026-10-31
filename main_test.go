package main

import (
	"path/filepath"
	"testing"

	"github.com/Vahtera/wordterm/internal/project"
	"github.com/Vahtera/wordterm/internal/settings"
)

func TestResolveProject(t *testing.T) {
	base := t.TempDir()
	if _, err := project.Create(base, "Test", "a", ""); err != nil {
		t.Fatal(err)
	}
	cfg := settings.Default()
	cfg.DefaultSavePath = base

	got, err := resolveProject("Test", cfg) // bare name: found in the default save path
	if err != nil || got != filepath.Join(base, "Test") {
		t.Fatalf("got %q, %v", got, err)
	}
	explicit := filepath.Join(base, "Elsewhere")
	if got, err := resolveProject(explicit, cfg); err != nil || got != explicit {
		t.Fatalf("explicit paths must be used as given: %q, %v", got, err)
	}
	if _, err := resolveProject("Missing", cfg); err == nil {
		t.Fatal("unknown bare name should fail")
	}
}
