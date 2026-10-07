package main

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func press(t *testing.T, m model, msg tea.KeyMsg) model {
	t.Helper()
	updated, _ := m.Update(msg)
	next, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model result, got %T", updated)
	}
	return next
}

func runes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func newSavedModel(t *testing.T, descriptions ...string) model {
	t.Helper()
	s := store{}
	for _, d := range descriptions {
		s.Items = append(s.Items, todo{Description: d})
	}
	location := storeLocation{Path: filepath.Join(t.TempDir(), ".todos")}
	if err := saveStore(location.Path, s); err != nil {
		t.Fatal(err)
	}
	return newModel(s, location)
}

func descriptions(t *testing.T, m model) []string {
	t.Helper()
	file, err := readStoreFile(m.location.Path)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, item := range file.Items {
		out = append(out, item.Description)
	}
	return out
}

func TestUndoRedoDelete(t *testing.T) {
	m := newSavedModel(t, "a", "b", "c")
	m.cursor = 1

	m = press(t, m, runes("d"))
	if got := strings.Join(descriptions(t, m), ","); got != "a,c" {
		t.Fatalf("after delete: %s", got)
	}

	m = press(t, m, runes("u"))
	if got := strings.Join(descriptions(t, m), ","); got != "a,b,c" {
		t.Fatalf("after undo: %s", got)
	}
	if m.cursor != 1 {
		t.Fatalf("expected cursor restored to 1, got %d", m.cursor)
	}
	if !strings.Contains(m.notice, "Undid: delete \"b\"") {
		t.Fatalf("unexpected notice %q", m.notice)
	}

	m = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlR})
	if got := strings.Join(descriptions(t, m), ","); got != "a,c" {
		t.Fatalf("after redo: %s", got)
	}
	if !strings.Contains(m.notice, "Redid: delete") {
		t.Fatalf("unexpected notice %q", m.notice)
	}
}

func TestUndoReportsNothingLeft(t *testing.T) {
	m := newSavedModel(t, "a")

	m = press(t, m, runes("u"))
	if m.notice != "Nothing to undo" {
		t.Fatalf("unexpected notice %q", m.notice)
	}
	if !strings.Contains(m.View(), "Nothing to undo") {
		t.Fatal("expected notice in view")
	}

	m = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlR})
	if m.notice != "Nothing to redo" {
		t.Fatalf("unexpected notice %q", m.notice)
	}

	// The notice is transient: any following key clears it.
	m = press(t, m, runes("j"))
	if m.notice != "" {
		t.Fatalf("expected notice cleared, got %q", m.notice)
	}
}

func TestNewChangeClearsRedo(t *testing.T) {
	m := newSavedModel(t, "a", "b")

	m = press(t, m, runes("d"))
	m = press(t, m, runes("u"))
	m = press(t, m, runes("x"))
	m = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlR})
	if m.notice != "Nothing to redo" {
		t.Fatalf("expected redo to be cleared, got %q", m.notice)
	}
}

func TestUndoCoversToggleMoveAddEditAndClearDone(t *testing.T) {
	m := newSavedModel(t, "a", "b")

	m = press(t, m, runes("x"))
	m = press(t, m, runes("J"))
	m = press(t, m, runes("D"))
	m = press(t, m, runes("o"))
	m = press(t, m, runes("n"))
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = press(t, m, runes("i"))
	m = press(t, m, runes("!"))
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	for i := 0; i < 5; i++ {
		m = press(t, m, runes("u"))
		if strings.Contains(m.notice, "Nothing") {
			t.Fatalf("undo %d found nothing to undo", i+1)
		}
	}

	file, _ := readStoreFile(m.location.Path)
	if len(file.Items) != 2 || file.Items[0].Description != "a" || file.Items[0].Done || file.Items[1].Description != "b" {
		t.Fatalf("expected original list, got %+v", file.Items)
	}
	m = press(t, m, runes("u"))
	if m.notice != "Nothing to undo" {
		t.Fatalf("expected empty stack, got %q", m.notice)
	}
}

func TestUndoKeepsExternalChangesInOtherLists(t *testing.T) {
	m := newSavedModel(t, "a")
	m = press(t, m, runes("d"))

	file, _ := readStoreFile(m.location.Path)
	file.Folders = map[string]folderTodos{"/x": {Items: []todo{{Description: "other"}}}}
	if err := writeStoreFile(m.location.Path, file); err != nil {
		t.Fatal(err)
	}

	m = press(t, m, runes("u"))
	file, _ = readStoreFile(m.location.Path)
	if len(file.Items) != 1 || len(file.Folders["/x"].Items) != 1 {
		t.Fatalf("undo clobbered other lists: %+v", file)
	}
}

func TestMutationReloadsExternalChangesFirst(t *testing.T) {
	m := newSavedModel(t, "a")

	file, _ := readStoreFile(m.location.Path)
	file.Items = append(file.Items, todo{Description: "added elsewhere"})
	if err := writeStoreFile(m.location.Path, file); err != nil {
		t.Fatal(err)
	}

	m = press(t, m, runes("x"))
	if got := strings.Join(descriptions(t, m), ","); got != "a,added elsewhere" {
		t.Fatalf("external change lost: %s", got)
	}

	m = press(t, m, runes("u"))
	if got := strings.Join(descriptions(t, m), ","); got != "a,added elsewhere" {
		t.Fatalf("undo lost external change: %s", got)
	}
}

func TestSwitchingScopeClearsHistory(t *testing.T) {
	dir := resolvedTempDir(t)
	location := storeLocation{Path: filepath.Join(dir, ".todos"), Dir: dir, LocalDir: dir}
	s := store{Items: []todo{{Description: "a"}}}
	if err := saveStore(location.Path, s); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, location)

	m = press(t, m, runes("d"))
	m = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlL})
	m = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlG})
	m = press(t, m, runes("u"))
	if m.notice != "Nothing to undo" {
		t.Fatalf("expected cleared history, got %q", m.notice)
	}
}

func TestOverviewUndoRedo(t *testing.T) {
	m := newSavedModel(t, "a", "b")
	m = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlA})

	m = press(t, m, runes("d"))
	if got := strings.Join(descriptions(t, m), ","); got != "b" {
		t.Fatalf("after delete: %s", got)
	}
	m = press(t, m, runes("u"))
	if got := strings.Join(descriptions(t, m), ","); got != "a,b" {
		t.Fatalf("after undo: %s", got)
	}
	if !strings.Contains(m.overviewView(), "Undid: delete") {
		t.Fatal("expected notice in overview")
	}
	m = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlR})
	if got := strings.Join(descriptions(t, m), ","); got != "b" {
		t.Fatalf("after redo: %s", got)
	}
}
