package main

import (
	"errors"
	"os"
	"slices"

	"github.com/charmbracelet/bubbles/key"
)

const maxUndo = 50

var (
	undoKey = key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "undo"),
	)
	redoKey = key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "redo"),
	)
)

// snapshot is the state of one list before a change. Restoring it replaces
// only that list, so undo never touches the other scopes in the shared file.
type snapshot struct {
	dir   string // "" is the global list
	items []todo
	idx   int // item to select afterwards, as an index into items
	label string
}

type history struct {
	undo []snapshot
	redo []snapshot
}

func (m *model) clearHistory() {
	m.history = history{}
}

// pushUndo clones items because callers keep mutating the slice in place.
func (m *model) pushUndo(dir string, items []todo, idx int, label string) {
	m.history.undo = append(m.history.undo, snapshot{dir: dir, items: slices.Clone(items), idx: idx, label: label})
	if len(m.history.undo) > maxUndo {
		m.history.undo = m.history.undo[1:]
	}
	m.history.redo = nil
}

// recordUndo captures the plain list view; call it after reload and before mutating.
func (m *model) recordUndo(label string) {
	idx := 0
	if visible := m.visibleIndexes(); m.cursor < len(visible) {
		idx = visible[m.cursor]
	}
	m.pushUndo(m.activeListDir(), m.store.Items, idx, label)
}

// reload picks up changes made on disk since the list was loaded (for example
// by `td add` in another terminal), so a mutation and its undo snapshot are
// based on what the file really contains rather than on a stale view.
func (m *model) reload() error {
	// Without a file there is nothing to pick up, and the in-memory list wins.
	if _, err := os.Stat(m.location.Path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	file, err := readStoreFile(m.location.Path)
	if err != nil {
		return err
	}
	m.store.Items = slices.Clone(sectionItems(file, m.activeListDir()))
	m.clampCursor()
	return nil
}

func (m *model) undo() error {
	return m.stepHistory(&m.history.undo, &m.history.redo, "Undid", "Nothing to undo")
}

func (m *model) redo() error {
	return m.stepHistory(&m.history.redo, &m.history.undo, "Redid", "Nothing to redo")
}

// stepHistory moves the newest snapshot of from into the file and records the
// replaced state in to, which is what makes undo and redo mirror each other.
func (m *model) stepHistory(from, to *[]snapshot, verb, empty string) error {
	if len(*from) == 0 {
		m.notice = empty
		return nil
	}

	file, err := readStoreFile(m.location.Path)
	if err != nil {
		return err
	}

	snap := (*from)[len(*from)-1]
	*from = (*from)[:len(*from)-1]
	*to = append(*to, snapshot{
		dir:   snap.dir,
		items: slices.Clone(sectionItems(file, snap.dir)),
		idx:   snap.idx,
		label: snap.label,
	})

	setSectionItems(&file, snap.dir, slices.Clone(snap.items))
	if err := writeStoreFile(m.location.Path, file); err != nil {
		return err
	}
	m.notice = verb + ": " + snap.label

	m.animatingDoneIndex, m.animatingDoneFrames = -1, 0
	if m.overview {
		if err := m.refreshOverview(); err != nil {
			return err
		}
		m.selectRow(snap.dir, snap.idx)
		return nil
	}
	m.store.Items = slices.Clone(snap.items)
	m.cursor = m.cursorForIndex(snap.idx)
	m.clampCursor()
	return nil
}

func quoted(description string) string {
	runes := []rune(description)
	if len(runes) > 30 {
		description = string(runes[:29]) + "…"
	}
	return `"` + description + `"`
}

func (m model) noticeLine() string {
	if m.notice == "" {
		return ""
	}
	return "\n" + accentStyle.Render(m.notice)
}
