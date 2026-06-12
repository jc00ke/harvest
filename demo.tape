# Regenerates the README screenshots by driving the TUI in demo mode
# (`harvest -ui --demo`), which runs against fixture data and needs no
# credentials. Run with: mise run screenshots

Output images/demo.gif

Set Shell bash
Set FontSize 14
Set Width 810
Set Height 640
Set Padding 12

Hide
Type "./bin/harvest -ui --demo"
Enter
Sleep 3s
Show

# Time sheet view
Sleep 2s
Screenshot images/time-sheet-view.png
Sleep 1s

# Edit form for the selected entry
Type "e"
Sleep 2s
Screenshot images/add-edit-time-entry.png
Sleep 1s

# Back to the list, then the help overlay
Escape
Sleep 1s
Type "?"
Sleep 2s
Screenshot images/help-menu.png
Sleep 1s
