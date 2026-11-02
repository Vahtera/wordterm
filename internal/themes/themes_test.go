package themes

import (
	"os"
	"strings"
	"testing"

	"github.com/Vahtera/wordterm/internal/settings"
	"github.com/Vahtera/wordterm/internal/ui"
)

func isolate(t *testing.T) {
	dir := t.TempDir()
	for _, k := range []string{"XDG_CONFIG_HOME", "APPDATA", "HOME"} {
		t.Setenv(k, dir)
	}
}

func TestBuiltinThemes(t *testing.T) {
	for _, e := range List() {
		if e.ID == "user" {
			continue
		}
		cfg := settings.Default()
		cfg.Theme = e.ID
		th, note := Resolve(cfg)
		if note != "" {
			t.Errorf("%s: %s", e.ID, note)
		}
		if (e.ID == "default") != (th == ui.Theme{}) {
			t.Errorf("%s: only the default theme may leave every color to the terminal: %+v", e.ID, th)
		}
	}
	cfg := settings.Default()
	cfg.Theme = "sakura"
	th, _ := Resolve(cfg)
	if th.EditorBG != "#000000" || th.NavBG != "#3a3a3a" || th.FrameFG != "#c0c0c0" || th.NavFG != "#ffffff" {
		t.Fatalf("sakura = %+v", th)
	}
}

func TestOverridesInConfig(t *testing.T) {
	cfg := settings.Default()
	cfg.Theme = "ocean"
	cfg.Themes = map[string]settings.ThemeColors{"ocean": {EditorForeground: "#ff0000", TitleForeground: "nonsense"}}
	th, note := Resolve(cfg)
	if th.EditorFG != "#ff0000" || th.TitleFG != "" || note == "" {
		t.Fatalf("th=%+v note=%q", th, note)
	}
}

func TestUserThemeFile(t *testing.T) {
	isolate(t)
	if err := EnsureUserFile(); err != nil {
		t.Fatal(err)
	}
	cfg := settings.Default()
	cfg.Theme = "user"
	if th, note := Resolve(cfg); th != (ui.Theme{}) || note != "" {
		t.Fatalf("fresh theme.conf must mean no theme: %+v %q", th, note)
	}
	path, _ := UserPath()
	data, _ := os.ReadFile(path)
	edited := strings.Replace(string(data), `editor_foreground    = "default"`, `editor_foreground    = "light_green"`, 1)
	edited = strings.Replace(edited, `title_foreground     = "default"`, `title_foreground     = "mine"`, 1)
	edited = strings.Replace(edited, "\n[colors]\n", "\n[colors]\nmine = \"#123\"\n", 1)
	os.WriteFile(path, []byte(edited), 0o644)
	th, note := Resolve(cfg)
	if note != "" || th.EditorFG != "#90ee90" || th.TitleFG != "#112233" {
		t.Fatalf("th=%+v note=%q", th, note)
	}
}
