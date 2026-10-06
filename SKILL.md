---
name: td
description: use, add, list, delete, manage todos with td CLI — non-interactive flag usage
---

`td` is a personal todo list CLI. Agents use it **non-interactively only** via flags. Always pass `-p` for plain, parseable output. Never run `td` without an action flag or positional arg — that opens a blocking interactive TUI.

## Usage

```bash
# Add
td -p "buy milk"           # add at bottom (default)
td -p -t "urgent thing"   # add at top
td -p -b "low priority"   # add at bottom

# List
td -p -l                  # list open todos
td -p -la                 # list all (incl. done)

# Delete
td -p -d 1                # delete todo #1

# Scope
td -p -L -l               # use this folder's local list (created on first add)
td -p -g -l               # force the global list
td -p --remove-local      # delete this folder's local list (no confirmation)
```

## Plain output format

```
1. buy milk
2. urgent thing
3. [done] finished task
```

No ANSI codes, no timestamps. Numbers are stable within a session — use `-l` output to find the index before `-d N`.

## Storage

- `~/Library/Application Support/td/todos.json` holds the global list and all folder-local lists
- Without a scope flag, td uses the local list if the current folder already has one, otherwise the global list
- `-L` forces the local list of the current folder (section is created by the first add); `-g` forces global

## Gotchas

- `td` with no action → blocking TUI. Always pair with `-l`, `-la`, `-d N`, or a positional arg.
- Without `-p`: ANSI color codes + timestamps in output → breaks grep/parsing.
- Output does not say which list was used. Pass `-g` or `-L` explicitly when the scope matters.
