# td

Simple terminal todo manager built with Go and Bubble Tea.

Warning: 100% vibe coded.

## Install

Install directly from GitHub:

```bash
go install github.com/andreas-taranetz/td@latest
```

Or install from a local checkout after cloning the repository:

```bash
git clone https://github.com/andreas-taranetz/td.git
cd td
go install .
```

By default this installs the binary into `$(go env GOPATH)/bin` unless `GOBIN` is set.

For most setups that means:

```bash
~/go/bin
```

If that directory is not already on your `PATH`, add it in your shell config file such as `.zshrc` or `.bashrc`:

```bash
export PATH="$HOME/go/bin:$PATH"
```

Then reload your shell.

## Usage

```bash
td --help
td
td "vibe features"
td -t "fix bugs"
td -H
td -H -l
td -g -l
td -l
td -la
```

- `td` opens interactive mode
- `td --help` shows help
- `td "text"` adds a new item at the bottom
- `td -t "text"` or `td --top "text"` adds at the top
- `td -b "text"` or `td --bottom "text"` adds at the bottom
- `td -H` or `td --here` creates `.todos` when needed and then uses it
- `td -g` or `td --global` forces the global store even when `./.todos` exists
- `td -l` or `td --list` lists open items
- `td -la` or `td --list-all` lists all items including done ones

After non-interactive add commands, the current open todo list is printed in a formatted, colorized view.
Open-only output omits checkboxes; `-la` includes them.
Interactive mode and list output show whether td is using the local or global store.
When `td -H` creates a new local `.todos` inside a git checkout, td also shows what to add to `.gitignore` so the file stays local.

## Interactive controls

- `j` / `k` or arrow keys: move selection
- `g`: jump to top
- `G`: jump to bottom
- `Tab`: switch between local and global scope
- `i`: edit the current item from the beginning
- `a`: edit the current item from the end
- `o`: create a new item below the current item
- `O`: create a new item above the current item
- `x`, `Enter`, or `Space`: toggle done/undone
- `J` or `Shift+Down`: move selected item down
- `K` or `Shift+Up`: move selected item up
- `d`: delete the selected item
- `D`: delete all done items
- `H`: toggle hidden vs visible done items and persist that preference
- `?`: expand help
- `Esc`: cancel editing or add mode
- `q`: quit

## Data storage

td uses project-local storage first when `./.todos` exists in the current directory.
If no local file exists, it falls back to the global store.

- local project file: `.todos`
- macOS global path: `~/Library/Application Support/td/todos.json`
- fallback global path: `.td.json`

Use `td -H` to create `.todos` and continue with the requested action in the current directory.
Use `td -g` to access the global todos even inside a project folder with a local `.todos` file.
