package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type todo struct {
	Description string    `json:"description"`
	Done        bool      `json:"done"`
	CreatedAt   time.Time `json:"created_at"`
	DoneAt      time.Time `json:"done_at,omitempty"`
}

type store struct {
	Items            []todo `json:"items"`
	HideDoneInTUI    bool   `json:"hide_done_in_tui,omitempty"`
}

type addPosition int

type completionFrameMsg struct{}

type editMode int

type actionMode int

type storageMode int

type storeScope int

const (
	addBottom addPosition = iota
	addTop
)

const (
	editModeNone editMode = iota
	editModeNewAbove
	editModeNewBelow
	editModeCurrent
)

const (
	actionInteractive actionMode = iota
	actionAdd
	actionList
	actionListAll
	actionDelete
	actionHelp
)

const (
	storageAuto storageMode = iota
	storageHere
	storageGlobal
)

const (
	storeScopeLocal storeScope = iota
	storeScopeGlobal
)

type runOptions struct {
	action      actionMode
	position    addPosition
	storage     storageMode
	addArgs     []string
	sawPosition bool
	deleteIndex int
}

type storeLocation struct {
	Path       string
	Scope      storeScope
	SourceText string
	Created    bool
	Notice     string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	opts, err := parseArgs(args)
	if err != nil {
		return err
	}

	if opts.action == actionHelp {
		printHelp()
		return nil
	}

	if err := confirmLocalStoreCreationIfNeeded(opts.storage, os.Stdin, os.Stdout); err != nil {
		return err
	}

	s, location, err := loadStore(opts.storage)
	if err != nil {
		return err
	}

	switch opts.action {
	case actionList:
		return printTodos(s, false, location)
	case actionListAll:
		return printTodos(s, true, location)
	case actionAdd:
		return runAdd(opts.addArgs, opts.position, s, location)
	case actionDelete:
		return runDelete(opts.deleteIndex, s, location)
	default:
		return runInteractive(s, location)
	}
}

func parseArgs(args []string) (runOptions, error) {
	opts := runOptions{
		action:   actionInteractive,
		position: addBottom,
		storage:  storageAuto,
		addArgs:  make([]string, 0, len(args)),
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-l", "--list":
			if len(opts.addArgs) > 0 {
				return runOptions{}, errors.New("list flag cannot be combined with todo text")
			}
			if opts.sawPosition {
				return runOptions{}, errors.New("list flag cannot be combined with add position flags")
			}
			if opts.action != actionInteractive {
				return runOptions{}, errors.New("list flag cannot be combined with other action flags")
			}
			opts.action = actionList
		case "-la", "--list-all":
			if len(opts.addArgs) > 0 {
				return runOptions{}, errors.New("list-all flag cannot be combined with todo text")
			}
			if opts.sawPosition {
				return runOptions{}, errors.New("list-all flag cannot be combined with add position flags")
			}
			if opts.action != actionInteractive {
				return runOptions{}, errors.New("list-all flag cannot be combined with other action flags")
			}
			opts.action = actionListAll
		case "help", "-h", "--help":
			if len(args) != 1 {
				return runOptions{}, errors.New("help flag cannot be combined with other arguments")
			}
			opts.action = actionHelp
		case "-d", "--delete":
			if opts.action != actionInteractive {
				return runOptions{}, errors.New("delete flag cannot be combined with other action flags")
			}
			if len(opts.addArgs) > 0 {
				return runOptions{}, errors.New("delete flag cannot be combined with todo text")
			}
			if opts.sawPosition {
				return runOptions{}, errors.New("delete flag cannot be combined with add position flags")
			}
			if i+1 >= len(args) {
				return runOptions{}, errors.New("delete flag requires an index argument")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 1 {
				return runOptions{}, fmt.Errorf("delete index must be a positive integer, got %q", args[i])
			}
			opts.action = actionDelete
			opts.deleteIndex = n
		case "-t", "--top":
			if opts.action == actionList || opts.action == actionListAll {
				return runOptions{}, errors.New("add position flags cannot be combined with list flags")
			}
			if opts.action == actionDelete {
				return runOptions{}, errors.New("add position flags cannot be combined with delete flag")
			}
			opts.position = addTop
			opts.sawPosition = true
		case "-b", "--bottom":
			if opts.action == actionList || opts.action == actionListAll {
				return runOptions{}, errors.New("add position flags cannot be combined with list flags")
			}
			if opts.action == actionDelete {
				return runOptions{}, errors.New("add position flags cannot be combined with delete flag")
			}
			opts.position = addBottom
			opts.sawPosition = true
		case "-H", "--here":
			if opts.storage == storageGlobal {
				return runOptions{}, errors.New("here and global flags cannot be combined")
			}
			opts.storage = storageHere
		case "-g", "--global":
			if opts.storage == storageHere {
				return runOptions{}, errors.New("here and global flags cannot be combined")
			}
			opts.storage = storageGlobal
		default:
			if strings.HasPrefix(arg, "-") {
				return runOptions{}, fmt.Errorf("unknown flag: %s", arg)
			}
			if opts.action == actionList {
				return runOptions{}, errors.New("list flag cannot be combined with todo text")
			}
			if opts.action == actionListAll {
				return runOptions{}, errors.New("list-all flag cannot be combined with todo text")
			}
			if opts.action == actionDelete {
				return runOptions{}, errors.New("delete flag cannot be combined with todo text")
			}
			opts.addArgs = append(opts.addArgs, arg)
		}
	}

	if len(opts.addArgs) > 0 || opts.sawPosition {
		opts.action = actionAdd
	}

	return opts, nil
}

func runAdd(args []string, position addPosition, s store, location storeLocation) error {
	description := strings.TrimSpace(strings.Join(args, " "))
	if description == "" {
		return errors.New("todo description cannot be empty")
	}

	item := todo{
		Description: description,
		CreatedAt:   time.Now(),
	}
	if position == addTop {
		s.Items = append([]todo{item}, s.Items...)
	} else {
		s.Items = append(s.Items, item)
	}

	if err := saveStore(location.Path, s); err != nil {
		return err
	}

	return printTodos(s, false, location)
}

func runDelete(index int, s store, location storeLocation) error {
	open := make([]int, 0, len(s.Items))
	for i, item := range s.Items {
		if !item.Done {
			open = append(open, i)
		}
	}

	if index < 1 || index > len(open) {
		return fmt.Errorf("no item %d (have %d open)", index, len(open))
	}

	itemIndex := open[index-1]
	s.Items = append(s.Items[:itemIndex], s.Items[itemIndex+1:]...)

	if err := saveStore(location.Path, s); err != nil {
		return err
	}

	return printTodos(s, false, location)
}

func printTodos(s store, showAll bool, location storeLocation) error {
	visible := make([]todo, 0, len(s.Items))
	for _, item := range s.Items {
		if !showAll && item.Done {
			continue
		}
		visible = append(visible, item)
	}

	count := len(visible)
	indexWidth := len(fmt.Sprintf("%d", count))
	if indexWidth < 1 {
		indexWidth = 1
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(todoTitle(location)))
	b.WriteString("\n")
	if location.Notice != "" {
		b.WriteString(subtitleStyle.Render(renderLocationNotice(location.Notice)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	now := time.Now()
	listWidth := 0
	for i, item := range visible {
		index := accentStyle.Render(fmt.Sprintf("%*d.", indexWidth, i+1))
		prefix := fmt.Sprintf("%s %s", index, item.Description)
		if showAll {
			checkbox := openBoxStyle.Render("[ ]")
			text := item.Description
			if item.Done {
				checkbox = doneBoxStyle.Render("[✓]")
				text = doneTextStyle.Render(text)
			}
			prefix = fmt.Sprintf("%s %s %s", index, checkbox, text)
		}
		line := renderAlignedRow(prefix, taskTimestampText(item, now), listWidth, false)
		b.WriteString(line)
		b.WriteString("\n")
	}

	if count == 0 {
		if showAll {
			b.WriteString(mutedStyle.Render("no todos"))
		} else {
			b.WriteString(mutedStyle.Render("no open todos"))
		}
	} else {
		b.WriteString("\n")
		if showAll {
			done := 0
			for _, item := range s.Items {
				if item.Done {
					done++
				}
			}
			b.WriteString(subtitleStyle.Render(fmt.Sprintf("%d items, %d done", count, done)))
		} else {
			b.WriteString(subtitleStyle.Render(fmt.Sprintf("%d open", count)))
		}
	}

	fmt.Println(appStyle(0).Render(strings.TrimRight(b.String(), "\n")))
	return nil
}

func runInteractive(s store, location storeLocation) error {
	m := newModel(s, location)
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	result, ok := finalModel.(model)
	if !ok {
		return nil
	}

	if result.err != nil {
		return result.err
	}

	return nil
}

func printHelp() {
	cmd := commandName()
	fmt.Printf("%s - simple terminal todo app\n", cmd)
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  %s                 open interactive mode\n", cmd)
	fmt.Printf("  %s --help          show help\n", cmd)
	fmt.Printf("  %s -H              create and use ./.todos\n", cmd)
	fmt.Printf("  %s -g              force the global todo store\n", cmd)
	fmt.Printf("  %s \"buy milk\"      add a todo at the bottom\n", cmd)
	fmt.Printf("  %s -t \"buy milk\"   add a todo at the top\n", cmd)
	fmt.Printf("  %s -l              list open todos\n", cmd)
	fmt.Printf("  %s -la             list all todos\n", cmd)
	fmt.Printf("  %s -d 2            delete open todo #2\n", cmd)
	fmt.Printf("  %s -H -l           list local todos from ./.todos\n", cmd)
	fmt.Printf("  %s -g -l           list global todos\n", cmd)
	fmt.Println()
	fmt.Println("Interactive controls:")
	fmt.Println("  j/down   move down")
	fmt.Println("  k/up     move up")
	fmt.Println("  gg       jump top")
	fmt.Println("  G        jump bottom")
	fmt.Println("  i        edit from start")
	fmt.Println("  a        edit from end")
	fmt.Println("  o        new below")
	fmt.Println("  O        new above")
	fmt.Println("  J/S-down move item down")
	fmt.Println("  K/S-up   move item up")
	fmt.Println("  x/enter  toggle done")
	fmt.Println("  d        delete item")
	fmt.Println("  D        delete all done")
	fmt.Println("  H        hide done")
	fmt.Println("  q        quit")
	fmt.Println()
	fmt.Printf("Global data file: %s\n", globalDataPath())
	fmt.Println("Local project file: ./.todos")
}

func renderLocationNotice(notice string) string {
	return notice
}

func todoTitle(location storeLocation) string {
	source := titleSourceText(location)
	if source == "" {
		return "Todo:"
	}
	return fmt.Sprintf("Todo (%s):", source)
}

func titleSourceText(location storeLocation) string {
	if location.Scope == storeScopeLocal {
		return "local"
	}

	localPath, err := localDataPath()
	if err != nil {
		return ""
	}
	if _, err := os.Stat(localPath); err == nil {
		return "global"
	}

	return ""
}

func commandName() string {
	name := filepath.Base(os.Args[0])
	if name == "" || name == "." {
		return "td"
	}
	return name
}

func globalDataPath() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return ".td.json"
	}
	return filepath.Join(base, "td", "todos.json")
}

func localDataPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve current directory: %w", err)
	}
	return filepath.Join(cwd, ".todos"), nil
}

func confirmLocalStoreCreationIfNeeded(mode storageMode, in io.Reader, out io.Writer) error {
	if mode != storageHere {
		return nil
	}

	path, err := localDataPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check local todos: %w", err)
	}

	ok, err := confirmLocalStoreCreation(in, out)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("local .todos creation canceled")
	}

	return nil
}

func confirmLocalStoreCreation(in io.Reader, out io.Writer) (bool, error) {
	if _, err := fmt.Fprint(out, "Create .todos file? [y/N]: "); err != nil {
		return false, fmt.Errorf("write confirmation prompt: %w", err)
	}

	response, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

func loadStore(mode storageMode) (store, storeLocation, error) {
	location, err := resolveStoreLocation(mode)
	if err != nil {
		return store{}, storeLocation{}, err
	}

	data, err := os.ReadFile(location.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return store{}, location, nil
		}
		return store{}, storeLocation{}, fmt.Errorf("read todos: %w", err)
	}

	var s store
	if err := json.Unmarshal(data, &s); err != nil {
		return store{}, storeLocation{}, fmt.Errorf("parse todos: %w", err)
	}

	return s, location, nil
}

func resolveStoreLocation(mode storageMode) (storeLocation, error) {
	localPath, err := localDataPath()
	if err != nil {
		return storeLocation{}, err
	}

	switch mode {
	case storageHere:
		created, notice, err := ensureStoreFile(localPath)
		if err != nil {
			return storeLocation{}, err
		}
		return storeLocation{Path: localPath, Scope: storeScopeLocal, SourceText: "local .todos", Created: created, Notice: notice}, nil
	case storageGlobal:
		return storeLocation{Path: globalDataPath(), Scope: storeScopeGlobal, SourceText: "global store"}, nil
	default:
		if _, err := os.Stat(localPath); err == nil {
			return storeLocation{Path: localPath, Scope: storeScopeLocal, SourceText: "local .todos"}, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return storeLocation{}, fmt.Errorf("check local todos: %w", err)
		}
		return storeLocation{Path: globalDataPath(), Scope: storeScopeGlobal, SourceText: "global store"}, nil
	}
}

func ensureStoreFile(path string) (bool, string, error) {
	if _, err := os.Stat(path); err == nil {
		return false, "", nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, "", fmt.Errorf("check todos file: %w", err)
	}

	if err := saveStore(path, store{}); err != nil {
		return false, "", err
	}

	return true, localStoreNotice(path), nil
}

func localStoreNotice(path string) string {
	gitRoot, ok := gitRoot(filepath.Dir(path))
	if !ok {
		return fmt.Sprintf("Created %s file.", path)
	}

	displayPath := gitignoreEntry(gitRoot, path)
	if strings.HasPrefix(displayPath, "/") {
		displayPath = strings.TrimPrefix(displayPath, "/")
	}
	ignoreEntry := gitignoreEntry(gitRoot, path)
	return fmt.Sprintf("Created %s file.\nTo keep local todos out of git, add %q to .gitignore.", displayPath, ignoreEntry)
}

func gitRoot(startDir string) (string, bool) {
	dir := filepath.Clean(startDir)
	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil {
			_ = info
			return dir, true
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", false
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func gitignoreEntry(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.Base(path)
	}
	rel = filepath.ToSlash(rel)
	if rel == ".todos" {
		return ".todos"
	}
	return "/" + rel
}

func saveStore(path string, s store) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode todos: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write todos: %w", err)
	}

	return nil
}

type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Top        key.Binding
	Bottom     key.Binding
	SwitchView key.Binding
	MoveUp     key.Binding
	MoveDown   key.Binding
	EditStart  key.Binding
	EditEnd    key.Binding
	OpenBelow  key.Binding
	OpenAbove  key.Binding
	Toggle     key.Binding
	Delete     key.Binding
	ClearDone  key.Binding
	ToggleAll  key.Binding
	Quit       key.Binding
	Cancel     key.Binding
	Help       key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Toggle, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down, k.Top, k.Bottom}, {k.EditStart, k.EditEnd, k.OpenBelow, k.OpenAbove}, {k.Toggle, k.Delete, k.ClearDone, k.MoveUp}, {k.MoveDown, k.ToggleAll, k.SwitchView, k.Help}, {k.Cancel, k.Quit}}
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/up", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/down", "move down"),
	),
	Top: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("gg", "jump top"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "jump bottom"),
	),
	SwitchView: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch scope"),
	),
	MoveUp: key.NewBinding(
		key.WithKeys("K", "shift+up"),
		key.WithHelp("K/S-up", "move item up"),
	),
	MoveDown: key.NewBinding(
		key.WithKeys("J", "shift+down"),
		key.WithHelp("J/S-down", "move item down"),
	),
	EditStart: key.NewBinding(
		key.WithKeys("i", "I"),
		key.WithHelp("i", "edit from start"),
	),
	EditEnd: key.NewBinding(
		key.WithKeys("a", "A"),
		key.WithHelp("a", "edit from end"),
	),
	OpenBelow: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "new below"),
	),
	OpenAbove: key.NewBinding(
		key.WithKeys("O"),
		key.WithHelp("O", "new above"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("x", "enter", " "),
		key.WithHelp("x/enter", "toggle done"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete item"),
	),
	ClearDone: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "delete all done"),
	),
	ToggleAll: key.NewBinding(
		key.WithKeys("H"),
		key.WithHelp("H", "show all/open"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel add"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
}

type model struct {
	store                 store
	location              storeLocation
	cursor                int
	showAll               bool
	confirmCreateLocal    bool
	directionalNewItem     int
	help                  help.Model
	showHelp              bool
	editMode              editMode
	input                 string
	inputCursor           int
	insertAt              int
	editIndex             int
	pendingG              bool
	animatingDoneIndex    int
	animatingDoneFrames   int
	animatingDoneCursor   int
	err                   error
	width                 int
	height                int
}

func newModel(s store, location storeLocation) model {
	h := help.New()
	h.ShowAll = false
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

	return model{
		store:              s,
		location:           location,
		showAll:            !s.HideDoneInTUI,
		help:               h,
		insertAt:           -1,
		editIndex:          -1,
		animatingDoneIndex: -1,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case completionFrameMsg:
		if m.animatingDoneFrames > 0 {
			m.animatingDoneFrames--
			if m.animatingDoneFrames > 0 {
				return m, nextCompletionFrame()
			}
			m.cursor = m.animatingDoneCursor
			m.clampCursor()
			m.animatingDoneIndex = -1
		}
	case tea.KeyMsg:
		if m.confirmCreateLocal {
			switch msg.Type {
			case tea.KeyEnter:
				m.confirmCreateLocal = false
			case tea.KeyEsc:
				m.confirmCreateLocal = false
			default:
				if msg.Type == tea.KeyRunes {
					switch strings.ToLower(msg.String()) {
					case "y":
						if err := m.confirmLocalScopeSwitch(); err != nil {
							m.err = err
							return m, tea.Quit
						}
					case "n":
						m.confirmCreateLocal = false
					}
				}
			}
			return m, nil
		}

		if m.isEditing() {
			if m.isDirectionalNewItemEdit() {
				switch msg.Type {
				case tea.KeyUp:
					return m.handleDirectionalEditKey(-1)
				case tea.KeyDown:
					return m.handleDirectionalEditKey(1)
				}
			}

			switch msg.Type {
			case tea.KeyEsc:
				m.cancelEdit()
				return m, nil
			case tea.KeyEnter:
				if err := m.commitInput(); err != nil {
					m.err = err
					return m, tea.Quit
				}
				return m, nil
			case tea.KeySpace:
				m.insertInput(" ")
				return m, nil
			case tea.KeyLeft:
				if m.inputCursor > 0 {
					m.inputCursor--
				}
				return m, nil
			case tea.KeyRight:
				if m.inputCursor < len([]rune(m.input)) {
					m.inputCursor++
				}
				return m, nil
			case tea.KeyBackspace, tea.KeyCtrlH:
				m.backspaceInput()
				return m, nil
			default:
				if msg.Type == tea.KeyRunes {
					m.insertInput(msg.String())
				}
				return m, nil
			}
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Help):
			m.pendingG = false
			m.showHelp = !m.showHelp
			m.help.ShowAll = m.showHelp
		case msg.Type == tea.KeyEnter && len(m.store.Items) == 0:
			m.pendingG = false
			m.startNewItem(true)
		case key.Matches(msg, keys.Top):
			if m.pendingG {
				m.cursor = 0
				m.pendingG = false
				break
			}
			m.pendingG = true
		case key.Matches(msg, keys.Bottom):
			m.pendingG = false
			visible := m.visibleIndexes()
			if len(visible) > 0 {
				m.cursor = len(visible) - 1
			}
		case key.Matches(msg, keys.SwitchView):
			m.pendingG = false
			if err := m.switchScope(); err != nil {
				m.err = err
				return m, tea.Quit
			}
		case key.Matches(msg, keys.EditStart):
			m.pendingG = false
			m.startEditCurrent(false)
		case key.Matches(msg, keys.EditEnd):
			m.pendingG = false
			m.startEditCurrent(true)
		case key.Matches(msg, keys.OpenBelow):
			m.pendingG = false
			m.startNewItem(true)
		case key.Matches(msg, keys.OpenAbove):
			m.pendingG = false
			m.startNewItem(false)
		case key.Matches(msg, keys.Up):
			m.pendingG = false
			if m.cursor > 0 {
				m.cursor--
			} else if len(m.visibleIndexes()) > 0 {
				m.startDirectionalNewItem(false, -1)
			}
		case key.Matches(msg, keys.Down):
			m.pendingG = false
			if m.cursor < len(m.visibleIndexes())-1 {
				m.cursor++
			} else if len(m.visibleIndexes()) > 0 {
				m.startDirectionalNewItem(true, 1)
			}
		case key.Matches(msg, keys.ToggleAll):
			m.pendingG = false
			selectedIdx := -1
			visible := m.visibleIndexes()
			if m.cursor >= 0 && m.cursor < len(visible) {
				selectedIdx = visible[m.cursor]
			}
			m.showAll = !m.showAll
			m.store.HideDoneInTUI = !m.showAll
			if err := saveStore(m.location.Path, m.store); err != nil {
				m.err = err
				return m, tea.Quit
			}
			if selectedIdx >= 0 && (m.showAll || !m.store.Items[selectedIdx].Done) {
				m.cursor = m.cursorForIndex(selectedIdx)
			} else {
				m.clampCursor()
			}
		case key.Matches(msg, keys.Toggle):
			m.pendingG = false
			cmd, err := m.toggleCurrent()
			if err != nil {
				m.err = err
				return m, tea.Quit
			}
			return m, cmd
		case key.Matches(msg, keys.Delete):
			m.pendingG = false
			if err := m.deleteCurrent(); err != nil {
				m.err = err
				return m, tea.Quit
			}
		case key.Matches(msg, keys.ClearDone):
			m.pendingG = false
			if err := m.clearArchived(); err != nil {
				m.err = err
				return m, tea.Quit
			}
		case key.Matches(msg, keys.MoveUp):
			m.pendingG = false
			if err := m.moveCurrent(-1); err != nil {
				m.err = err
				return m, tea.Quit
			}
		case key.Matches(msg, keys.MoveDown):
			m.pendingG = false
			if err := m.moveCurrent(1); err != nil {
				m.err = err
				return m, tea.Quit
			}
		default:
			m.pendingG = false
		}
	}

	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Todo:"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render(m.statusLine()))
	if m.location.Notice != "" {
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render(renderLocationNotice(m.location.Notice)))
	}
	if m.confirmCreateLocal {
		b.WriteString("\n")
		b.WriteString(accentStyle.Render("Create .todos file? [y/N]"))
	}
	b.WriteString("\n\n")

	if len(m.store.Items) == 0 {
		if m.isEditing() {
			b.WriteString(renderInputRow(m.input, m.inputCursor, openBoxStyle.Background(background).Render("[ ]"), false, m.contentWidth()))
			b.WriteString("\n")
			b.WriteString(mutedStyle.Render(m.editModeHelp()))
		} else {
			b.WriteString(mutedStyle.Render("No todos yet. Press enter to add one."))
		}
		b.WriteString("\n\n")
		b.WriteString(m.help.View(keys))
		return appStyle(m.width).Render(b.String())
	}

	visible := m.visibleIndexes()
	if len(visible) == 0 {
		if m.isEditing() {
			b.WriteString(renderInputRow(m.input, m.inputCursor, openBoxStyle.Background(background).Render("[ ]"), false, m.contentWidth()))
			b.WriteString("\n")
			b.WriteString(mutedStyle.Render(m.editModeHelp()))
			b.WriteString("\n\n")
		}
		b.WriteString(mutedStyle.Render("No items in this view. Press H to toggle filters."))
		b.WriteString("\n\n")
		b.WriteString(m.help.View(keys))
		return appStyle(m.width).Render(b.String())
	}

	insertRow := -1
	if m.isEditingNewItem() {
		insertRow = m.insertRow(visible)
	}

	now := time.Now()
	contentWidth := m.contentWidth()
	for row := 0; row < len(visible); row++ {
		if row == insertRow {
			b.WriteString(renderInputRow(m.input, m.inputCursor, openBoxStyle.Background(background).Render("[ ]"), false, contentWidth))
			b.WriteString("\n")
		}

		idx := visible[row]
		if m.isEditingCurrentIndex(idx) {
			checkbox := openBoxStyle.Background(background).Render("[ ]")
			if m.store.Items[idx].Done {
				checkbox = doneBoxStyle.Background(background).Render("[✓]")
			}
			b.WriteString(renderInputRow(m.input, m.inputCursor, checkbox, false, contentWidth))
			b.WriteString("\n")
			continue
		}

		item := m.store.Items[idx]
		timestamp := taskTimestampText(item, now)
		isSelected := row == m.cursor && !m.isEditingNewItem()
		cursor := "  "
		if isSelected {
			cursor = accentStyle.Render("->")
		}

		checkbox, text := m.rowAppearance(idx, item.Description, item.Done, isSelected, contentWidth, timestamp)
		left := fmt.Sprintf("%s %s %s", cursor, checkbox, text)
		line := renderAlignedRow(left, timestamp, contentWidth, isSelected)
		if isSelected {
			b.WriteString(line)
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	if m.isEditingNewItem() && insertRow == len(visible) {
		b.WriteString(renderInputRow(m.input, m.inputCursor, openBoxStyle.Background(background).Render("[ ]"), false, contentWidth))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.isEditing() {
		b.WriteString(mutedStyle.Render(m.editModeHelp()))
		b.WriteString("\n\n")
	}
	b.WriteString(m.help.View(keys))

	return appStyle(m.width).Render(b.String())
}

func (m model) visibleIndexes() []int {
	indexes := make([]int, 0, len(m.store.Items))
	for i, item := range m.store.Items {
		if !m.showAll && item.Done && !(i == m.animatingDoneIndex && m.animatingDoneFrames > 0) {
			continue
		}
		indexes = append(indexes, i)
	}
	return indexes
}

func (m *model) clampCursor() {
	visible := m.visibleIndexes()
	if len(visible) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= len(visible) {
		m.cursor = len(visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *model) toggleCurrent() (tea.Cmd, error) {
	visible := m.visibleIndexes()
	if len(visible) == 0 {
		return nil, nil
	}

	idx := visible[m.cursor]
	wasDone := m.store.Items[idx].Done
	m.store.Items[idx].Done = !m.store.Items[idx].Done
	if m.store.Items[idx].Done {
		m.store.Items[idx].DoneAt = time.Now()
	} else {
		m.store.Items[idx].DoneAt = time.Time{}
	}

	if err := saveStore(m.location.Path, m.store); err != nil {
		return nil, err
	}

	if !wasDone && m.store.Items[idx].Done {
		m.animatingDoneIndex = idx
		m.animatingDoneFrames = 4
		m.animatingDoneCursor = m.cursor
		m.clampCursor()
		return nextCompletionFrame(), nil
	}

	m.animatingDoneIndex = -1

	m.clampCursor()
	return nil, nil
}

func (m model) animatedDoneAppearance(description string) (string, string) {
	switch m.animatingDoneFrames {
	case 4:
		return accentStyle.Render("[•]"), lipgloss.NewStyle().Foreground(foreground).Render(description)
	case 3:
		return accentStyle.Render("[✔]"), lipgloss.NewStyle().Foreground(foreground).Bold(true).Render(description)
	case 2:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("79")).Render("[✓]"), lipgloss.NewStyle().Foreground(foreground).Render(description)
	default:
		return doneBoxStyle.Render("[✓]"), doneTextStyle.Render(description)
	}
}

func (m model) rowAppearance(idx int, description string, done bool, selected bool, rowWidth int, timestamp string) (string, string) {
	if idx == m.animatingDoneIndex && m.animatingDoneFrames > 0 {
		checkbox, text := m.animatedDoneAppearance(description)
		return checkbox, truncateStyledText(text, availableDescriptionWidth(rowWidth, selected, timestamp))
	}

	textStyle := lipgloss.NewStyle()
	if selected {
		textStyle = textStyle.Background(background)
	}

	checkbox := openBoxStyle.Render("[ ]")
	text := textStyle.Foreground(foreground).Render(description)
	if done {
		boxStyle := doneBoxStyle
		if selected {
			boxStyle = boxStyle.Background(background)
		}
		checkbox = boxStyle.Render("[✓]")
		text = textStyle.Foreground(doneColor).Strikethrough(true).Render(description)
	} else if selected {
		checkbox = openBoxStyle.Background(background).Render("[ ]")
	}

	return checkbox, truncateStyledText(text, availableDescriptionWidth(rowWidth, selected, timestamp))
}

func renderAlignedRow(left, timestamp string, width int, selected bool) string {
	if timestamp == "" {
		return left
	}
	right := renderTimestampText(timestamp, selected)
	if width <= 0 {
		return left + rightAlignSpacer(left, right, 0) + right
	}
	return left + rightAlignSpacer(left, right, width) + right
}

func availableDescriptionWidth(rowWidth int, selected bool, timestamp string) int {
	if rowWidth <= 0 {
		return 0
	}
	reserved := lipgloss.Width("   [ ] ")
	if selected {
		reserved += lipgloss.Width("->")
	} else {
		reserved += lipgloss.Width("  ")
	}
	if timestamp != "" {
		reserved += lipgloss.Width(renderTimestampText(timestamp, selected)) + 2
	}
	available := rowWidth - reserved
	if available < 4 {
		return 4
	}
	return available
}

func truncateStyledText(text string, width int) string {
	if width <= 0 || lipgloss.Width(text) <= width {
		return text
	}
	if width <= 1 {
		return lipgloss.NewStyle().MaxWidth(width).Inline(true).Render("…")
	}
	visible := lipgloss.NewStyle().MaxWidth(width-1).Inline(true).Render(text)
	for lipgloss.Width(visible) > width-1 {
		visible = lipgloss.NewStyle().MaxWidth(lipgloss.Width(visible)-1).Inline(true).Render(visible)
	}
	return visible + "…"
}

func renderTimestampText(text string, selected bool) string {
	if text == "" {
		return ""
	}

	style := timestampStyle
	if selected {
		style = style.Background(background)
	}
	return style.Render(text)

}

func rightAlignSpacer(left, right string, width int) string {
	gap := 2
	if width > 0 {
		remaining := width - lipgloss.Width(left) - lipgloss.Width(right)
		if remaining > gap {
			gap = remaining
		}
	}
	if gap < 1 {
		gap = 1
	}
	return strings.Repeat(" ", gap)
}

func (m model) contentWidth() int {
	if m.width <= 0 {
		return 0
	}
	width := m.width - 4
	if width < 0 {
		return 0
}
	return width
}

func taskTimestampText(item todo, now time.Time) string {
	if item.Done {
		if item.DoneAt.IsZero() {
			return ""
		}
		return "done " + formatRelativeTaskTime(item.DoneAt, now)
	}
	if item.CreatedAt.IsZero() {
		return ""
	}
	return "created " + formatRelativeTaskTime(item.CreatedAt, now)
}

func formatRelativeTaskTime(ts, now time.Time) string {
	if ts.IsZero() {
		return ""
	}
	if now.IsZero() {
		now = time.Now()
	}

	loc := now.Location()
	ts = ts.In(loc)
	now = now.In(loc)
	if ts.After(now) {
		return ts.Format("15:04")
	}

	if now.Sub(ts) < 24*time.Hour {
		if sameDay(ts, now) {
			return ts.Format("15:04")
		}
		if dayDiff(ts, now) == 1 {
			return "yesterday " + ts.Format("15:04")
		}
	}

	days := dayDiff(ts, now)
	switch {
	case days <= 0:
		return ts.Format("15:04")
	case days == 1:
		return "yesterday"
	case days < 7:
		return "on " + ts.Format("Monday")
	case days < 14:
		return "last week"
	case days < 21:
		return "2 weeks ago"
	case days < 28:
		return "3 weeks ago"
	}

	monthDiff := (now.Year()-ts.Year())*12 + int(now.Month()-ts.Month())
	years := now.Year() - ts.Year()
	if years > 0 {
		if years == 1 {
			return "last year"
		}
		return fmt.Sprintf("%d years ago", years)
	}

	switch {
	case monthDiff <= 0:
		weeks := days / 7
		if weeks <= 1 {
			return "last week"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	case monthDiff == 1:
		return "last month"
	case monthDiff < 12:
		return "in " + ts.Format("January")
	}

	return "in " + ts.Format("January")
}

func dayDiff(ts, now time.Time) int {
	tsDay := time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, ts.Location())
	nowDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return int(nowDay.Sub(tsDay) / (24 * time.Hour))
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func (m model) isEditing() bool {
	return m.editMode != editModeNone
}

func (m model) isEditingNewItem() bool {
	return m.editMode == editModeNewAbove || m.editMode == editModeNewBelow
}

func (m model) isDirectionalNewItemEdit() bool {
	return m.isEditingNewItem() && m.directionalNewItem != 0
}

func (m model) isEditingCurrentIndex(idx int) bool {
	return m.editMode == editModeCurrent && m.editIndex == idx
}

func (m *model) cancelEdit() {
	m.editMode = editModeNone
	m.input = ""
	m.inputCursor = 0
	m.insertAt = -1
	m.editIndex = -1
	m.directionalNewItem = 0
}

func (m *model) insertInput(s string) {
	runes := []rune(m.input)
	insert := []rune(s)
	runes = append(runes[:m.inputCursor], append(insert, runes[m.inputCursor:]...)...)
	m.input = string(runes)
	m.inputCursor += len(insert)
}

func (m *model) backspaceInput() {
	if m.inputCursor == 0 {
		return
	}
	runes := []rune(m.input)
	runes = append(runes[:m.inputCursor-1], runes[m.inputCursor:]...)
	m.input = string(runes)
	m.inputCursor--
}

func nextCompletionFrame() tea.Cmd {
	return tea.Tick(70*time.Millisecond, func(time.Time) tea.Msg {
		return completionFrameMsg{}
	})
}

func (m *model) moveCurrent(delta int) error {
	visible := m.visibleIndexes()
	if len(visible) == 0 {
		return nil
	}

	from := visible[m.cursor]
	toCursor := m.cursor + delta
	if toCursor < 0 || toCursor >= len(visible) {
		return nil
	}
	to := visible[toCursor]

	m.store.Items[from], m.store.Items[to] = m.store.Items[to], m.store.Items[from]
	if err := saveStore(m.location.Path, m.store); err != nil {
		return err
	}

	m.cursor = toCursor
	return nil
}

func (m model) statusLine() string {
	open := 0
	done := 0
	for _, item := range m.store.Items {
		if item.Done {
			done++
			continue
		}
		open++
	}

	mode := "showing all"
	if !m.showAll {
		mode = "showing open"
	}

	return fmt.Sprintf("%d open, %d done • %s • %s", open, done, mode, m.location.SourceText)
}

func (m *model) switchScope() error {
	if m.location.Scope == storeScopeLocal {
		return m.switchToScope(storageGlobal)
	}

	localPath, err := localDataPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(localPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			m.confirmCreateLocal = true
			return nil
		}
		return fmt.Errorf("check local todos: %w", err)
	}

	return m.switchToScope(storageHere)
}

func (m *model) confirmLocalScopeSwitch() error {
	m.confirmCreateLocal = false
	return m.switchToScope(storageHere)
}

func (m *model) switchToScope(mode storageMode) error {
	s, location, err := loadStore(mode)
	if err != nil {
		return err
	}

	m.store = s
	m.location = location
	m.showAll = !s.HideDoneInTUI
	m.confirmCreateLocal = false
	m.cursor = 0
	m.pendingG = false
	m.animatingDoneIndex = -1
	m.animatingDoneFrames = 0
	m.animatingDoneCursor = 0
	m.cancelEdit()
	m.clampCursor()
	return nil
}

var (
	background = lipgloss.Color("236")
	foreground = lipgloss.Color("252")
	muted      = lipgloss.Color("245")
	accent     = lipgloss.Color("39")
	doneColor  = lipgloss.Color("241")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(accent).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().Foreground(muted)
	mutedStyle    = lipgloss.NewStyle().Foreground(muted)
	timestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	accentStyle   = lipgloss.NewStyle().Foreground(accent).Bold(true)
	openBoxStyle  = lipgloss.NewStyle().Foreground(accent)
	doneBoxStyle  = lipgloss.NewStyle().Foreground(doneColor)
	doneTextStyle = lipgloss.NewStyle().Foreground(doneColor).Strikethrough(true)
	selectedStyle = lipgloss.NewStyle().
			Foreground(foreground).
			Background(background)
)

func appStyle(width int) lipgloss.Style {
	style := lipgloss.NewStyle().Padding(1, 2)
	if width > 0 {
		style = style.MaxWidth(width)
	}
	return style
}

func (m *model) commitInput() error {
	description := strings.TrimSpace(m.input)
	if description == "" {
		return nil
	}
	if m.editMode == editModeCurrent && m.editIndex >= 0 && m.editIndex < len(m.store.Items) {
		m.store.Items[m.editIndex].Description = description
		if err := saveStore(m.location.Path, m.store); err != nil {
			return err
		}
		m.cancelEdit()
		m.cursor = m.cursorForIndex(m.editIndex)
		m.clampCursor()
		return nil
	}

	newTodo := todo{
		Description: description,
		CreatedAt:   time.Now(),
	}

	insertAt := m.insertAt
	if insertAt < 0 || insertAt > len(m.store.Items) {
		insertAt = len(m.store.Items)
	}

	m.store.Items = append(m.store.Items, todo{})
	copy(m.store.Items[insertAt+1:], m.store.Items[insertAt:])
	m.store.Items[insertAt] = newTodo
	if err := saveStore(m.location.Path, m.store); err != nil {
		return err
	}

	m.cancelEdit()
	m.cursor = m.cursorForIndex(insertAt)
	m.clampCursor()
	return nil
}

func (m *model) startNewItem(after bool) {
	m.editMode = editModeNewAbove
	if after {
		m.editMode = editModeNewBelow
	}
	m.directionalNewItem = 0
	m.input = ""
	m.inputCursor = 0
	m.editIndex = -1
	m.insertAt = m.currentInsertIndex(after)
}

func (m *model) startDirectionalNewItem(after bool, direction int) {
	m.startNewItem(after)
	m.directionalNewItem = direction
}

func (m *model) startEditCurrent(atEnd bool) {
	visible := m.visibleIndexes()
	if len(visible) == 0 {
		m.startNewItem(true)
		return
	}

	idx := visible[m.cursor]
	m.editMode = editModeCurrent
	m.directionalNewItem = 0
	m.editIndex = idx
	m.insertAt = -1
	m.input = m.store.Items[idx].Description
	m.inputCursor = 0
	if atEnd {
		m.inputCursor = len([]rune(m.input))
	}
}

func (m model) currentInsertIndex(after bool) int {
	visible := m.visibleIndexes()
	if len(m.store.Items) == 0 || len(visible) == 0 {
		return len(m.store.Items)
	}

	idx := visible[m.cursor]
	if after {
		return idx + 1
	}
	return idx
}

func (m model) cursorForIndex(idx int) int {
	visible := m.visibleIndexes()
	for pos, itemIdx := range visible {
		if itemIdx == idx {
			return pos
		}
	}
	if len(visible) == 0 {
		return 0
	}
	if idx >= len(m.store.Items)-1 {
		return len(visible) - 1
	}
	return m.cursor
}

func (m model) insertRow(visible []int) int {
	for row, idx := range visible {
		if m.insertAt <= idx {
			return row
		}
	}
	return len(visible)
}

func (m model) editModeHelp() string {
	switch m.editMode {
	case editModeCurrent:
		return "Editing current item. Enter saves. Esc cancels."
	case editModeNewAbove:
		return "Adding item. Enter saves. Esc cancels."
	case editModeNewBelow:
		return "Adding item. Enter saves. Esc cancels."
	}

	visible := m.visibleIndexes()
	if len(m.store.Items) == 0 || len(visible) == 0 {
		return "Adding item. Enter saves. Esc cancels."
	}
	current := visible[m.cursor]
	if m.insertAt <= current {
		return "Adding item. Enter saves. Esc cancels."
	}
	return "Adding item. Enter saves. Esc cancels."
}

func (m *model) deleteCurrent() error {
	visible := m.visibleIndexes()
	if len(visible) == 0 {
		return nil
	}

	idx := visible[m.cursor]
	m.store.Items = append(m.store.Items[:idx], m.store.Items[idx+1:]...)
	if err := saveStore(m.location.Path, m.store); err != nil {
		return err
	}

	m.clampCursor()
	return nil
}

func (m *model) clearArchived() error {
	items := m.store.Items[:0]
	for _, item := range m.store.Items {
		if item.Done {
			continue
		}
		items = append(items, item)
	}
	m.store.Items = items

	if err := saveStore(m.location.Path, m.store); err != nil {
		return err
	}

	m.clampCursor()
	return nil
}

func (m *model) handleDirectionalEditKey(direction int) (tea.Model, tea.Cmd) {
	if direction == m.directionalNewItem {
		if strings.TrimSpace(m.input) != "" {
			if err := m.commitInput(); err != nil {
				m.err = err
				return *m, tea.Quit
			}
		}
		m.startDirectionalNewItem(direction > 0, direction)
		return *m, nil
	}

	m.cancelEdit()
	return *m, nil
}

func cursorGlyph() string {
	return accentStyle.Render("█")
}

var inputStyle = lipgloss.NewStyle().
	Foreground(foreground).
	Background(lipgloss.Color("238")).
	Padding(0, 1)

var inputInlineStyle = lipgloss.NewStyle().
	Foreground(foreground)

func renderInputRow(input string, cursorPos int, checkbox string, padded bool, width int) string {
	runes := []rune(input)
	if cursorPos < 0 {
		cursorPos = 0
	}
	if cursorPos > len(runes) {
		cursorPos = len(runes)
	}
	content := string(runes[:cursorPos]) + cursorGlyph() + string(runes[cursorPos:])
	text := inputInlineStyle.Render(content)
	if padded {
		text = inputStyle.Render(content)
	}
	text = truncateStyledText(text, availableDescriptionWidth(width, true, ""))
	return renderSelectedRow(accentStyle.Render("->"), checkbox, text)
}

func renderSelectedRow(cursor, checkbox, text string) string {
	segment := lipgloss.NewStyle().Background(background)
	if checkbox == "" {
		return segment.Render(cursor) + segment.Render("  ") + segment.Render(text)
	}
	return segment.Render(cursor) + segment.Render(" ") + segment.Render(checkbox) + segment.Render(" ") + segment.Render(text)
}
