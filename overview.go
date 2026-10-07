package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ovRow is one line of the overview: either a section header or a todo.
// idx is the position inside the section's full item slice, not the visible one.
type ovRow struct {
	header bool
	title  string
	dir    string // "" is the global list
	idx    int
	item   todo
}

var overviewKey = key.NewBinding(
	key.WithKeys("ctrl+a"),
	key.WithHelp("ctrl+a", "show all"),
)

var (
	listNextKey = key.NewBinding(key.WithKeys("}"), key.WithHelp("}/]]", "next list"))
	listPrevKey = key.NewBinding(key.WithKeys("{"), key.WithHelp("{/[[", "prev list"))
)

type overviewKeyMap struct{}

func (overviewKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{keys.Toggle, keys.EditStart, keys.Delete, keys.ToggleAll, overviewKey, keys.Quit}
}

func (overviewKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{keys.Up, keys.Down, keys.Top, keys.Bottom, listNextKey, listPrevKey},
		{keys.EditStart, keys.EditEnd, keys.OpenBelow, keys.OpenAbove},
		{keys.Toggle, keys.Delete, keys.ClearDone, keys.MoveUp, keys.MoveDown, keys.Undo, keys.Redo},
		{keys.ToggleAll, keys.WrapText, keys.Yank, keys.Paste, keys.OpenLinks},
		{keys.Local, keys.Global, overviewKey, keys.Quit},
	}
}

func shortenPath(dir string) string {
	if home := homeDir(); home != "" {
		if dir == home {
			return "~"
		}
		if strings.HasPrefix(dir, home+string(os.PathSeparator)) {
			return "~" + dir[len(home):]
		}
	}
	return dir
}

// buildOverview orders sections as: current folder, global, other folders.
// Sections without visible items are left out.
func buildOverview(file store, currentDir string, showAll bool) []ovRow {
	type section struct {
		dir   string
		title string
		items []todo
	}

	var others []string
	for dir := range file.Folders {
		if dir != currentDir {
			others = append(others, dir)
		}
	}
	sort.Strings(others)

	var sections []section
	if f, ok := file.Folders[currentDir]; ok {
		sections = append(sections, section{currentDir, shortenPath(currentDir), f.Items})
	}
	sections = append(sections, section{"", "Global", file.Items})
	for _, dir := range others {
		sections = append(sections, section{dir, shortenPath(dir), file.Folders[dir].Items})
	}

	var rows []ovRow
	for _, sec := range sections {
		var items []ovRow
		for i, item := range sec.items {
			if item.Done && !showAll {
				continue
			}
			items = append(items, ovRow{title: sec.title, dir: sec.dir, idx: i, item: item})
		}
		if len(items) == 0 {
			continue
		}
		rows = append(rows, ovRow{header: true, title: fmt.Sprintf("%s (%d)", sec.title, len(items))})
		rows = append(rows, items...)
	}
	return rows
}

func sectionItems(file store, dir string) []todo {
	if dir == "" {
		return file.Items
	}
	return file.Folders[dir].Items
}

func setSectionItems(file *store, dir string, items []todo) {
	if dir == "" {
		file.Items = items
		return
	}
	if file.Folders == nil {
		file.Folders = map[string]folderTodos{}
	}
	if items == nil {
		items = []todo{}
	}
	file.Folders[dir] = folderTodos{Items: items}
}

func countTodos(items []todo) (open, done int) {
	for _, item := range items {
		if item.Done {
			done++
		} else {
			open++
		}
	}
	return open, done
}

func (m *model) refreshOverview() error {
	file, err := readStoreFile(m.location.Path)
	if err != nil {
		return err
	}
	m.ovRows = buildOverview(file, m.location.listDir(), m.showAll)
	m.ovOpen, m.ovDone = countTodos(file.Items)
	for _, f := range file.Folders {
		open, done := countTodos(f.Items)
		m.ovOpen += open
		m.ovDone += done
	}
	m.clampOverviewCursor()
	return nil
}

func (m *model) enterOverview() error {
	if m.location.Dir == "" {
		dir, err := currentDir()
		if err != nil {
			return err
		}
		m.location.Dir = dir
	}
	m.overview = true
	m.clearHistory()
	m.ovCursor = 0
	m.animatingDoneIndex, m.animatingDoneFrames = -1, 0
	m.yankAnimatingIndex, m.yankAnimatingFrames = -1, 0
	return m.refreshOverview()
}

func (m *model) exitOverview() error {
	m.overview = false
	m.clearHistory()
	m.yankAnimatingIndex, m.yankAnimatingFrames = -1, 0
	mode := scopeGlobal
	if m.location.Local {
		mode = scopeLocal
	}
	s, location, err := loadStoreAt(m.location.Path, m.location.Dir, mode)
	if err != nil {
		return err
	}
	m.store = s
	m.location = location
	m.cursor = 0
	m.clampCursor()
	return nil
}

func (m *model) clampOverviewCursor() {
	if len(m.ovRows) == 0 {
		m.ovCursor = 0
		return
	}
	if m.ovCursor >= len(m.ovRows) {
		m.ovCursor = len(m.ovRows) - 1
	}
	if m.ovCursor < 0 {
		m.ovCursor = 0
	}
	// Headers are not selectable: prefer the next todo, else the previous one.
	for i := m.ovCursor; i < len(m.ovRows); i++ {
		if !m.ovRows[i].header {
			m.ovCursor = i
			return
		}
	}
	for i := m.ovCursor; i >= 0; i-- {
		if !m.ovRows[i].header {
			m.ovCursor = i
			return
		}
	}
}

func (m *model) moveOverviewCursor(delta int) {
	for i := m.ovCursor + delta; i >= 0 && i < len(m.ovRows); i += delta {
		if !m.ovRows[i].header {
			m.ovCursor = i
			return
		}
	}
}

// mutateOverviewItem applies fn to the selected todo's list and writes the
// whole file back, so the overview never works from stale data.
func (m *model) mutateOverviewItem(verb string, fn func(items []todo, idx int) []todo) error {
	if m.ovCursor >= len(m.ovRows) || m.ovRows[m.ovCursor].header {
		return nil
	}
	row := m.ovRows[m.ovCursor]

	file, err := readStoreFile(m.location.Path)
	if err != nil {
		return err
	}
	items := sectionItems(file, row.dir)
	m.pushUndo(row.dir, items, row.idx, verb+" "+quoted(row.item.Description))
	setSectionItems(&file, row.dir, fn(items, row.idx))
	if err := writeStoreFile(m.location.Path, file); err != nil {
		return err
	}
	return m.refreshOverview()
}

// toggleOverviewDone flips and persists the shared hide-done preference,
// keeping the cursor on the same todo when it stays visible.
func (m *model) toggleOverviewDone() error {
	var selected *ovRow
	if m.ovCursor < len(m.ovRows) && !m.ovRows[m.ovCursor].header {
		selected = &m.ovRows[m.ovCursor]
	}

	m.showAll = !m.showAll
	m.store.HideDoneInTUI = !m.showAll

	file, err := readStoreFile(m.location.Path)
	if err != nil {
		return err
	}
	file.HideDoneInTUI = m.store.HideDoneInTUI
	if err := writeStoreFile(m.location.Path, file); err != nil {
		return err
	}

	prev := m.ovCursor
	if err := m.refreshOverview(); err != nil {
		return err
	}
	if selected != nil {
		for i, r := range m.ovRows {
			if !r.header && r.dir == selected.dir && r.idx == selected.idx {
				m.ovCursor = i
				return nil
			}
		}
	}
	m.ovCursor = prev
	m.clampOverviewCursor()
	return nil
}

// activeListDir is where a new todo goes when the overview has no rows to anchor on.
func (m model) activeListDir() string {
	if m.location.Local {
		return m.location.listDir()
	}
	return ""
}

func (m *model) startOverviewEdit(mode editMode, atEnd bool) {
	var row *ovRow
	if m.ovCursor < len(m.ovRows) && !m.ovRows[m.ovCursor].header {
		row = &m.ovRows[m.ovCursor]
	}
	if row == nil && mode == editModeCurrent {
		return
	}

	m.editMode = mode
	m.input = ""
	m.inputCursor = 0
	m.directionalNewItem = 0
	m.editIndex = -1
	m.insertAt = -1
	m.ovEditRow = m.ovCursor
	m.ovEditDir = m.activeListDir()
	if row == nil {
		m.ovEditRow = -1
		return
	}

	m.ovEditDir = row.dir
	switch mode {
	case editModeCurrent:
		m.editIndex = row.idx
		m.input = row.item.Description
		if atEnd {
			m.inputCursor = len([]rune(m.input))
		}
	case editModeNewBelow:
		m.insertAt = row.idx + 1
	case editModeNewAbove:
		m.insertAt = row.idx
	}
}

func (m *model) commitOverviewInput() error {
	description := strings.TrimSpace(m.input)
	if description == "" {
		return nil
	}

	file, err := readStoreFile(m.location.Path)
	if err != nil {
		return err
	}

	dir := m.ovEditDir
	items := sectionItems(file, dir)
	at := m.editIndex
	if m.editMode == editModeCurrent && at >= 0 && at < len(items) {
		m.pushUndo(dir, items, at, "edit "+quoted(description))
		items[at].Description = description
	} else {
		at = m.insertAt
		if at < 0 || at > len(items) {
			at = len(items)
		}
		m.pushUndo(dir, items, at, "add "+quoted(description))
		items = append(items, todo{})
		copy(items[at+1:], items[at:])
		items[at] = todo{Description: description, CreatedAt: time.Now()}
	}
	setSectionItems(&file, dir, items)
	if err := writeStoreFile(m.location.Path, file); err != nil {
		return err
	}

	m.cancelEdit()
	if err := m.refreshOverview(); err != nil {
		return err
	}
	m.selectRow(dir, at)
	return nil
}

func (m *model) selectRow(dir string, idx int) {
	for i, r := range m.ovRows {
		if !r.header && r.dir == dir && r.idx == idx {
			m.ovCursor = i
			return
		}
	}
}

func (m model) selectedRow() *ovRow {
	if m.ovCursor < len(m.ovRows) && !m.ovRows[m.ovCursor].header {
		return &m.ovRows[m.ovCursor]
	}
	return nil
}

func (m *model) yankOverviewItem() tea.Cmd {
	row := m.selectedRow()
	if row == nil || writeClipboard(row.item.Description) != nil {
		return nil
	}
	m.yankAnimatingIndex = m.ovCursor
	m.yankAnimatingFrames = 2
	return nextYankFrame()
}

func (m *model) pasteOverviewItem() error {
	text, err := readClipboard()
	if err != nil || strings.TrimSpace(text) == "" {
		return nil
	}

	file, err := readStoreFile(m.location.Path)
	if err != nil {
		return err
	}

	dir, at := m.activeListDir(), -1
	items := sectionItems(file, dir)
	if row := m.selectedRow(); row != nil {
		dir, at = row.dir, row.idx+1
		items = sectionItems(file, dir)
	}
	if at < 0 || at > len(items) {
		at = len(items)
	}

	m.pushUndo(dir, items, at, "paste "+quoted(strings.TrimSpace(text)))
	items = append(items, todo{})
	copy(items[at+1:], items[at:])
	items[at] = todo{Description: strings.TrimSpace(text), CreatedAt: time.Now()}
	setSectionItems(&file, dir, items)
	if err := writeStoreFile(m.location.Path, file); err != nil {
		return err
	}
	if err := m.refreshOverview(); err != nil {
		return err
	}
	m.selectRow(dir, at)
	return nil
}

// moveOverviewItem swaps with the neighbouring todo, but never across lists.
func (m *model) moveOverviewItem(delta int) error {
	row := m.selectedRow()
	next := m.ovCursor + delta
	if row == nil || next < 0 || next >= len(m.ovRows) || m.ovRows[next].header || m.ovRows[next].dir != row.dir {
		return nil
	}
	other := m.ovRows[next]

	file, err := readStoreFile(m.location.Path)
	if err != nil {
		return err
	}
	items := sectionItems(file, row.dir)
	if row.idx >= len(items) || other.idx >= len(items) {
		return nil
	}
	m.pushUndo(row.dir, items, row.idx, "move "+quoted(row.item.Description))
	items[row.idx], items[other.idx] = items[other.idx], items[row.idx]
	setSectionItems(&file, row.dir, items)
	if err := writeStoreFile(m.location.Path, file); err != nil {
		return err
	}
	if err := m.refreshOverview(); err != nil {
		return err
	}
	m.selectRow(row.dir, other.idx)
	return nil
}

// clearOverviewDone only touches the list under the cursor, so one keypress
// can't wipe finished todos from every project.
func (m *model) clearOverviewDone() error {
	dir := m.activeListDir()
	if row := m.selectedRow(); row != nil {
		dir = row.dir
	}

	file, err := readStoreFile(m.location.Path)
	if err != nil {
		return err
	}
	items := sectionItems(file, dir)
	kept := []todo{}
	for _, item := range items {
		if !item.Done {
			kept = append(kept, item)
		}
	}
	if len(kept) == len(items) {
		return nil
	}
	m.pushUndo(dir, items, 0, "delete all done")
	setSectionItems(&file, dir, kept)
	if err := writeStoreFile(m.location.Path, file); err != nil {
		return err
	}
	return m.refreshOverview()
}

// jumpList moves like vim's paragraph motions: forward to the first todo of the
// next list, backward to the start of the current list, then to earlier lists.
func (m *model) jumpList(delta int) {
	var starts []int
	for i, r := range m.ovRows {
		if r.header && i+1 < len(m.ovRows) {
			starts = append(starts, i+1)
		}
	}
	if len(starts) == 0 {
		return
	}

	if delta > 0 {
		for _, s := range starts {
			if s > m.ovCursor {
				m.ovCursor = s
				return
			}
		}
		m.ovCursor = len(m.ovRows) - 1
		return
	}
	for i := len(starts) - 1; i >= 0; i-- {
		if starts[i] < m.ovCursor {
			m.ovCursor = starts[i]
			return
		}
	}
	m.ovCursor = starts[0]
}

func (m *model) toggleOverviewItem() error {
	verb := "complete"
	if row := m.selectedRow(); row != nil && row.item.Done {
		verb = "reopen"
	}
	return m.mutateOverviewItem(verb, func(items []todo, idx int) []todo {
		if idx >= len(items) {
			return items
		}
		items[idx].Done = !items[idx].Done
		items[idx].DoneAt = time.Time{}
		if items[idx].Done {
			items[idx].DoneAt = time.Now()
		}
		return items
	})
}

func (m *model) deleteOverviewItem() error {
	return m.mutateOverviewItem("delete", func(items []todo, idx int) []todo {
		if idx >= len(items) {
			return items
		}
		return append(items[:idx], items[idx+1:]...)
	})
}

func (m model) updateOverview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	fail := func(err error) (tea.Model, tea.Cmd) {
		m.err = err
		return m, tea.Quit
	}

	pendingG := m.pendingG
	pendingBracket := m.pendingBracket
	m.pendingG = false
	m.pendingBracket = ""

	switch msg.String() {
	case "[", "]":
		if pendingBracket == msg.String() {
			m.jumpList(map[string]int{"]": 1, "[": -1}[msg.String()])
		} else {
			m.pendingBracket = msg.String()
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, listNextKey):
		m.jumpList(1)
	case key.Matches(msg, listPrevKey):
		m.jumpList(-1)
	case key.Matches(msg, keys.EditStart):
		m.startOverviewEdit(editModeCurrent, false)
	case key.Matches(msg, keys.EditEnd):
		m.startOverviewEdit(editModeCurrent, true)
	case key.Matches(msg, keys.OpenBelow):
		m.startOverviewEdit(editModeNewBelow, false)
	case key.Matches(msg, keys.OpenAbove):
		m.startOverviewEdit(editModeNewAbove, false)
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, overviewKey), msg.Type == tea.KeyEsc:
		if err := m.exitOverview(); err != nil {
			return fail(err)
		}
	case key.Matches(msg, keys.Local):
		if err := m.exitOverview(); err != nil {
			return fail(err)
		}
		if err := m.switchScope(scopeLocal); err != nil {
			return fail(err)
		}
	case key.Matches(msg, keys.Global):
		if err := m.exitOverview(); err != nil {
			return fail(err)
		}
		if err := m.switchScope(scopeGlobal); err != nil {
			return fail(err)
		}
	case key.Matches(msg, keys.Help):
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
	case key.Matches(msg, keys.Up):
		m.moveOverviewCursor(-1)
	case key.Matches(msg, keys.Down):
		m.moveOverviewCursor(1)
	case key.Matches(msg, keys.Top):
		if !pendingG {
			m.pendingG = true
			break
		}
		m.ovCursor = 0
		m.clampOverviewCursor()
	case key.Matches(msg, keys.Bottom):
		m.ovCursor = len(m.ovRows) - 1
		m.clampOverviewCursor()
	case key.Matches(msg, keys.ToggleAll):
		if err := m.toggleOverviewDone(); err != nil {
			return fail(err)
		}
	case key.Matches(msg, keys.WrapText):
		m.wrapText = !m.wrapText
	case key.Matches(msg, keys.Yank):
		return m, m.yankOverviewItem()
	case key.Matches(msg, keys.Paste):
		if err := m.pasteOverviewItem(); err != nil {
			return fail(err)
		}
	case key.Matches(msg, keys.OpenLinks):
		if row := m.selectedRow(); row != nil {
			return m, openLinksCmd(row.item.Description)
		}
	case key.Matches(msg, keys.MoveUp):
		if err := m.moveOverviewItem(-1); err != nil {
			return fail(err)
		}
	case key.Matches(msg, keys.MoveDown):
		if err := m.moveOverviewItem(1); err != nil {
			return fail(err)
		}
	case key.Matches(msg, keys.ClearDone):
		if err := m.clearOverviewDone(); err != nil {
			return fail(err)
		}
	case key.Matches(msg, keys.Toggle):
		if err := m.toggleOverviewItem(); err != nil {
			return fail(err)
		}
	case key.Matches(msg, keys.Delete):
		if err := m.deleteOverviewItem(); err != nil {
			return fail(err)
		}
	case key.Matches(msg, keys.Undo):
		if err := m.undo(); err != nil {
			return fail(err)
		}
	case key.Matches(msg, keys.Redo):
		if err := m.redo(); err != nil {
			return fail(err)
		}
	}
	return m, nil
}

func (m model) overviewView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Todo: all lists"))
	b.WriteString("\n")

	b.WriteString(subtitleStyle.Render(formatStatus(m.ovOpen, m.ovDone, "all", m.showAll)))
	b.WriteString(m.noticeLine())
	b.WriteString("\n\n")

	inputRow := func() string {
		checkbox := openBoxStyle.Background(background).Render("[ ]")
		return renderInputRow(m.input, m.inputCursor, checkbox, m.contentWidth()) + "\n"
	}

	if len(m.ovRows) == 0 {
		if m.isEditing() {
			b.WriteString(inputRow())
			b.WriteString(mutedStyle.Render(m.editModeHelp()))
		} else {
			b.WriteString(mutedStyle.Render("No open todos in any list. Press o to add one."))
		}
		b.WriteString("\n\n")
		b.WriteString(m.help.View(overviewKeyMap{}))
		return appStyle(m.width).Render(b.String())
	}

	contentWidth := m.contentWidth()
	now := time.Now()
	for i, row := range m.ovRows {
		if row.header {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(accentStyle.Render(row.title))
			b.WriteString("\n")
			continue
		}

		if m.editMode == editModeNewAbove && i == m.ovEditRow {
			b.WriteString(inputRow())
		}
		if m.editMode == editModeCurrent && i == m.ovEditRow {
			b.WriteString(inputRow())
			continue
		}

		selected := i == m.ovCursor && !m.isEditingNewItem()
		cursor := "  "
		if selected {
			cursor = accentStyle.Render("->")
		}
		timestamp := m.rowTimestamp(row.item, now)
		if m.wrapText {
			checkbox, textLines := m.rowWrappedAppearance(i, row.item.Description, row.item.Done, selected, contentWidth, timestamp)
			prefix := fmt.Sprintf("%s %s ", cursor, checkbox)
			indent := strings.Repeat(" ", lipgloss.Width(prefix))
			b.WriteString(prefix + textLines[0] + "\n")
			for _, tl := range textLines[1:] {
				b.WriteString(indent + tl + "\n")
			}
			if timestamp != "" {
				b.WriteString(indent + renderTimestampText(timestamp, selected) + "\n")
			}
			if m.editMode == editModeNewBelow && i == m.ovEditRow {
				b.WriteString(inputRow())
			}
			continue
		}
		checkbox, text := m.rowAppearance(i, row.item.Description, row.item.Done, selected, contentWidth, timestamp)
		left := fmt.Sprintf("%s %s %s", cursor, checkbox, text)
		b.WriteString(renderAlignedRow(left, timestamp, contentWidth, selected))
		b.WriteString("\n")
		if m.editMode == editModeNewBelow && i == m.ovEditRow {
			b.WriteString(inputRow())
		}
	}

	b.WriteString("\n")
	if m.isEditing() {
		b.WriteString(mutedStyle.Render(m.editModeHelp()))
		b.WriteString("\n\n")
	}
	b.WriteString(m.help.View(overviewKeyMap{}))
	return appStyle(m.width).Render(b.String())
}
