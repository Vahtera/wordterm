package main

import (
	"log"

	"wordterm/internal/app"

	tea "charm.land/bubbletea/v2"
)

func main() {
	m := app.NewModel()
	if _, err := tea.NewProgram(m).Run(); err != nil {
		log.Fatal(err)
	}
}
