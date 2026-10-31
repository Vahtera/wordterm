package app

import (
	"os"
	"testing"

	"wordterm/internal/project"
	"wordterm/internal/settings"
	"wordterm/internal/storage"
)

func TestWelcomeProject(t *testing.T) {
	tmp := t.TempDir()
	for _, k := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(k, tmp)
	}
	m, dir, err := NewWelcome(settings.Default(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	if !m.welcome || m.cur != (project.SceneRef{Chapter: "Help", Scene: "Keys"}) {
		t.Fatalf("welcome=%v cur=%+v", m.welcome, m.cur)
	}
	if m.ed.Text() != m.helpText() {
		t.Fatal("scene text must equal the F1 help text")
	}
	if recs, _ := storage.FindRecoverable(); len(recs) != 0 {
		t.Fatal("the help project must never be offered for recovery")
	}
}
