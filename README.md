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
td "vibe features"
td -t "fix bugs"
td -H
td -H -l
td -g -l
td -l
td -la
td -d 2
td -p -l
```

- `td` opens interactive mode
- `td --help` shows help
- `td "text"` adds a new item at the bottom
- `td -t "text"` or `td --top "text"` adds at the top
- `td -b "text"` or `td --bottom "text"` adds at the bottom
- `td -l` or `td --list` lists open items
- `td -la` or `td --list-all` lists all items including done ones
- `td -d <N>` or `td --delete <N>` deletes open item #N (matches numbering from `td -l`)
- `td -p` or `td --plain` outputs a plain numbered list — no colors, no timestamps, no header; useful for piping or agent use
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
- `H`: toggle hidden vs visible done items and persist that preference
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
