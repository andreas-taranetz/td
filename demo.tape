Output usage.gif

Set FontSize 13
Set Width 680
Set Height 400
Set Theme "Dracula"
Set Shell "zsh"
Set TypingSpeed 80ms

# Setup: isolated data dir, demo binary on PATH
Hide
Type "export PATH=/tmp:$PATH && export HOME=/tmp/td-vhs-home && rm -rf $HOME && mkdir -p $HOME && clear"
Enter
Sleep 300ms
Show

# Non-interactive: add todos — no quotes needed
Type "td add vibe features"
Enter
Sleep 700ms

Type "td ship to prod"
Enter
Sleep 700ms

# Add to top with -t
Type "td -t review vibe PR"
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

# Cursor on "review vibe PR" (top). Add new item below (o)
Type "o"
Sleep 300ms
Type "fix context overflow"
Enter
Sleep 700ms

# Cursor is now on "fix context overflow". Go up one row
Type "k"
Sleep 400ms

# Cursor on "review vibe PR". Add new item above (O)
Type "O"
Sleep 300ms
Type "write AI tests"
Enter
Sleep 700ms

# Cursor on "write AI tests" (top). Navigate down two rows to "fix context overflow"
Type "j"
Sleep 350ms
Type "j"
Sleep 350ms

# Move "fix context overflow" down one position (J), then back up (K)
Type "J"
Sleep 500ms
Type "K"
Sleep 600ms

# Toggle "fix context overflow" done
Type "x"
Sleep 800ms

# Quit
Type "q"
Sleep 500ms
