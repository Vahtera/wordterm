// Package settings holds user preferences (TOML) and the key bindings.
package settings

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/BurntSushi/toml"

	"wordterm/internal/storage"
)

// Action is a rebindable command. NavOnly actions only fire while the
// Navigator has focus, so they may use printable keys such as "+".
type Action struct {
	ID, Label, Default string
	NavOnly            bool
}

var Actions = []Action{
	{"help", "Help overlay", "f1", false},
	{"switch_pane", "Switch pane", "f3", false},
	{"save", "Save project", "f10", false},
	{"reload", "Reload project", "f9", false},
	{"quit", "Quit", "ctrl+q", false},
	{"rename", "Rename item", "f2", true},
	{"move_up", "Move item up", "f5", true},
	{"move_down", "Move item down", "f6", true},
	{"add", "Add item", "+", true},
	{"delete", "Delete item", "-", true},
}

// Keys that always keep their editing/navigation meaning.
var reserved = map[string]bool{"up": true, "down": true, "left": true, "right": true, "enter": true,
	"backspace": true, "delete": true, "home": true, "end": true, "pgup": true, "pgdown": true,
	"tab": true, "shift+tab": true, "ctrl+z": true, "ctrl+y": true, "ctrl+c": true}

type Settings struct {
	EditorWidth      int               `toml:"editor_width"`      // 0 = default
	WriteSize        int               `toml:"write_size"`        // 0 = same as editor width
	WordWrap         bool              `toml:"word_wrap"`
	Autosave         bool              `toml:"autosave"`
	AutosaveInterval int               `toml:"autosave_interval"` // seconds, minimum 5
	Keys             map[string]string `toml:"keys"`
}

func find(id string) (Action, bool) {
	for _, a := range Actions {
		if a.ID == id {
			return a, true
		}
	}
	return Action{}, false
}

func NavOnly(id string) bool { a, ok := find(id); return ok && a.NavOnly }

func Default() Settings {
	s := Settings{WordWrap: true, AutosaveInterval: 60, Keys: map[string]string{}}
	for _, a := range Actions {
		s.Keys[a.ID] = a.Default
	}
	return s
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "wordterm", "config.toml"), nil
}

// Load reads the config file. It never fails: a missing file gives defaults,
// and an unreadable one is moved aside (config.toml.broken) so it is not overwritten.
func Load() (Settings, string) {
	s := Default()
	path, err := Path()
	if err != nil {
		return s, ""
	}
	if _, err := os.Stat(path); err != nil {
		return s, ""
	}
	in := Settings{WordWrap: true, AutosaveInterval: 60}
	if _, err := toml.DecodeFile(path, &in); err != nil {
		os.Rename(path, path+".broken")
		return s, "Invalid config moved to config.toml.broken; using defaults"
	}
	s.EditorWidth, s.WriteSize = max(in.EditorWidth, 0), max(in.WriteSize, 0)
	s.WordWrap, s.Autosave = in.WordWrap, in.Autosave
	s.AutosaveInterval = max(in.AutosaveInterval, 5)
	note := ""
	for _, a := range Actions {
		if k, ok := in.Keys[a.ID]; ok && k != a.Default {
			if err := s.SetKey(a.ID, k); err != nil {
				note = "Ignored key binding for " + a.Label + ": " + err.Error()
			}
		}
	}
	return s, note
}

func (s Settings) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(s); err != nil {
		return err
	}
	return storage.WriteFileAtomic(path, buf.Bytes(), 0o644)
}

// ActionFor returns the action bound to key, or "".
func (s Settings) ActionFor(key string) string {
	for id, k := range s.Keys {
		if k == key {
			return id
		}
	}
	return ""
}

func isPrintable(key string) bool { return key == "space" || utf8.RuneCountInString(key) == 1 }

// SetKey binds key to the action, rejecting Esc, reserved keys, conflicts and
// (for global actions) plain printable keys that would block typing.
func (s *Settings) SetKey(id, key string) error {
	a, ok := find(id)
	switch {
	case !ok:
		return fmt.Errorf("unknown action %q", id)
	case key == "esc":
		return fmt.Errorf("Esc is reserved for the main menu")
	case key == "" || reserved[key]:
		return fmt.Errorf("%q cannot be used: it is needed for editing and navigation", key)
	case !a.NavOnly && isPrintable(key):
		return fmt.Errorf("%q would block typing; use a function key or a Ctrl/Alt combination", key)
	}
	for _, o := range Actions {
		if o.ID != id && s.Keys[o.ID] == key {
			return fmt.Errorf("%s is already used by %q", key, o.Label)
		}
	}
	s.Keys[id] = key
	return nil
}
