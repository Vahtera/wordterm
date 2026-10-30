package settings

// Settings stores the user preferences that persist between sessions.
type Settings struct {
	EditorWidth      int
	WriteSize        int
	WordWrap         bool
	Autosave         bool
	AutosaveInterval int
	Bindings         map[string]string
}

func Default() Settings {
	return Settings{
		EditorWidth:      40,
		WriteSize:        60,
		WordWrap:         true,
		Autosave:         false,
		AutosaveInterval: 30,
		Bindings: map[string]string{
			"help":         "f1",
			"menu":         "esc",
			"switch-pane":  "ctrl+tab",
			"save":         "f10",
			"reload":       "f9",
			"screenwriter": "f11",
		},
	}
}
