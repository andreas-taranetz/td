Output usage.gif

Set FontSize 13
Set Width 680
Set Height 480
Set Theme "Dracula"
Set Shell "zsh"
Set TypingSpeed 80ms

# Setup: isolated data dir, demo binary on PATH
Hide
Type "export PATH=/tmp:$PATH && export HOME=/tmp/td-vhs-home && rm -rf $HOME && mkdir -p $HOME/projects/demo && cd $HOME/projects/demo && clear"
Enter
Sleep 300ms
Show

# Non-interactive: add todos — no quotes needed
Type "td vibe code new features"
Enter
Sleep 700ms

Type "td push straight to main"
Enter
Sleep 700ms

# Add to top with -t
Type "td -t get coffee"
Enter
Sleep 800ms

# List open todos
Type "td -l"
Enter
Sleep 1200ms

# Interactive TUI
Type "td"
Enter
Sleep 1s

# Cursor on "get coffee" (top). Add new item below (o)
Type "o"
Sleep 300ms
Type "add below with o (then i, J/K, x)"
Enter
Sleep 700ms

# Cursor is now on "add below with o (then i, J/K, x)". Go up one row
Type "k"
Sleep 400ms

# Cursor on "get coffee". Add new item above (O)
Type "O"
Sleep 300ms
Type "add above with O"
Enter
Sleep 700ms

# Cursor on "add above with O" (top). Navigate down two rows to "add below with o (then i, J/K, x)"
Type "j"
Sleep 350ms
Type "j"
Sleep 350ms

# Edit "add below with o (then i, J/K, x)" — show vim-style block cursor
Type "i"
Sleep 500ms
Right
Sleep 250ms
Right
Sleep 250ms
Right
Sleep 250ms
Right
Sleep 350ms
Left
Sleep 250ms
Left
Sleep 350ms
Escape
Sleep 600ms

# Move "add below with o (then i, J/K, x)" down one position (J), then back up (K)
Type "J"
Sleep 500ms
Type "K"
Sleep 600ms

# Toggle "add below with o (then i, J/K, x)" done
Type "x"
Sleep 800ms

# This folder has no local list yet: the status line hints at ctrl+l
Sleep 1500ms

# Switch to a local list for this folder (created on the first todo)
Ctrl+L
Sleep 1200ms
Enter
Sleep 300ms
Type "add a folder scoped todo (notice the title)"
Enter
Sleep 700ms
Type "o"
Sleep 300ms
Type "only visible in this folder"
Enter
Sleep 800ms

# Overview of all lists: local section first, then global
Ctrl+A
Sleep 1500ms
Type "}"
Sleep 800ms
Type "{"
Sleep 800ms
Type "x"
Sleep 800ms
Ctrl+A
Sleep 600ms

# Back to the global list
Ctrl+G
Sleep 1200ms

# Quit
Type "q"
Sleep 500ms

# Non-interactive: td picks the local list automatically, -g forces global
Type "td -l"
Enter
Sleep 1500ms

Type "td -g -l"
Enter
Sleep 1500ms
