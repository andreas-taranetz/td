package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func canonicalPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved
	}
	return filepath.Clean(path)
}

func TestParseArgsHereList(t *testing.T) {
	opts, err := parseArgs([]string{"-H", "-l"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if opts.storage != storageHere {
		t.Fatalf("expected storageHere, got %v", opts.storage)
	}
	if opts.action != actionList {
		t.Fatalf("expected actionList, got %v", opts.action)
	}
}

func TestParseArgsGlobalAdd(t *testing.T) {
	opts, err := parseArgs([]string{"-g", "-t", "ship", "it"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if opts.storage != storageGlobal {
		t.Fatalf("expected storageGlobal, got %v", opts.storage)
	}
	if opts.action != actionAdd {
		t.Fatalf("expected actionAdd, got %v", opts.action)
	}
	if opts.position != addTop {
		t.Fatalf("expected addTop, got %v", opts.position)
	}
	if len(opts.addArgs) != 2 || opts.addArgs[0] != "ship" || opts.addArgs[1] != "it" {
		t.Fatalf("unexpected add args: %#v", opts.addArgs)
	}
}

func TestParseArgsRejectsHereAndGlobal(t *testing.T) {
	if _, err := parseArgs([]string{"-H", "-g"}); err == nil {
		t.Fatal("expected parseArgs to reject -H with -g")
	}
}

func TestResolveStoreLocationPrefersLocal(t *testing.T) {
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

	localPath := filepath.Join(tempDir, ".todos")
	if err := saveStore(localPath, store{}); err != nil {
		t.Fatalf("save local store: %v", err)
	}

	location, err := resolveStoreLocation(storageAuto)
	if err != nil {
		t.Fatalf("resolveStoreLocation: %v", err)
	}
	if location.Scope != storeScopeLocal {
		t.Fatalf("expected local scope, got %v", location.Scope)
	}
	if canonicalPath(t, location.Path) != canonicalPath(t, localPath) {
		t.Fatalf("expected local path %q, got %q", localPath, location.Path)
	}
}

func TestResolveStoreLocationGlobalOverride(t *testing.T) {
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

	localPath := filepath.Join(tempDir, ".todos")
	if err := saveStore(localPath, store{}); err != nil {
		t.Fatalf("save local store: %v", err)
	}

	location, err := resolveStoreLocation(storageGlobal)
	if err != nil {
		t.Fatalf("resolveStoreLocation: %v", err)
	}
	if location.Scope != storeScopeGlobal {
		t.Fatalf("expected global scope, got %v", location.Scope)
	}
	if canonicalPath(t, location.Path) == canonicalPath(t, localPath) {
		t.Fatalf("expected global path, got local path %q", location.Path)
	}
}

func TestResolveStoreLocationHereCreatesFile(t *testing.T) {
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

	localPath := filepath.Join(tempDir, ".todos")
	if err := saveStore(localPath, store{}); err != nil {
		t.Fatalf("save local store: %v", err)
	}
	location, err := resolveStoreLocation(storageHere)
	if err != nil {
		t.Fatalf("resolveStoreLocation: %v", err)
	}
	if location.Created {
		t.Fatal("expected existing local store not to be marked as created")
	}
	if canonicalPath(t, location.Path) != canonicalPath(t, localPath) {
		t.Fatalf("expected local path %q, got %q", localPath, location.Path)
	}
	if _, err := os.Stat(localPath); err != nil {
		t.Fatalf("expected local file to exist: %v", err)
	}
	if location.Notice != "" {
		t.Fatalf("expected no creation notice for existing store, got %q", location.Notice)
	}

	s, loadedLocation, err := loadStore(storageHere)
	if err != nil {
		t.Fatalf("loadStore: %v", err)
	}
	if len(s.Items) != 0 {
		t.Fatalf("expected empty local store, got %d items", len(s.Items))
	}
	if loadedLocation.Scope != storeScopeLocal {
		t.Fatalf("expected local scope, got %v", loadedLocation.Scope)
	}
}

func TestConfirmLocalStoreCreationIfNeededAcceptsCreation(t *testing.T) {
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

	var out bytes.Buffer
	if err := confirmLocalStoreCreationIfNeeded(storageHere, strings.NewReader("y\n"), &out); err != nil {
		t.Fatalf("confirmLocalStoreCreationIfNeeded: %v", err)
	}
	if !strings.Contains(out.String(), "Create .todos file? [y/N]:") {
		t.Fatalf("expected confirmation prompt, got %q", out.String())
	}
	location, err := resolveStoreLocation(storageHere)
	if err != nil {
		t.Fatalf("resolveStoreLocation: %v", err)
	}
	if !location.Created {
		t.Fatal("expected newly created local store")
	}
	if _, err := os.Stat(filepath.Join(tempDir, ".todos")); err != nil {
		t.Fatalf("expected local file to be created: %v", err)
	}
}

func TestConfirmLocalStoreCreationIfNeededDeclinesCreation(t *testing.T) {
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

	var out bytes.Buffer
	err = confirmLocalStoreCreationIfNeeded(storageHere, strings.NewReader("n\n"), &out)
	if err == nil {
		t.Fatal("expected declined local store creation to return error")
	}
	if !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("expected cancel error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(tempDir, ".todos")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected local file not to be created, got %v", statErr)
	}
}

func TestResolveStoreLocationHereGitIgnoreNotice(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(filepath.Join(projectDir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir git dir: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir project dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()

	location, err := resolveStoreLocation(storageHere)
	if err != nil {
		t.Fatalf("resolveStoreLocation: %v", err)
	}
	if !strings.Contains(location.Notice, ".gitignore") {
		t.Fatalf("expected gitignore notice, got %q", location.Notice)
	}
	if strings.Contains(location.Notice, projectDir) {
		t.Fatalf("expected relative gitignore path in notice, got %q", location.Notice)
	}
	if !strings.Contains(location.Notice, "\nTo keep local todos out of git") {
		t.Fatalf("expected second-line gitignore guidance, got %q", location.Notice)
	}
	if !strings.Contains(location.Notice, ".todos") {
		t.Fatalf("expected .todos ignore hint, got %q", location.Notice)
	}
}

func TestRenderLocationNoticePreservesNoticeText(t *testing.T) {
	notice := "Created .todos file.\nTo keep local todos out of git, add \".todos\" to .gitignore."
	rendered := renderLocationNotice(notice)

	if !strings.Contains(rendered, "Created .todos file.") {
		t.Fatalf("expected creation line to remain readable, got %q", rendered)
	}
	if rendered != notice {
		t.Fatalf("expected notice rendering to stay plain, got %q", rendered)
	}
	if !strings.Contains(rendered, ".gitignore") {
		t.Fatalf("expected git help text to remain present, got %q", rendered)
	}
}

func TestTitleSourceTextUsesSimpleScopeLabels(t *testing.T) {
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

	if got := titleSourceText(storeLocation{Scope: storeScopeLocal, SourceText: "local .todos"}); got != "local" {
		t.Fatalf("expected local title label, got %q", got)
	}
	if got := titleSourceText(storeLocation{Scope: storeScopeGlobal, SourceText: "global store"}); got != "" {
		t.Fatalf("expected no global title label without local file, got %q", got)
	}
	if err := saveStore(filepath.Join(tempDir, ".todos"), store{}); err != nil {
		t.Fatalf("save local store: %v", err)
	}
	if got := titleSourceText(storeLocation{Scope: storeScopeGlobal, SourceText: "global store"}); got != "global" {
		t.Fatalf("expected global title label, got %q", got)
	}
}

func TestTodoTitleOmitsGlobalWhenNoLocalStoreExists(t *testing.T) {
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

	if got := todoTitle(storeLocation{Scope: storeScopeGlobal, SourceText: "global store"}); got != "Todo:" {
		t.Fatalf("expected plain global title without local store, got %q", got)
	}
}

func TestSwitchScopeFromLocalToGlobal(t *testing.T) {
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

	localPath := filepath.Join(tempDir, ".todos")
	localStore := store{Items: []todo{{Description: "local item"}}, HideDoneInTUI: true}
	if err := saveStore(localPath, localStore); err != nil {
		t.Fatalf("save local store: %v", err)
	}

	globalPath := globalDataPath()
	globalStore := store{Items: []todo{{Description: "global item"}}}
	if err := saveStore(globalPath, globalStore); err != nil {
		t.Fatalf("save global store: %v", err)
	}

	m := newModel(localStore, storeLocation{Path: localPath, Scope: storeScopeLocal, SourceText: "local .todos"})
	m.cursor = 3
	m.pendingG = true
	m.editMode = editModeCurrent
	m.editIndex = 0
	m.input = "editing"
	m.inputCursor = len([]rune(m.input))

	if err := m.switchScope(); err != nil {
		t.Fatalf("switchScope: %v", err)
	}
	if m.location.Scope != storeScopeGlobal {
		t.Fatalf("expected global scope, got %v", m.location.Scope)
	}
	if len(m.store.Items) != 1 || m.store.Items[0].Description != "global item" {
		t.Fatalf("expected global store items, got %#v", m.store.Items)
	}
	if !m.showAll {
		t.Fatal("expected showAll to reflect global store settings")
	}
	if m.cursor != 0 {
		t.Fatalf("expected cursor reset to 0, got %d", m.cursor)
	}
	if m.isEditing() {
		t.Fatal("expected edit mode to be cleared on scope switch")
	}
	if m.pendingG {
		t.Fatal("expected pending g state to be cleared on scope switch")
	}
	if canonicalPath(t, m.location.Path) != canonicalPath(t, globalPath) {
		t.Fatalf("expected global path %q, got %q", globalPath, m.location.Path)
	}
	if m.location.SourceText != "global store" {
		t.Fatalf("expected global source text, got %q", m.location.SourceText)
	}
	if m.location.Notice != "" {
		t.Fatalf("expected no global notice, got %q", m.location.Notice)
	}
	if m.animatingDoneIndex != -1 || m.animatingDoneFrames != 0 || m.animatingDoneCursor != 0 {
		t.Fatalf("expected animation state reset, got index=%d frames=%d cursor=%d", m.animatingDoneIndex, m.animatingDoneFrames, m.animatingDoneCursor)
	}
	_ = os.Remove(globalPath)
}

func TestSwitchScopeFromGlobalToLocalUsesExistingLocalStore(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(filepath.Join(projectDir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir git dir: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir project dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()

	globalPath := globalDataPath()
	globalStore := store{Items: []todo{{Description: "global item"}}}
	if err := saveStore(globalPath, globalStore); err != nil {
		t.Fatalf("save global store: %v", err)
	}
	localPath := filepath.Join(projectDir, ".todos")
	localStore := store{Items: []todo{{Description: "local item"}}, HideDoneInTUI: true}
	if err := saveStore(localPath, localStore); err != nil {
		t.Fatalf("save local store: %v", err)
	}

	m := newModel(globalStore, storeLocation{Path: globalPath, Scope: storeScopeGlobal, SourceText: "global store"})
	if err := m.switchScope(); err != nil {
		t.Fatalf("switchScope: %v", err)
	}

	if m.location.Scope != storeScopeLocal {
		t.Fatalf("expected local scope, got %v", m.location.Scope)
	}
	if canonicalPath(t, m.location.Path) != canonicalPath(t, localPath) {
		t.Fatalf("expected local path %q, got %q", localPath, m.location.Path)
	}
	if len(m.store.Items) != 1 || m.store.Items[0].Description != "local item" {
		t.Fatalf("expected local store items, got %#v", m.store.Items)
	}
	if m.showAll {
		t.Fatal("expected showAll to reflect local store settings")
	}
	if m.location.Created {
		t.Fatal("expected existing local store not to be marked created")
	}
	if m.location.Notice != "" {
		t.Fatalf("expected no creation notice for existing local store, got %q", m.location.Notice)
	}
	if _, err := os.Stat(localPath); err != nil {
		t.Fatalf("expected local store to exist: %v", err)
	}
	_ = os.Remove(globalPath)
}

func TestSwitchScopeFromGlobalPromptsBeforeCreatingLocalStore(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(filepath.Join(projectDir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir git dir: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir project dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()

	globalPath := globalDataPath()
	globalStore := store{Items: []todo{{Description: "global item"}}}
	if err := saveStore(globalPath, globalStore); err != nil {
		t.Fatalf("save global store: %v", err)
	}

	m := newModel(globalStore, storeLocation{Path: globalPath, Scope: storeScopeGlobal, SourceText: "global store"})
	if err := m.switchScope(); err != nil {
		t.Fatalf("switchScope: %v", err)
	}

	localPath := filepath.Join(projectDir, ".todos")
	if !m.confirmCreateLocal {
		t.Fatal("expected confirm-create-local prompt")
	}
	if m.location.Scope != storeScopeGlobal {
		t.Fatalf("expected to stay in global scope, got %v", m.location.Scope)
	}
	if _, err := os.Stat(localPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected local store not to exist yet, got %v", err)
	}
	view := m.View()
	if !strings.Contains(view, "Create .todos file? [y/N]") {
		t.Fatalf("expected confirmation prompt in view, got %q", view)
	}
	_ = os.Remove(globalPath)
}

func TestConfirmLocalScopeSwitchWithEnterCancels(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(filepath.Join(projectDir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir git dir: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir project dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()

	globalPath := globalDataPath()
	globalStore := store{Items: []todo{{Description: "global item"}}}
	if err := saveStore(globalPath, globalStore); err != nil {
		t.Fatalf("save global store: %v", err)
	}

	m := newModel(globalStore, storeLocation{Path: globalPath, Scope: storeScopeGlobal, SourceText: "global store"})
	if err := m.switchScope(); err != nil {
		t.Fatalf("switchScope: %v", err)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected no command when canceling local store creation")
	}
	next, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model result, got %T", updated)
	}
	localPath := filepath.Join(projectDir, ".todos")
	if next.confirmCreateLocal {
		t.Fatal("expected confirm prompt to close after enter")
	}
	if next.location.Scope != storeScopeGlobal {
		t.Fatalf("expected to stay in global scope, got %v", next.location.Scope)
	}
	if _, err := os.Stat(localPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected local store not to exist after enter cancel, got %v", err)
	}
	_ = os.Remove(globalPath)
}

func TestCancelLocalScopeSwitchWithEscStaysGlobal(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(filepath.Join(projectDir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir git dir: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir project dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()

	globalPath := globalDataPath()
	globalStore := store{Items: []todo{{Description: "global item"}}}
	if err := saveStore(globalPath, globalStore); err != nil {
		t.Fatalf("save global store: %v", err)
	}

	m := newModel(globalStore, storeLocation{Path: globalPath, Scope: storeScopeGlobal, SourceText: "global store"})
	if err := m.switchScope(); err != nil {
		t.Fatalf("switchScope: %v", err)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatal("expected no command when cancelling local store creation")
	}
	next, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model result, got %T", updated)
	}
	localPath := filepath.Join(projectDir, ".todos")
	if next.confirmCreateLocal {
		t.Fatal("expected confirm prompt to close after cancel")
	}
	if next.location.Scope != storeScopeGlobal {
		t.Fatalf("expected to stay in global scope, got %v", next.location.Scope)
	}
	if _, err := os.Stat(localPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected local store not to exist after cancel, got %v", err)
	}
	_ = os.Remove(globalPath)
}

func TestConfirmLocalScopeSwitchWithYCreatesLocalStore(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(filepath.Join(projectDir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir git dir: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir project dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()

	globalPath := globalDataPath()
	globalStore := store{Items: []todo{{Description: "global item"}}}
	if err := saveStore(globalPath, globalStore); err != nil {
		t.Fatalf("save global store: %v", err)
	}

	m := newModel(globalStore, storeLocation{Path: globalPath, Scope: storeScopeGlobal, SourceText: "global store"})
	if err := m.switchScope(); err != nil {
		t.Fatalf("switchScope: %v", err)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd != nil {
		t.Fatal("expected no command when confirming with y")
	}
	next, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model result, got %T", updated)
	}
	localPath := filepath.Join(projectDir, ".todos")
	if next.location.Scope != storeScopeLocal {
		t.Fatalf("expected local scope, got %v", next.location.Scope)
	}
	if _, err := os.Stat(localPath); err != nil {
		t.Fatalf("expected local store to exist: %v", err)
	}
	_ = os.Remove(globalPath)
}

func TestCancelLocalScopeSwitchWithNStaysGlobal(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(filepath.Join(projectDir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir git dir: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir project dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()

	globalPath := globalDataPath()
	globalStore := store{Items: []todo{{Description: "global item"}}}
	if err := saveStore(globalPath, globalStore); err != nil {
		t.Fatalf("save global store: %v", err)
	}

	m := newModel(globalStore, storeLocation{Path: globalPath, Scope: storeScopeGlobal, SourceText: "global store"})
	if err := m.switchScope(); err != nil {
		t.Fatalf("switchScope: %v", err)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd != nil {
		t.Fatal("expected no command when cancelling with n")
	}
	next, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model result, got %T", updated)
	}
	localPath := filepath.Join(projectDir, ".todos")
	if next.location.Scope != storeScopeGlobal {
		t.Fatalf("expected to stay in global scope, got %v", next.location.Scope)
	}
	if _, err := os.Stat(localPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected local store not to exist after n cancel, got %v", err)
	}
	_ = os.Remove(globalPath)
}

func TestTabKeyBindingShowsSwitchScope(t *testing.T) {
	help := keys.SwitchView.Help()
	if help.Key != "tab" {
		t.Fatalf("expected tab key help, got %q", help.Key)
	}
	if help.Desc != "switch scope" {
		t.Fatalf("expected switch scope help text, got %q", help.Desc)
	}
	short := keys.ShortHelp()
	if len(short) != 3 {
		t.Fatalf("expected 3 short help bindings, got %d", len(short))
	}
	got := []string{short[0].Help().Key, short[1].Help().Key, short[2].Help().Key}
	want := []string{"x/enter", "?", "q"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected short help %v, got %v", want, got)
		}
	}
	if short[0].Help().Desc != "toggle done" || short[1].Help().Desc != "toggle help" || short[2].Help().Desc != "quit" {
		t.Fatalf("unexpected short help descriptions: %q, %q, %q", short[0].Help().Desc, short[1].Help().Desc, short[2].Help().Desc)
	}
}

func TestEnterStartsAddingFirstItemWhenEmpty(t *testing.T) {
	tempDir := t.TempDir()
	location := storeLocation{Path: filepath.Join(tempDir, ".todos"), Scope: storeScopeLocal, SourceText: "local .todos"}
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
	location := storeLocation{Path: filepath.Join(tempDir, ".todos"), Scope: storeScopeLocal, SourceText: "local .todos"}
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
	location := storeLocation{Path: filepath.Join(tempDir, ".todos"), Scope: storeScopeLocal, SourceText: "local .todos"}
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
	location := storeLocation{Path: filepath.Join(tempDir, ".todos"), Scope: storeScopeLocal, SourceText: "local .todos"}
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
	location := storeLocation{Path: filepath.Join(tempDir, ".todos"), Scope: storeScopeLocal, SourceText: "local .todos"}
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
	location := storeLocation{Path: filepath.Join(tempDir, ".todos"), Scope: storeScopeLocal, SourceText: "local .todos"}
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
	if saved, _, err := loadStore(storageHere); err != nil {
		t.Fatalf("load saved store: %v", err)
	} else if len(saved.Items) != 3 || saved.Items[2].Description != "third" {
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
	location := storeLocation{Path: filepath.Join(tempDir, ".todos"), Scope: storeScopeLocal, SourceText: "local .todos"}
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
	if saved, _, err := loadStore(storageHere); err != nil {
		t.Fatalf("load saved store: %v", err)
	} else if len(saved.Items) != 0 {
		t.Fatalf("expected no persisted item after cancel, got %#v", saved.Items)
	}
}

func TestHideDoneKeepsCursorOnSameOpenTask(t *testing.T) {
	tempDir := t.TempDir()
	location := storeLocation{Path: filepath.Join(tempDir, ".todos"), Scope: storeScopeLocal, SourceText: "local .todos"}
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
	location := storeLocation{Path: filepath.Join(tempDir, ".todos"), Scope: storeScopeLocal, SourceText: "local .todos"}
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
	location := storeLocation{Path: filepath.Join(tempDir, ".todos"), Scope: storeScopeLocal, SourceText: "local .todos"}
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
	location := storeLocation{Path: filepath.Join(tempDir, ".todos"), Scope: storeScopeLocal, SourceText: "local .todos"}
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
	location := storeLocation{Path: filepath.Join(tempDir, ".todos"), Scope: storeScopeLocal, SourceText: "local .todos"}
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
