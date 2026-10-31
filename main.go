package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"wordterm/internal/app"
	"wordterm/internal/project"
	"wordterm/internal/settings"
	"wordterm/internal/storage"
)

const usage = `usage:
  wordterm                       open the built-in help (read-only); create a project from the menu
  wordterm <project-dir>         open a project; a bare name (wordterm Novel) is looked up
                                 in the default save path first, then the current directory
  wordterm new <name> [-path DIR] [-author NAME]
                                 create a project and open it (DIR and NAME default to the settings)`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "wordterm:", err)
		os.Exit(1)
	}
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// resolveProject turns the command-line argument into a project directory.
// An explicit path (absolute, ~, or containing a separator) is used as given.
// A bare name is tried in the default save path first, then the current directory.
func resolveProject(arg string, cfg settings.Settings) (string, error) {
	arg = settings.ExpandPath(arg)
	if filepath.IsAbs(arg) || arg == "." || arg == ".." || strings.ContainsAny(arg, `/\`) {
		return arg, nil
	}
	var errs []error
	for _, base := range []string{settings.ExpandPath(cfg.DefaultSavePath), "."} {
		if base == "" {
			continue
		}
		dir := filepath.Join(base, arg)
		_, err := project.Load(dir)
		if err == nil {
			return dir, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", dir, err))
	}
	return "", fmt.Errorf("no project named %q found:\n%w", arg, errors.Join(errs...))
}

// projectDir resolves the command line to a project directory (creating it for "new").
func projectDir(args []string, cfg settings.Settings) (string, error) {
	switch {
	case len(args) == 1 && args[0] != "new":
		return resolveProject(args[0], cfg)
	case len(args) >= 2 && args[0] == "new":
		fs := flag.NewFlagSet("new", flag.ContinueOnError)
		path := fs.String("path", "", "folder to create the project in")
		author := fs.String("author", "", "author name")
		if err := fs.Parse(args[2:]); err != nil {
			return "", err
		}
		parent := firstNonEmpty(settings.ExpandPath(*path), settings.ExpandPath(cfg.DefaultSavePath), ".")
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return "", err
		}
		return project.Create(parent, args[1], firstNonEmpty(*author, cfg.AuthorName), "")
	}
	return "", errors.New(usage)
}

func run(args []string) error {
	cfg, note := settings.Load()
	sess, err := offerRecovery()
	if err != nil {
		return err
	}
	var model app.Model
	switch {
	case sess != nil: // recovered a crashed session
		proj, err := project.Load(sess.Root)
		if err != nil {
			return fmt.Errorf("working copy %s: %w", sess.Dir, err) // working copy is kept
		}
		model = app.New(sess, proj, cfg, note)
	case len(args) == 0:
		var helpDir string
		if model, helpDir, err = app.NewWelcome(cfg, note); err != nil {
			return err
		}
		defer os.RemoveAll(helpDir)
	default:
		dir, err := projectDir(args, cfg)
		if err != nil {
			return err
		}
		if _, err := project.Load(dir); err != nil { // validate the original before copying
			return fmt.Errorf("cannot open %s: %w", dir, err)
		}
		if sess, err = storage.Open(dir); err != nil {
			return err
		}
		proj, err := project.Load(sess.Root)
		if err != nil {
			sess.Close()
			return err
		}
		model = app.New(sess, proj, cfg, note)
	}

	final, runErr := tea.NewProgram(model).Run()

	// Normal exit: delete the working copy if the user discarded it or nothing would be lost.
	cur, discarded := model.Session(), false
	if fm, ok := final.(app.Model); ok {
		cur, discarded = fm.Session(), fm.Discarded() // the session changes after New Project
	}
	if has, err := cur.HasChanges(); discarded || (err == nil && !has) {
		cur.Close()
	} else {
		fmt.Fprintln(os.Stderr, "Unsaved working copy kept at:", cur.Dir)
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
