// Package themes defines the built-in color themes, the named-color palette and
// the user theme file (theme.conf), and turns them into a ui.Theme.
package themes

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/Vahtera/wordterm/internal/settings"
	"github.com/Vahtera/wordterm/internal/storage"
	"github.com/Vahtera/wordterm/internal/ui"
)

type Entry struct{ ID, Label string }

// List is the order shown in Settings > Change Theme...
func List() []Entry {
	return []Entry{
		{"default", "Default (terminal colors)"},
		{"high-contrast", "High-contrast"},
		{"pastel", "Pastel"},
		{"sakura", "Sakura"},
		{"ocean", "Ocean"},
		{"user", "User Theme"},
	}
}

type color struct{ name, hex string }

// palette lists every color name usable in theme.conf and in config.toml overrides.
var palette = []color{
	{"black", "#000000"}, {"dark_gray", "#3a3a3a"}, {"gray", "#808080"}, {"light_gray", "#c0c0c0"}, {"white", "#ffffff"}, {"bright_white", "#ffffff"},
	{"dark_red", "#8b0000"}, {"red", "#e03c31"}, {"light_red", "#ff8a80"}, {"bright_red", "#ff0000"},
	{"dark_green", "#006400"}, {"green", "#2ecc40"}, {"light_green", "#90ee90"}, {"bright_green", "#00ff00"},
	{"dark_blue", "#00008b"}, {"blue", "#3b82f6"}, {"light_blue", "#9ecbff"}, {"bright_blue", "#5c5cff"},
	{"dark_yellow", "#b8860b"}, {"yellow", "#ffd600"}, {"light_yellow", "#fff59d"}, {"bright_yellow", "#ffff00"},
	{"dark_cyan", "#008b8b"}, {"cyan", "#00bcd4"}, {"light_cyan", "#a5f3fc"}, {"bright_cyan", "#00ffff"},
	{"dark_magenta", "#8b008b"}, {"magenta", "#d946ef"}, {"light_magenta", "#f0abfc"}, {"bright_magenta", "#ff00ff"},
	{"dark_orange", "#cc5500"}, {"orange", "#ff9800"}, {"light_orange", "#ffcc80"},
	{"dark_pink", "#c2185b"}, {"pink", "#ff7eb6"}, {"light_pink", "#ffc0d9"},
	{"dark_purple", "#4a148c"}, {"purple", "#9c27b0"}, {"light_purple", "#ce93d8"},
	{"dark_brown", "#5d4037"}, {"brown", "#8d6e63"}, {"light_brown", "#bcaaa4"},
	{"dark_teal", "#00695c"}, {"teal", "#009688"}, {"light_teal", "#80cbc4"},
}

var paletteMap = func() map[string]string {
	m := map[string]string{}
	for _, c := range palette {
		m[c.name] = c.hex
	}
	return m
}()

var builtin = map[string]settings.ThemeColors{
	"default": {},
	"high-contrast": {
		EditorBackground: "black", NavigatorBackground: "black", OverlayBackground: "black",
		NavigatorForeground: "bright_white", EditorForeground: "bright_white", OverlayForeground: "bright_white",
		TitleForeground: "bright_yellow", FrameForeground: "bright_green",
	},
	"pastel": {
		EditorBackground: "black", NavigatorBackground: "dark_gray", OverlayBackground: "dark_gray",
		NavigatorForeground: "#aec6cf", EditorForeground: "#fff1d6", OverlayForeground: "#b5ead7",
		TitleForeground: "#ffdac1", FrameForeground: "#c7ceea",
	},
	"sakura": {
		EditorBackground: "black", NavigatorBackground: "dark_gray", OverlayBackground: "dark_gray",
		NavigatorForeground: "white", EditorForeground: "#ffd1dc", OverlayForeground: "white",
		TitleForeground: "#ffb7c5", FrameForeground: "light_gray",
	},
	"ocean": {
		EditorBackground: "#08161f", NavigatorBackground: "#0d2433", OverlayBackground: "#123247",
		NavigatorForeground: "#8fe3cf", EditorForeground: "#d7f3ff", OverlayForeground: "#bfe6ff",
		TitleForeground: "#4dd0e1", FrameForeground: "#3b82c4",
	},
}

func normHex(v string) (string, bool) {
	h := strings.TrimPrefix(v, "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return "", false
	}
	if _, err := strconv.ParseUint(h, 16, 32); err != nil {
		return "", false
	}
	return "#" + strings.ToLower(h), true
}

// resolveColor turns a name, hex value or "default" into "#rrggbb" ("" = terminal color).
func resolveColor(v string, extra map[string]string) (string, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	switch v {
	case "", "default", "none", "terminal", "inherit":
		return "", nil
	}
	if strings.HasPrefix(v, "#") {
		if hex, ok := normHex(v); ok {
			return hex, nil
		}
		return "", fmt.Errorf("invalid color %q", v)
	}
	key := strings.ReplaceAll(v, "-", "_")
	if hex, ok := extra[key]; ok {
		return hex, nil
	}
	if hex, ok := paletteMap[key]; ok {
		return hex, nil
	}
	return "", fmt.Errorf("unknown color %q", v)
}

func toTheme(c settings.ThemeColors, extra map[string]string) (ui.Theme, []string) {
	var errs []string
	get := func(field, v string) string {
		hex, err := resolveColor(v, extra)
		if err != nil {
			errs = append(errs, field+": "+err.Error())
		}
		return hex
	}
	return ui.Theme{
		EditorBG:  get("editor_background", c.EditorBackground),
		NavBG:     get("navigator_background", c.NavigatorBackground),
		OverlayBG: get("overlay_background", c.OverlayBackground),
		NavFG:     get("navigator_foreground", c.NavigatorForeground),
		EditorFG:  get("editor_foreground", c.EditorForeground),
		OverlayFG: get("overlay_foreground", c.OverlayForeground),
		TitleFG:   get("title_foreground", c.TitleForeground),
		FrameFG:   get("frame_foreground", c.FrameForeground),
	}, errs
}

func merge(base, over settings.ThemeColors) settings.ThemeColors {
	set := func(dst *string, v string) {
		if v != "" {
			*dst = v
		}
	}
	set(&base.EditorBackground, over.EditorBackground)
	set(&base.NavigatorBackground, over.NavigatorBackground)
	set(&base.OverlayBackground, over.OverlayBackground)
	set(&base.NavigatorForeground, over.NavigatorForeground)
	set(&base.EditorForeground, over.EditorForeground)
	set(&base.OverlayForeground, over.OverlayForeground)
	set(&base.TitleForeground, over.TitleForeground)
	set(&base.FrameForeground, over.FrameForeground)
	return base
}

// UserPath is the location of theme.conf, next to config.toml.
func UserPath() (string, error) {
	p, err := settings.Path()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(p), "theme.conf"), nil
}

type userFile struct {
	Colors map[string]string    `toml:"colors"`
	Theme  settings.ThemeColors `toml:"theme"`
}

func loadUser() (settings.ThemeColors, map[string]string, error) {
	path, err := UserPath()
	if err != nil {
		return settings.ThemeColors{}, nil, nil
	}
	if _, err := os.Stat(path); err != nil {
		return settings.ThemeColors{}, nil, nil
	}
	var f userFile
	if _, err := toml.DecodeFile(path, &f); err != nil {
		return settings.ThemeColors{}, nil, err
	}
	extra := map[string]string{}
	for k, v := range f.Colors {
		hex, ok := normHex(strings.ToLower(strings.TrimSpace(v)))
		if !ok {
			return settings.ThemeColors{}, nil, fmt.Errorf("color %q has an invalid value %q", k, v)
		}
		extra[strings.ReplaceAll(strings.ToLower(k), "-", "_")] = hex
	}
	return f.Theme, extra, nil
}

// Resolve returns the colors for cfg.Theme. The note is non-empty when part of the
// theme could not be used (the affected colors then follow the terminal).
func Resolve(cfg settings.Settings) (ui.Theme, string) {
	id := strings.ToLower(cfg.Theme)
	var c settings.ThemeColors
	var extra map[string]string
	var notes []string
	switch id {
	case "", "default":
	case "user":
		uc, ex, err := loadUser()
		if err != nil {
			notes = append(notes, "theme.conf: "+err.Error()+" (using terminal colors)")
		} else {
			c, extra = uc, ex
		}
	default:
		b, ok := builtin[id]
		if !ok {
			notes = append(notes, fmt.Sprintf("unknown theme %q (using terminal colors)", cfg.Theme))
		}
		c = merge(b, cfg.Themes[id])
	}
	th, errs := toTheme(c, extra)
	notes = append(notes, errs...)
	return th, strings.Join(notes, "; ")
}

func userTemplate() string {
	var b strings.Builder
	b.WriteString(`# Wordterm user theme. Choose it in Settings > Change Theme... > User Theme.
#
# [colors] lists every color name you can use below (names like dark_blue, light_green).
# You may change a value, or add your own:  my_color = "#rrggbb".
# [theme] sets one color for each part of the screen: a name from [colors], a hex value
# such as "#ff8800", or "default" to keep the terminal's own color for that part.
# With every entry "default" the theme looks exactly like an unthemed terminal.

[colors]
`)
	for _, c := range palette {
		fmt.Fprintf(&b, "%s = %q\n", c.name, c.hex)
	}
	b.WriteString(`
[theme]
editor_background    = "default"   # behind the text you write
navigator_background = "default"   # navigator pane, status bar and the frame lines
overlay_background   = "default"   # menus and dialogs
navigator_foreground = "default"   # navigator and status bar text
editor_foreground    = "default"   # your text
overlay_foreground   = "default"   # menu and dialog text
title_foreground     = "default"   # "Wordterm", "Navigator" and "Editor" titles
frame_foreground     = "default"   # the frame lines
`)
	return b.String()
}

// EnsureUserFile creates theme.conf (all colors "default") if it does not exist.
func EnsureUserFile() error {
	path, err := UserPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return storage.WriteFileAtomic(path, []byte(userTemplate()), 0o644)
}
