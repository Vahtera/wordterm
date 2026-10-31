package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"wordterm/internal/app"
	"wordterm/internal/project"
	"wordterm/internal/settings"
	"wordterm/internal/storage"
)

const usage = "usage: wordterm <project-dir>  |  wordterm new <parent-dir> <name> <author>"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "wordterm:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	sess, err := offerRecovery()
	if err != nil {
		return err
	}
	if sess == nil {
		var dir string
		switch {
		case len(args) == 4 && args[0] == "new":
			if dir, err = project.Create(args[1], args[2], args[3], ""); err != nil {
				return err
			}
		case len(args) == 1:
			dir = args[0]
		default:
			return errors.New(usage)
		}
		if _, err := project.Load(dir); err != nil { // validate the original before copying
			return fmt.Errorf("cannot open %s: %w", dir, err)
		}
		if sess, err = storage.Open(dir); err != nil {
			return err
		}
	}
	proj, err := project.Load(sess.Root)
	if err != nil {
		return fmt.Errorf("working copy %s: %w", sess.Dir, err) // working copy is kept
	}
	cfg, note := settings.Load()
	final, runErr := tea.NewProgram(app.New(sess, proj, cfg, note)).Run()

	// Normal exit: delete the working copy if the user discarded it or nothing would be lost.
	discarded := false
	if fm, ok := final.(app.Model); ok {
		discarded = fm.Discarded()
		sess = fm.Session() // the session may have changed (New Project)
	}
	if has, err := sess.HasChanges(); discarded || (err == nil && !has) {
		sess.Close()
	} else {
		fmt.Fprintln(os.Stderr, "Unsaved working copy kept at:", sess.Dir)
	}
	return runErr
}

// offerRecovery asks (before the TUI starts) what to do with working
// copies left behind by crashed sessions. It returns an adopted session,
// or nil to continue normally.
func offerRecovery() (*storage.Session, error) {
	recs, err := storage.FindRecoverable()
	if err != nil || len(recs) == 0 {
		return nil, err
	}
	in := bufio.NewReader(os.Stdin)
	for _, r := range recs {
		fmt.Printf("Unsaved work found for %s (started %s).\n", r.Original, r.Started.Format("2006-01-02 15:04"))
		fmt.Print("[R]ecover, [D]iscard, or [S]kip for now? [s] ")
		line, _ := in.ReadString('\n')
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "r":
			return storage.Adopt(r)
		case "d":
			if err := storage.Discard(r); err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}
