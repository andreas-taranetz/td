# td

Simple terminal todo manager built with Go and Bubble Tea.

Warning: 100% vibe coded.

## Install

Install with Go:

```bash
go install .
```

If you publish this repository, users can install it with:

```bash
go install github.com/yourname/td@latest
```

By default this installs the binary into `$(go env GOPATH)/bin` unless `GOBIN` is set.

For most setups that means:

```bash
~/go/bin
```

If that directory is not already on your `PATH`, add:

```bash
export PATH="$HOME/go/bin:$PATH"
```

Then reload your shell.

## Usage

```bash
td --help
td
td "buy milk"
td -t "pay rent"
td -l
td -la
```

- `td` opens interactive mode
- `td --help` shows help
- `td "text"` adds a new item at the bottom
- `td -t "text"` or `td --top "text"` adds at the top
- `td -b "text"` or `td --bottom "text"` adds at the bottom
- `td -l` or `td --list` lists open items
- `td -la` or `td --list-all` lists all items including done ones

After non-interactive add commands, the current open todo list is printed in a formatted, colorized view.
Open-only output omits checkboxes; `-la` includes them.

## Interactive controls

- `j` / `k` or arrow keys: move selection
- `gg`: jump to top
- `G`: jump to bottom
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
- `Esc`: cancel add mode
- `q`: quit

## Data storage

Todos are stored in a local JSON file.

- macOS path: `~/Library/Application Support/td/todos.json`
- fallback path: `.td.json`
