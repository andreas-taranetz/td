# td

Simple terminal todo manager built with Go and Bubble Tea.

Warning: 100% vibe coded.

![](usage.gif)

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
td vibe code features
td -t fix bugs
td -l
td -la
td -d 2
td -p -l
td -L fix bugs
td -g -l
```

- `td` opens interactive mode
- `td --help` shows help
- `td text` adds a new item at the bottom (multi-word, no quotes needed)
- `td -t text` or `td --top text` adds at the top
- `td -b text` or `td --bottom text` adds at the bottom
- `td -l` or `td --list` lists open items
- `td -la` or `td --list-all` lists all items including done ones
- `td -d <N>` or `td --delete <N>` deletes open item #N (matches numbering from `td -l`)
- `td -p` or `td --plain` outputs a plain numbered list — no colors, no timestamps, no header; useful for piping or agent use
- `td -L` or `td --local` uses the todo list of the current folder; the list is created by the first item you add
- `td --create-local` uses the local list of exactly this folder and creates it on first add, even if a parent folder already has a local list
- `td -g` or `td --global` uses the global list, even if the current folder has a local list
- `td --remove-local` deletes the local list of the current folder (and its todos) from the file; the global list is untouched
- `td --install-skill` installs the agent skill via `npx skills` (interactive agent selector)

After non-interactive add commands, the current open todo list is printed in a formatted, colorized view.
Open-only output omits checkboxes; `-la` includes them.

## Interactive controls

- `j` / `k` or arrow keys: move selection
- `g`: jump to top
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
- `u` / `Ctrl+R`: undo / redo the last change (up to 50 steps, also in the overview); a message shows what was undone, or that nothing is left. The history is cleared when you switch lists or enter/leave the overview
- `y`: yank (copy) the current item's text to the clipboard — the line flashes to confirm
- `p`: paste clipboard text as a new item below the cursor
- `l`: open all URLs found in the current item in the default browser
- `h`: toggle hidden vs visible done items and persist that preference
- `w`: toggle text wrapping for long items
- `Ctrl+L`: switch to the local list of the current folder (created on the first item you add)
- `Ctrl+G`: switch to the global list
- `Ctrl+A`: overview of all open todos from every list, grouped into sections (current folder, global, other folders); `x`/`Enter`/`Space` toggles, `d` deletes, `i`/`a`/`o`/`O` edit or add items in the list under the cursor, `J`/`K` move an item within its list, `D` deletes done items of the list under the cursor, `y`/`p`/`l` yank, paste and open links, `w` wraps text, `}`/`]]` and `{`/`[[` jump to the next/previous list, `Ctrl+A` or `Esc` returns; `h` toggles hidden vs visible done items there too (same persisted setting as the normal view)
- `Ctrl+X`: remove the current folder's local list after a `y` confirmation (an empty local list shows a hint for this)
- `?`: expand help
- `Esc`: cancel editing or add mode
- `q`: quit

## Agent skill

Install the td skill so AI agents (Claude Code, Copilot, etc.) can manage todos non-interactively:

```bash
td --install-skill
```

Writes the embedded `SKILL.md` to a temp dir and hands off to `npx skills` for agent selection. Requires `npx` (Node.js) on your PATH.

## Data storage

- macOS: `~/Library/Application Support/td/todos.json`
- fallback: `.td.json`

### Global and folder-local lists

Folder-local lists live in the same file as the global list, in a `folders` section keyed by absolute directory path. There are no per-project files.

- If the current folder or one of its parents already has a local list, `td` uses the nearest one automatically. The title shows the list's folder in local view, e.g. `Todo: ~/projects/td`.
- The home directory never matches for its subfolders, so a list at `~` doesn't capture everything below it.
- Otherwise `td` opens the global list and shows a hint that `Ctrl+L` creates a local list for the folder.
- `Ctrl+L` / `Ctrl+G` switch between local and global while running. A local list is only written once you add an item.
- `Ctrl+X` or `td --remove-local` deletes the local list again.
- `--create-local` is for a separate list in a subfolder: it ignores parents and uses (or creates) the exact folder's list.
- `-L` / `-g` override the automatic choice for non-interactive use. `-L` with no list in the current folder or any parent creates one for the current folder.
