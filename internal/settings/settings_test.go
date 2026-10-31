package settings

import "testing"

func TestDefaults(t *testing.T) {
	s := Default()
	seen := map[string]string{}
	for id, k := range s.Keys {
		if o, dup := seen[k]; dup {
			t.Fatalf("%s and %s share %q", id, o, k)
		}
		seen[k] = id
	}
	if s.Autosave || !s.WordWrap {
		t.Fatal("autosave must default to off and word wrap to on")
	}
}

func TestSetKeyRules(t *testing.T) {
	s := Default()
	for _, c := range []struct{ id, key string }{{"save", "f1"}, {"save", "esc"}, {"save", "a"}, {"add", "up"}} {
		if s.SetKey(c.id, c.key) == nil {
			t.Errorf("SetKey(%s,%s) should fail", c.id, c.key)
		}
	}
	if err := s.SetKey("save", "ctrl+s"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetKey("add", "="); err != nil {
		t.Fatalf("printable key for a navigator-only action should be allowed: %v", err)
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	for _, k := range []string{"XDG_CONFIG_HOME", "APPDATA", "HOME"} {
		t.Setenv(k, dir)
	}
	s := Default()
	s.Autosave, s.EditorWidth = true, 50
	if err := s.SetKey("save", "ctrl+s"); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	got, note := Load()
	if note != "" || !got.Autosave || got.EditorWidth != 50 || got.Keys["save"] != "ctrl+s" || got.Keys["help"] != "f1" {
		t.Fatalf("note=%q got=%+v", note, got)
	}
}
