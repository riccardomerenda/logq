# logq demo — record with: vhs demo.tape
Output demo.gif

Set Width 1200
Set Height 600
Set FontSize 14
Set Theme "Dracula"
Set TypingSpeed 75ms
Set Padding 10

# Start logq
Type "logq testdata/sample.jsonl"
Enter
Sleep 2s

# Scroll through logs
Down Down Down Down Down
Sleep 1s

# Open filter and type a query — live filtering + match highlighting
Type "/"
Sleep 500ms
Type "error"
Sleep 2s

# Clear and show auto-complete
Backspace Backspace Backspace Backspace Backspace
Sleep 500ms

# Auto-complete field name: "lev" → Tab → "level:"
Type "lev"
Sleep 1s
Tab
Sleep 500ms

# Auto-complete field value: "err" → Tab → "error"
Type "err"
Sleep 1s
Tab
Sleep 2s

# Open detail view on selected record
Enter
Sleep 500ms
Enter
Sleep 2s

# Close detail
Escape
Sleep 1s

# Clear filter
Escape
Sleep 1s

# Quit
Type "q"
Sleep 500ms
