# wordterm
Simple TUI writing application aimed at story writers.

## Architecture overview

Wordterm follows a layered architecture that keeps the project model, working-copy storage, and terminal UI separate so the original project remains untouched until explicit save or autosave.

- app: orchestration, lifecycle, key handling, and TUI state
- editor: text buffer, cursor movement, undo/redo, wrapping, and viewport logic
- navigator: hierarchy selection and chapter/scene navigation
- project: metadata, chapter/scene model, project loading, and active selection state
- storage: temporary working copies, file writing, project serialization, and recovery
- settings: persistent app preferences and keyboard bindings
- ui: rendering helpers and screen layout routines

## Design decisions

1. Working copy isolation is the primary invariant. Every read and write during an active session happens against a temp project mirror, while the original project is only updated by explicit save or autosave.
2. The app state remains intentionally simple: a project model, an editor model, a navigator model, and a settings model.
3. Project metadata is stored as JSON and kept human-readable so it can be inspected and debugged easily.
4. Platform differences are isolated in the storage and config packages to avoid OS-specific assumptions in the UI logic.

## Implementation plan

Phase 1: foundation and app shell
- Initialize the Go modules and TUI shell
- Create the main app model, terminal sizing logic, and resize handling
- Establish the basic Navigator + Editor layout

Phase 2: project and storage model
- Implement project metadata and chapter/scene loading
- Create temp working copies and recovery logic
- Add save, reload, and dirty-state tracking

Phase 3: editor behavior
- Implement a usable plain-text editor buffer
- Add cursor movement, scrolling, line breaks, and basic undo/redo
- Support Unicode-aware display width and wrapped output

Phase 4: navigator and project operations
- Add chapter/scene selection, creation, deletion, and rename flow
- Support scene movement between chapters and change ordering

Phase 5: settings and menus
- Persist configuration in TOML
- Add keyboard rebinding, autosave, and word-wrap controls
- Wire the settings menu and help overlay

Phase 6: polish and distribution
- Screenwriter mode, save-as flow, and project recovery checks
- Validate platform-specific behavior and final compilation
