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
```

## Plain output format

```
1. buy milk
2. urgent thing
3. [done] finished task
```

No ANSI codes, no timestamps. Numbers are stable within a session — use `-l` output to find the index before `-d N`.

## Storage

- `~/Library/Application Support/td/todos.json`

## Gotchas

- `td` with no action → blocking TUI. Always pair with `-l`, `-la`, `-d N`, or a positional arg.
- Without `-p`: ANSI color codes + timestamps in output → breaks grep/parsing.
