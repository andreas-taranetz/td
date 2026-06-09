Output usage.gif

Set FontSize 13
Set Width 680
Set Height 360
Set Theme "Dracula"
Set Shell "zsh"
Set TypingSpeed 80ms

# Setup: isolated data dir, demo binary on PATH
Hide
Type "export PATH=/tmp:$PATH && export HOME=/tmp/td-vhs-home && rm -rf $HOME && mkdir -p $HOME && clear"
Enter
Sleep 300ms
Show

# Add some todos
Type `td "buy milk"`
Enter
Sleep 700ms

Type `td "write tests"`
Enter
Sleep 700ms

Type `td "ship the release"`
Enter
Sleep 1s

# Open interactive TUI
Type "td"
Enter
Sleep 1s

# Navigate down to second item and toggle it done
Type "j"
Sleep 400ms
Type "x"
Sleep 700ms

# Navigate to third item and toggle it done
Type "j"
Sleep 400ms
Type "x"
Sleep 700ms

# Move back up
Type "k"
Sleep 300ms
Type "k"
Sleep 500ms

# Toggle first item done too
Type "x"
Sleep 600ms

# Toggle it back open
Type "x"
Sleep 800ms

# Quit
Type "q"
Sleep 500ms
