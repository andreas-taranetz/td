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

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
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

func TestLocalScopeIsStoredInGlobalFileAndIsolated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todos.json")
	dir := "/work/project"

	global := store{Items: []todo{{Description: "global item"}}}
	if err := saveStore(path, global); err != nil {
		t.Fatal(err)
	}

	local, location, err := loadStoreAt(path, dir, scopeLocal)
	if err != nil {
		t.Fatal(err)
	}
	if !location.Local || location.HasLocal || len(local.Items) != 0 {
		t.Fatalf("unexpected fresh local view: %+v %+v", location, local)
	}

	local.Items = append(local.Items, todo{Description: "local item"})
	if err := saveStore(path, local); err != nil {
		t.Fatal(err)
	}

	auto, location, err := loadStoreAt(path, dir, scopeAuto)
	if err != nil {
		t.Fatal(err)
	}
	if !location.Local || len(auto.Items) != 1 || auto.Items[0].Description != "local item" {
		t.Fatalf("auto should pick local section, got %+v", auto)
	}

	other, location, err := loadStoreAt(path, "/elsewhere", scopeAuto)
	if err != nil {
		t.Fatal(err)
	}
	if location.Local || len(other.Items) != 1 || other.Items[0].Description != "global item" {
		t.Fatalf("other folder should stay global, got %+v", other)
	}

	forcedGlobal, location, err := loadStoreAt(path, dir, scopeGlobal)
	if err != nil {
		t.Fatal(err)
	}
	if location.Local || !location.HasLocal || forcedGlobal.Items[0].Description != "global item" {
		t.Fatalf("forced global should ignore local section, got %+v", forcedGlobal)
	}

	// Writing the global list must not drop the local section.
	forcedGlobal.Items = append(forcedGlobal.Items, todo{Description: "second global"})
	if err := saveStore(path, forcedGlobal); err != nil {
		t.Fatal(err)
	}
	again, _, err := loadStoreAt(path, dir, scopeLocal)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Items) != 1 || again.Items[0].Description != "local item" {
		t.Fatalf("local section lost after global save: %+v", again)
	}
}

func TestSwitchScopeInModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todos.json")
	dir := "/work/project"
	if err := saveStore(path, store{Items: []todo{{Description: "g"}}}); err != nil {
		t.Fatal(err)
	}

	s, location, err := loadStoreAt(path, dir, scopeAuto)
	if err != nil {
		t.Fatal(err)
	}
	m := newModel(s, location)

	if err := m.switchScope(scopeLocal); err != nil {
		t.Fatal(err)
	}
	if !m.location.Local || len(m.store.Items) != 0 {
		t.Fatalf("expected empty local view, got %+v", m.location)
	}
	if err := m.switchScope(scopeGlobal); err != nil {
		t.Fatal(err)
	}
	if m.location.Local || len(m.store.Items) != 1 {
		t.Fatalf("expected global view, got %+v", m.location)
	}
}

func TestParseArgsScopeFlags(t *testing.T) {
	opts, err := parseArgs([]string{"-L", "-l"})
	if err != nil || opts.scope != scopeLocal {
		t.Fatalf("got %+v, %v", opts, err)
	}
	opts, err = parseArgs([]string{"--global", "text"})
	if err != nil || opts.scope != scopeGlobal || opts.action != actionAdd {
		t.Fatalf("got %+v, %v", opts, err)
	}
	if _, err := parseArgs([]string{"-g", "-L"}); err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestRemoveLocalStoreKeepsOtherScopes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todos.json")
	dir := "/work/project"
	if err := saveStore(path, store{Items: []todo{{Description: "g"}}}); err != nil {
		t.Fatal(err)
	}
	if err := saveStore(path, store{Dir: dir, Items: []todo{{Description: "l"}}}); err != nil {
		t.Fatal(err)
	}

	removed, err := removeLocalStore(path, dir)
	if err != nil || !removed {
		t.Fatalf("removed=%v err=%v", removed, err)
	}
	if removed, _ := removeLocalStore(path, dir); removed {
		t.Fatal("second removal should report nothing removed")
	}

	s, location, err := loadStoreAt(path, dir, scopeAuto)
	if err != nil || location.Local || len(s.Items) != 1 {
		t.Fatalf("expected global fallback, got %+v %+v %v", s, location, err)
	}
}

func TestRemoveLocalRequiresConfirmation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todos.json")
	dir := "/work/project"
	m := newModel(store{Dir: dir}, storeLocation{Path: path, Dir: dir, Local: true})
	if err := saveStore(path, m.store); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m.statusLine(), "ctrl+x") {
		t.Fatal("empty local list should hint at ctrl+x")
	}

	press := func(m model, k tea.KeyMsg) model {
		next, _ := m.Update(k)
		return next.(model)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyCtrlX})
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if !m.location.Local {
		t.Fatal("cancel must keep the local list")
	}

	m = press(m, tea.KeyMsg{Type: tea.KeyCtrlX})
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if m.location.Local {
		t.Fatal("confirm should switch to global")
	}
	if _, location, _ := loadStoreAt(path, dir, scopeAuto); location.HasLocal {
		t.Fatal("section should be gone from file")
	}
}

func TestOverviewTogglesAndDeletesAcrossLists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todos.json")
	dir := "/work/here"
	if err := saveStore(path, store{Items: []todo{{Description: "g1"}}}); err != nil {
		t.Fatal(err)
	}
	if err := saveStore(path, store{Dir: dir, Items: []todo{{Description: "h1"}, {Description: "h2"}}}); err != nil {
		t.Fatal(err)
	}
	if err := saveStore(path, store{Dir: "/work/other", Items: []todo{{Description: "o1"}}}); err != nil {
		t.Fatal(err)
	}

	s, location, err := loadStoreAt(path, dir, scopeAuto)
	if err != nil {
		t.Fatal(err)
	}
	m := newModel(s, location)
	press := func(k tea.KeyMsg) {
		next, _ := m.Update(k)
		m = next.(model)
	}
	rune := func(r string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(r)} }

	press(tea.KeyMsg{Type: tea.KeyCtrlA})
	// header, h1, h2, header, g1, header, o1
	if len(m.ovRows) != 7 || !m.ovRows[0].header || m.ovRows[1].item.Description != "h1" {
		t.Fatalf("unexpected rows: %+v", m.ovRows)
	}
	if m.ovRows[m.ovCursor].item.Description != "h1" {
		t.Fatal("cursor should start on the first todo, not a header")
	}

	press(rune("j"))
	press(rune("j")) // skips the Global header
	if m.ovRows[m.ovCursor].item.Description != "g1" {
		t.Fatalf("cursor on %+v", m.ovRows[m.ovCursor])
	}
	press(rune("x"))
	press(rune("j")) // o1 is now the next row, as g1 is hidden
	press(rune("d"))

	file, err := readStoreFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !file.Items[0].Done {
		t.Fatal("global item should be done")
	}
	if len(file.Folders["/work/other"].Items) != 0 {
		t.Fatal("o1 should be deleted")
	}
	if len(file.Folders[dir].Items) != 2 {
		t.Fatal("current folder list must be untouched")
	}
}

func TestOverviewToggleHideDone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todos.json")
	dir := "/work/here"
	if err := saveStore(path, store{Items: []todo{{Description: "open"}, {Description: "finished", Done: true}}}); err != nil {
		t.Fatal(err)
	}

	s, location, err := loadStoreAt(path, dir, scopeAuto)
	if err != nil {
		t.Fatal(err)
	}
	m := newModel(s, location)
	press := func(k tea.KeyMsg) {
		next, _ := m.Update(k)
		m = next.(model)
	}

	press(tea.KeyMsg{Type: tea.KeyCtrlA})
	if len(m.ovRows) != 3 {
		t.Fatalf("done items visible by default, got %+v", m.ovRows)
	}
	press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if len(m.ovRows) != 2 {
		t.Fatalf("done item should be hidden, got %+v", m.ovRows)
	}
	file, err := readStoreFile(path)
	if err != nil || !file.HideDoneInTUI {
		t.Fatalf("preference not persisted: %+v %v", file, err)
	}
}

func TestLocalListIsInheritedBySubfolders(t *testing.T) {
	home := resolvedTempDir(t)
	t.Setenv("HOME", home)
	path := filepath.Join(t.TempDir(), "todos.json")
	project := filepath.Join(home, "project")
	nested := filepath.Join(project, "nested")
	sub := filepath.Join(project, "src", "deep")

	for _, dir := range []string{project, nested} {
		if err := saveStore(path, store{Dir: dir, Items: []todo{{Description: dir}}}); err != nil {
			t.Fatal(err)
		}
	}

	s, location, err := loadStoreAt(path, sub, scopeAuto)
	if err != nil {
		t.Fatal(err)
	}
	if !location.Local || location.LocalDir != project || s.Dir != project {
		t.Fatalf("subfolder should use the project list, got %+v", location)
	}

	// The nearest list wins over a farther parent.
	_, location, _ = loadStoreAt(path, filepath.Join(nested, "x"), scopeAuto)
	if location.LocalDir != nested {
		t.Fatalf("nearest list should win, got %q", location.LocalDir)
	}

	// -L with no list anywhere targets the current folder.
	other := filepath.Join(t.TempDir(), "unrelated")
	_, location, _ = loadStoreAt(path, other, scopeLocal)
	if location.HasLocal || location.LocalDir != other {
		t.Fatalf("expected new list for current folder, got %+v", location)
	}
}

func TestHomeListIsNotInherited(t *testing.T) {
	home := resolvedTempDir(t)
	t.Setenv("HOME", home)
	path := filepath.Join(t.TempDir(), "todos.json")
	if err := saveStore(path, store{Dir: home, Items: []todo{{Description: "home"}}}); err != nil {
		t.Fatal(err)
	}

	_, location, _ := loadStoreAt(path, filepath.Join(home, "project"), scopeAuto)
	if location.Local || location.HasLocal {
		t.Fatalf("home list must not capture subfolders, got %+v", location)
	}
	_, location, _ = loadStoreAt(path, home, scopeAuto)
	if !location.Local {
		t.Fatal("home itself may still have a list")
	}
}

func TestCreateLocalIgnoresParentList(t *testing.T) {
	home := resolvedTempDir(t)
	t.Setenv("HOME", home)
	path := filepath.Join(t.TempDir(), "todos.json")
	project := filepath.Join(home, "project")
	sub := filepath.Join(project, "sub")
	if err := saveStore(path, store{Dir: project, Items: []todo{{Description: "parent"}}}); err != nil {
		t.Fatal(err)
	}

	s, location, err := loadStoreAt(path, sub, scopeCreateLocal)
	if err != nil {
		t.Fatal(err)
	}
	if !location.Local || location.HasLocal || location.LocalDir != sub || s.Dir != sub || len(s.Items) != 0 {
		t.Fatalf("expected a fresh list for the exact folder, got %+v %+v", location, s)
	}

	s.Items = append(s.Items, todo{Description: "child"})
	if err := saveStore(path, s); err != nil {
		t.Fatal(err)
	}
	_, location, _ = loadStoreAt(path, sub, scopeCreateLocal)
	if !location.HasLocal {
		t.Fatal("exact list should exist after the first add")
	}
	_, location, _ = loadStoreAt(path, sub, scopeAuto)
	if location.LocalDir != sub {
		t.Fatalf("auto should now prefer the exact folder's list, got %q", location.LocalDir)
	}

	opts, err := parseArgs([]string{"--create-local", "-l"})
	if err != nil || opts.scope != scopeCreateLocal {
		t.Fatalf("got %+v, %v", opts, err)
	}
	if _, err := parseArgs([]string{"--create-local", "-L"}); err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestOverviewEditAndListJumps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todos.json")
	dir := "/work/here"
	for _, s := range []store{
		{Dir: dir, Items: []todo{{Description: "h1"}, {Description: "h2"}}},
		{Items: []todo{{Description: "g1"}, {Description: "g2"}}},
		{Dir: "/work/other", Items: []todo{{Description: "o1"}}},
	} {
		if err := saveStore(path, s); err != nil {
			t.Fatal(err)
		}
	}

	s, location, err := loadStoreAt(path, dir, scopeAuto)
	if err != nil {
		t.Fatal(err)
	}
	m := newModel(s, location)
	press := func(k tea.KeyMsg) {
		next, _ := m.Update(k)
		m = next.(model)
	}
	keyOf := func(r string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(r)} }
	typeText := func(text string) {
		for _, r := range text {
			press(keyOf(string(r)))
		}
		press(tea.KeyMsg{Type: tea.KeyEnter})
	}
	current := func() string { return m.ovRows[m.ovCursor].item.Description }

	press(tea.KeyMsg{Type: tea.KeyCtrlA})
	press(keyOf("}"))
	if current() != "g1" {
		t.Fatalf("} should jump to the next list, on %q", current())
	}
	press(keyOf("j"))
	press(keyOf("{"))
	if current() != "g1" {
		t.Fatalf("{ should go to the start of this list, on %q", current())
	}
	press(keyOf("{"))
	if current() != "h1" {
		t.Fatalf("{ again should reach the previous list, on %q", current())
	}
	press(keyOf("]"))
	press(keyOf("]"))
	if current() != "g1" {
		t.Fatalf("]] should jump like }, on %q", current())
	}

	// Add below g1: lands in the global list between g1 and g2.
	press(keyOf("o"))
	typeText("new")
	// Edit the current item from the end.
	press(keyOf("a"))
	typeText("!")

	file, err := readStoreFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, item := range file.Items {
		got = append(got, item.Description)
	}
	if strings.Join(got, ",") != "g1,new!,g2" {
		t.Fatalf("global list = %v", got)
	}
	if len(file.Folders[dir].Items) != 2 || len(file.Folders["/work/other"].Items) != 1 {
		t.Fatal("other lists must be untouched")
	}
	if current() != "new!" {
		t.Fatalf("cursor should follow the edited item, on %q", current())
	}

	press(keyOf("O"))
	press(tea.KeyMsg{Type: tea.KeyEsc})
	if m.isEditing() || !m.overview {
		t.Fatal("esc should cancel editing but stay in the overview")
	}
}

func TestOverviewMoveAndClearDoneStayInTheirList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todos.json")
	dir := "/work/here"
	for _, s := range []store{
		{Dir: dir, Items: []todo{{Description: "h1"}, {Description: "h2"}, {Description: "hd", Done: true}}},
		{Items: []todo{{Description: "g1"}, {Description: "gd", Done: true}}},
	} {
		if err := saveStore(path, s); err != nil {
			t.Fatal(err)
		}
	}

	s, location, err := loadStoreAt(path, dir, scopeAuto)
	if err != nil {
		t.Fatal(err)
	}
	m := newModel(s, location)
	press := func(r string) {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(r)})
		m = next.(model)
	}
	names := func(items []todo) string {
		var out []string
		for _, i := range items {
			out = append(out, i.Description)
		}
		return strings.Join(out, ",")
	}

	press("l") // no-op in the normal view; keeps the key table honest
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	m = next.(model)

	press("J")
	press("J") // second J must not cross into the done row or another list
	file, _ := readStoreFile(path)
	if got := names(file.Folders[dir].Items); got != "h2,hd,h1" && got != "h2,h1,hd" {
		t.Fatalf("unexpected order %s", got)
	}
	press("K")
	file, _ = readStoreFile(path)
	if names(file.Items) != "g1,gd" {
		t.Fatalf("global list must be untouched by moves, got %s", names(file.Items))
	}

	press("D")
	file, _ = readStoreFile(path)
	if strings.Contains(names(file.Folders[dir].Items), "hd") {
		t.Fatal("done item of the cursor's list should be gone")
	}
	if names(file.Items) != "g1,gd" {
		t.Fatal("D must not touch other lists")
	}
}

// resolvedTempDir avoids macOS's /var -> /private/var symlink, which currentDir resolves.
func resolvedTempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSymlinkedHomeIsResolved(t *testing.T) {
	real := resolvedTempDir(t)
	link := filepath.Join(resolvedTempDir(t), "home-link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", link)

	if got := shortenPath(filepath.Join(real, "projects", "x")); got != "~/projects/x" {
		t.Fatalf("shortenPath = %q", got)
	}

	path := filepath.Join(t.TempDir(), "todos.json")
	if err := saveStore(path, store{Dir: real, Items: []todo{{Description: "home"}}}); err != nil {
		t.Fatal(err)
	}
	if _, location, _ := loadStoreAt(path, filepath.Join(real, "sub"), scopeAuto); location.HasLocal {
		t.Fatal("home list must stay excluded when HOME is a symlink")
	}
}

func TestTitleTruncatesPathFromTheStart(t *testing.T) {
	home := resolvedTempDir(t)
	t.Setenv("HOME", home)
	l := storeLocation{Local: true, LocalDir: filepath.Join(home, "projects", "demo")}

	if got := l.title(0); got != "Todo: ~/projects/demo" {
		t.Fatalf("no width limit, got %q", got)
	}
	if got := l.title(40); got != "Todo: ~/projects/demo" {
		t.Fatalf("fits, got %q", got)
	}
	got := l.title(24) // 24 - 6 padding - 6 prefix = 12 cells for the path
	if got != "Todo: …ojects/demo" || len([]rune(got))-len("Todo: ") != 12 {
		t.Fatalf("got %q", got)
	}
	if got := l.title(12); got != "Todo: …" {
		t.Fatalf("tiny width, got %q", got)
	}
	if got := (storeLocation{}).title(10); got != "Todo:" {
		t.Fatalf("global title, got %q", got)
	}
}
