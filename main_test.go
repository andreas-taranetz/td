package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)



func TestEnterStartsAddingFirstItemWhenEmpty(t *testing.T) {
	tempDir := t.TempDir()
	location := storeLocation{Path: filepath.Join(tempDir, ".todos")}
	m := newModel(store{}, location)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected no command when starting first item edit")
	}
	next, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model result, got %T", updated)
	}
	if next.editMode != editModeNewBelow {
		t.Fatalf("expected new-below edit mode, got %v", next.editMode)
	}
	if next.input != "" {
		t.Fatalf("expected empty input, got %q", next.input)
	}
	if next.insertAt != 0 {
		t.Fatalf("expected insertAt 0, got %d", next.insertAt)
	}
	if help := next.editModeHelp(); help != "Adding item. Enter saves. Esc cancels." {
		t.Fatalf("unexpected edit help: %q", help)
	}
	view := next.View()
	if !strings.Contains(view, "Adding item. Enter saves. Esc cancels.") {
		t.Fatalf("expected simplified add help in view, got %q", view)
	}
}

func TestEmptyViewPromptsEnterToAdd(t *testing.T) {
	tempDir := t.TempDir()
	location := storeLocation{Path: filepath.Join(tempDir, ".todos")}
	m := newModel(store{}, location)
	view := m.View()

	if !strings.Contains(view, "No todos yet. Press enter to add one.") {
		t.Fatalf("expected enter prompt in empty view, got %q", view)
	}
	if strings.Contains(view, "Press o or O to add one") {
		t.Fatalf("expected old empty-state copy to be removed, got %q", view)
	}
}

func TestUppercaseEditShortcutsMatchLowercase(t *testing.T) {
	tempDir := t.TempDir()
	location := storeLocation{Path: filepath.Join(tempDir, ".todos")}
	base := newModel(store{Items: []todo{{Description: "write tests"}}}, location)

	updatedStart, _ := base.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'I'}})
	startModel, ok := updatedStart.(model)
	if !ok {
		t.Fatalf("expected model result for I, got %T", updatedStart)
	}
	if startModel.editMode != editModeCurrent {
		t.Fatalf("expected current edit mode for I, got %v", startModel.editMode)
	}
	if startModel.input != "write tests" {
		t.Fatalf("expected current item text for I, got %q", startModel.input)
	}
	if startModel.inputCursor != 0 {
		t.Fatalf("expected cursor at start for I, got %d", startModel.inputCursor)
	}

	updatedEnd, _ := base.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	endModel, ok := updatedEnd.(model)
	if !ok {
		t.Fatalf("expected model result for A, got %T", updatedEnd)
	}
	if endModel.editMode != editModeCurrent {
		t.Fatalf("expected current edit mode for A, got %v", endModel.editMode)
	}
	if endModel.input != "write tests" {
		t.Fatalf("expected current item text for A, got %q", endModel.input)
	}
	if endModel.inputCursor != len([]rune("write tests")) {
		t.Fatalf("expected cursor at end for A, got %d", endModel.inputCursor)
	}
}

func TestUpAtFirstItemStartsAddingAbove(t *testing.T) {
	tempDir := t.TempDir()
	location := storeLocation{Path: filepath.Join(tempDir, ".todos")}
	m := newModel(store{Items: []todo{{Description: "first"}, {Description: "second"}}}, location)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if cmd != nil {
		t.Fatal("expected no command when starting add-above")
	}
	next, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model result, got %T", updated)
	}
	if next.editMode != editModeNewAbove {
		t.Fatalf("expected new-above edit mode, got %v", next.editMode)
	}
	if next.insertAt != 0 {
		t.Fatalf("expected insertAt 0 for add-above, got %d", next.insertAt)
	}
	if help := next.editModeHelp(); help != "Adding item. Enter saves. Esc cancels." {
		t.Fatalf("unexpected edit help: %q", help)
	}
}

func TestDownAtLastItemStartsAddingBelow(t *testing.T) {
	tempDir := t.TempDir()
	location := storeLocation{Path: filepath.Join(tempDir, ".todos")}
	m := newModel(store{Items: []todo{{Description: "first"}, {Description: "second"}}}, location)
	m.cursor = 1

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if cmd != nil {
		t.Fatal("expected no command when starting add-below")
	}
	next, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model result, got %T", updated)
	}
	if next.editMode != editModeNewBelow {
		t.Fatalf("expected new-below edit mode, got %v", next.editMode)
	}
	if next.insertAt != 2 {
		t.Fatalf("expected insertAt 2 for add-below, got %d", next.insertAt)
	}
	if help := next.editModeHelp(); help != "Adding item. Enter saves. Esc cancels." {
		t.Fatalf("unexpected edit help: %q", help)
	}
}

func TestDirectionalAddDownSavesAndContinuesOnDown(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()
	location := storeLocation{Path: filepath.Join(tempDir, ".todos")}
	m := newModel(store{Items: []todo{{Description: "first"}, {Description: "second"}}}, location)
	m.cursor = 1

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	next := updated.(model)
	next.input = "third"
	next.inputCursor = len([]rune(next.input))

	updated, cmd := next.Update(tea.KeyMsg{Type: tea.KeyDown})
	if cmd != nil {
		t.Fatal("expected no command when chaining directional add")
	}
	chained, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model result, got %T", updated)
	}
	if len(chained.store.Items) != 3 {
		t.Fatalf("expected saved item before continuing, got %#v", chained.store.Items)
	}
	if chained.store.Items[2].Description != "third" {
		t.Fatalf("expected saved third item, got %#v", chained.store.Items)
	}
	if chained.editMode != editModeNewBelow {
		t.Fatalf("expected to continue in new-below edit mode, got %v", chained.editMode)
	}
	if chained.input != "" {
		t.Fatalf("expected fresh empty input after chaining, got %q", chained.input)
	}
	if chained.insertAt != 3 {
		t.Fatalf("expected next insertAt at end, got %d", chained.insertAt)
	}
	if chained.directionalNewItem != 1 {
		t.Fatalf("expected downward directional add marker, got %d", chained.directionalNewItem)
	}
	data, err := os.ReadFile(location.Path)
	if err != nil {
		t.Fatalf("load saved store: %v", err)
	}
	var saved store
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("decode saved store: %v", err)
	}
	if len(saved.Items) != 3 || saved.Items[2].Description != "third" {
		t.Fatalf("expected saved chained item in store file, got %#v", saved.Items)
	}
}

func TestDirectionalAddDownCancelsOnUp(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()
	location := storeLocation{Path: filepath.Join(tempDir, ".todos")}
	m := newModel(store{Items: []todo{{Description: "first"}, {Description: "second"}}}, location)
	m.cursor = 1

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	next := updated.(model)
	next.input = "discard me"
	next.inputCursor = len([]rune(next.input))

	updated, cmd := next.Update(tea.KeyMsg{Type: tea.KeyUp})
	if cmd != nil {
		t.Fatal("expected no command when cancelling directional add")
	}
	cancelled, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model result, got %T", updated)
	}
	if cancelled.isEditing() {
		t.Fatal("expected reverse direction to cancel edit")
	}
	if len(cancelled.store.Items) != 2 {
		t.Fatalf("expected no new item after cancel, got %#v", cancelled.store.Items)
	}
	if cancelled.directionalNewItem != 0 {
		t.Fatalf("expected directional marker reset, got %d", cancelled.directionalNewItem)
	}
	if _, err := os.Stat(location.Path); err == nil {
		t.Fatal("expected no store file to be created after cancel")
	}
}

func TestHideDoneKeepsCursorOnSameOpenTask(t *testing.T) {
	tempDir := t.TempDir()
	location := storeLocation{Path: filepath.Join(tempDir, ".todos")}
	m := newModel(store{Items: []todo{
		{Description: "open first"},
		{Description: "done", Done: true, DoneAt: time.Now()},
		{Description: "open keep"},
		{Description: "open last"},
	}}, location)
	m.cursor = 2

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	if cmd != nil {
		t.Fatal("expected no command when toggling done visibility")
	}
	next, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model result, got %T", updated)
	}
	if next.showAll {
		t.Fatal("expected done items to be hidden")
	}
	visible := next.visibleIndexes()
	if next.cursor != 1 {
		t.Fatalf("expected cursor to move to filtered row 1, got %d", next.cursor)
	}
	if visible[next.cursor] != 2 {
		t.Fatalf("expected cursor to stay on item 2, got item %d", visible[next.cursor])
	}
}

func TestDirectionalAddUpSavesAndContinuesOnUp(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()
	location := storeLocation{Path: filepath.Join(tempDir, ".todos")}
	m := newModel(store{Items: []todo{{Description: "first"}, {Description: "second"}}}, location)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	next := updated.(model)
	next.input = "zero"
	next.inputCursor = len([]rune(next.input))

	updated, cmd := next.Update(tea.KeyMsg{Type: tea.KeyUp})
	if cmd != nil {
		t.Fatal("expected no command when chaining upward directional add")
	}
	chained, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model result, got %T", updated)
	}
	if len(chained.store.Items) != 3 {
		t.Fatalf("expected saved item before continuing, got %#v", chained.store.Items)
	}
	if chained.store.Items[0].Description != "zero" {
		t.Fatalf("expected saved item at top, got %#v", chained.store.Items)
	}
	if chained.editMode != editModeNewAbove {
		t.Fatalf("expected to continue in new-above edit mode, got %v", chained.editMode)
	}
	if chained.insertAt != 0 {
		t.Fatalf("expected next insertAt at top, got %d", chained.insertAt)
	}
	if chained.directionalNewItem != -1 {
		t.Fatalf("expected upward directional add marker, got %d", chained.directionalNewItem)
	}
}

func TestDirectionalAddUpCancelsOnDown(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()
	location := storeLocation{Path: filepath.Join(tempDir, ".todos")}
	m := newModel(store{Items: []todo{{Description: "first"}, {Description: "second"}}}, location)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	next := updated.(model)
	next.input = "discard top"
	next.inputCursor = len([]rune(next.input))

	updated, cmd := next.Update(tea.KeyMsg{Type: tea.KeyDown})
	if cmd != nil {
		t.Fatal("expected no command when cancelling upward directional add")
	}
	cancelled, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model result, got %T", updated)
	}
	if cancelled.isEditing() {
		t.Fatal("expected reverse direction to cancel edit")
	}
	if len(cancelled.store.Items) != 2 {
		t.Fatalf("expected no new item after cancel, got %#v", cancelled.store.Items)
	}
	if cancelled.directionalNewItem != 0 {
		t.Fatalf("expected directional marker reset, got %d", cancelled.directionalNewItem)
	}
}

func TestDirectionalAddTypingJAndKDoesNotTriggerBindings(t *testing.T) {
	tempDir := t.TempDir()
	location := storeLocation{Path: filepath.Join(tempDir, ".todos")}
	m := newModel(store{Items: []todo{{Description: "first"}, {Description: "second"}}}, location)
	m.cursor = 1

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	next := updated.(model)

	updated, cmd := next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if cmd != nil {
		t.Fatal("expected no command when typing j in new-item input")
	}
	typing, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model result, got %T", updated)
	}
	if !typing.isEditing() {
		t.Fatal("expected to remain in edit mode after typing j")
	}
	if typing.input != "j" {
		t.Fatalf("expected typed j in input, got %q", typing.input)
	}
	if typing.directionalNewItem != 1 {
		t.Fatalf("expected directional marker to remain set, got %d", typing.directionalNewItem)
	}

	updated, cmd = typing.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if cmd != nil {
		t.Fatal("expected no command when typing k in new-item input")
	}
	typing, ok = updated.(model)
	if !ok {
		t.Fatalf("expected model result, got %T", updated)
	}
	if typing.input != "jk" {
		t.Fatalf("expected typed jk in input, got %q", typing.input)
	}
}

func TestDirectionalAddArrowKeysStillChain(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()
	location := storeLocation{Path: filepath.Join(tempDir, ".todos")}
	m := newModel(store{Items: []todo{{Description: "first"}, {Description: "second"}}}, location)
	m.cursor = 1

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	next := updated.(model)
	next.input = "third"
	next.inputCursor = len([]rune(next.input))

	updated, cmd := next.Update(tea.KeyMsg{Type: tea.KeyDown})
	if cmd != nil {
		t.Fatal("expected no command when chaining directional add with down arrow")
	}
	chained, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model result, got %T", updated)
	}
	if len(chained.store.Items) != 3 || chained.store.Items[2].Description != "third" {
		t.Fatalf("expected saved third item, got %#v", chained.store.Items)
	}
	if chained.editMode != editModeNewBelow {
		t.Fatalf("expected to continue in new-below edit mode, got %v", chained.editMode)
	}
}

func TestParseArgsDelete(t *testing.T) {
	opts, err := parseArgs([]string{"-d", "3"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if opts.action != actionDelete {
		t.Fatalf("expected actionDelete, got %v", opts.action)
	}
	if opts.deleteIndex != 3 {
		t.Fatalf("expected deleteIndex 3, got %d", opts.deleteIndex)
	}
}

func TestParseArgsDeleteLongFlag(t *testing.T) {
	opts, err := parseArgs([]string{"--delete", "1"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if opts.action != actionDelete {
		t.Fatalf("expected actionDelete, got %v", opts.action)
	}
	if opts.deleteIndex != 1 {
		t.Fatalf("expected deleteIndex 1, got %d", opts.deleteIndex)
	}
}

func TestParseArgsDeleteRejectsMissingIndex(t *testing.T) {
	if _, err := parseArgs([]string{"-d"}); err == nil {
		t.Fatal("expected error for missing delete index")
	}
}

func TestParseArgsDeleteRejectsInvalidIndex(t *testing.T) {
	if _, err := parseArgs([]string{"-d", "abc"}); err == nil {
		t.Fatal("expected error for non-integer delete index")
	}
	if _, err := parseArgs([]string{"-d", "0"}); err == nil {
		t.Fatal("expected error for zero delete index")
	}
	if _, err := parseArgs([]string{"-d", "-1"}); err == nil {
		t.Fatal("expected error for negative delete index")
	}
}

func TestParseArgsDeleteRejectsCombinations(t *testing.T) {
	if _, err := parseArgs([]string{"-d", "1", "-l"}); err == nil {
		t.Fatal("expected error combining delete with list")
	}
	if _, err := parseArgs([]string{"-l", "-d", "1"}); err == nil {
		t.Fatal("expected error combining list with delete")
	}
	if _, err := parseArgs([]string{"-d", "1", "some text"}); err == nil {
		t.Fatal("expected error combining delete with todo text")
	}
	if _, err := parseArgs([]string{"-t", "-d", "1"}); err == nil {
		t.Fatal("expected error combining delete with position flag")
	}
}

func TestParseArgsGreedyPositional(t *testing.T) {
	opts, err := parseArgs([]string{"buy", "milk", "-p"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := strings.Join(opts.addArgs, " "); got != "buy milk -p" {
		t.Fatalf("expected addArgs %q, got %q", "buy milk -p", got)
	}

	opts, err = parseArgs([]string{"-p", "-t", "buy", "milk", "--unknown"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := strings.Join(opts.addArgs, " "); got != "buy milk --unknown" {
		t.Fatalf("expected addArgs %q, got %q", "buy milk --unknown", got)
	}
	if !opts.plain {
		t.Fatal("expected plain flag set")
	}
	if opts.position != addTop {
		t.Fatal("expected top position")
	}
}

func TestRunDeleteRemovesOpenItem(t *testing.T) {
	tempDir := t.TempDir()

	storePath := filepath.Join(tempDir, ".todos")
	s := store{Items: []todo{
		{Description: "first"},
		{Description: "second"},
		{Description: "third"},
	}}
	if err := saveStore(storePath, s); err != nil {
		t.Fatalf("save store: %v", err)
	}
	location := storeLocation{Path: storePath}

	if err := runDelete(2, false, s, location); err != nil {
		t.Fatalf("runDelete: %v", err)
	}

	data, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	var saved store
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("decode store: %v", err)
	}
	if len(saved.Items) != 2 {
		t.Fatalf("expected 2 items after delete, got %d", len(saved.Items))
	}
	if saved.Items[0].Description != "first" || saved.Items[1].Description != "third" {
		t.Fatalf("expected first and third items, got %#v", saved.Items)
	}
}

func TestRunDeleteSkipsDoneItems(t *testing.T) {
	tempDir := t.TempDir()

	storePath := filepath.Join(tempDir, ".todos")
	s := store{Items: []todo{
		{Description: "open first"},
		{Description: "done item", Done: true},
		{Description: "open second"},
	}}
	if err := saveStore(storePath, s); err != nil {
		t.Fatalf("save store: %v", err)
	}
	location := storeLocation{Path: storePath}

	if err := runDelete(2, false, s, location); err != nil {
		t.Fatalf("runDelete: %v", err)
	}

	data, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	var saved store
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("decode store: %v", err)
	}
	if len(saved.Items) != 2 {
		t.Fatalf("expected 2 items after delete, got %d", len(saved.Items))
	}
	if saved.Items[0].Description != "open first" || saved.Items[1].Description != "done item" {
		t.Fatalf("expected open first and done item to remain, got %#v", saved.Items)
	}
}

func TestRunDeleteRejectsOutOfRange(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, ".todos")
	s := store{Items: []todo{{Description: "only one"}}}
	if err := saveStore(storePath, s); err != nil {
		t.Fatalf("save store: %v", err)
	}
	location := storeLocation{Path: storePath}

	if err := runDelete(2, false, s, location); err == nil {
		t.Fatal("expected error for out-of-range delete index")
	}
	if err := runDelete(0, false, s, location); err == nil {
		t.Fatal("expected error for zero delete index")
	}
}

func TestFormatRelativeTaskTime(t *testing.T) {
	now := time.Date(2026, time.May, 7, 15, 30, 0, 0, time.UTC)
	tests := []struct {
		name string
		ts   time.Time
		want string
	}{
		{name: "same day", ts: time.Date(2026, time.May, 7, 9, 45, 0, 0, time.UTC), want: "09:45"},
		{name: "yesterday with time", ts: time.Date(2026, time.May, 6, 18, 45, 0, 0, time.UTC), want: "yesterday 18:45"},
		{name: "day of week", ts: time.Date(2026, time.May, 4, 11, 0, 0, 0, time.UTC), want: "on Monday"},
		{name: "last week", ts: time.Date(2026, time.April, 30, 12, 0, 0, 0, time.UTC), want: "last week"},
		{name: "two weeks ago", ts: time.Date(2026, time.April, 20, 12, 0, 0, 0, time.UTC), want: "2 weeks ago"},
		{name: "three weeks ago", ts: time.Date(2026, time.April, 14, 12, 0, 0, 0, time.UTC), want: "3 weeks ago"},
		{name: "last month", ts: time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC), want: "last month"},
		{name: "named month", ts: time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC), want: "in January"},
		{name: "last year", ts: time.Date(2025, time.December, 31, 12, 0, 0, 0, time.UTC), want: "last year"},
		{name: "years ago", ts: time.Date(2023, time.March, 1, 12, 0, 0, 0, time.UTC), want: "3 years ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatRelativeTaskTime(tt.ts, now); got != tt.want {
				t.Fatalf("formatRelativeTaskTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTaskTimestampText(t *testing.T) {
	now := time.Date(2026, time.May, 7, 15, 30, 0, 0, time.UTC)
	open := todo{Description: "open", CreatedAt: time.Date(2026, time.May, 7, 9, 45, 0, 0, time.UTC)}
	if got := taskTimestampText(open, now); got != "created 09:45" {
		t.Fatalf("open timestamp = %q, want %q", got, "created 09:45")
	}

	done := todo{Description: "done", Done: true, CreatedAt: time.Date(2026, time.May, 1, 9, 0, 0, 0, time.UTC), DoneAt: time.Date(2026, time.May, 6, 18, 45, 0, 0, time.UTC)}
	if got := taskTimestampText(done, now); got != "done yesterday 18:45" {
		t.Fatalf("done timestamp = %q, want %q", got, "done yesterday 18:45")
	}
}
