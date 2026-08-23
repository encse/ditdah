Require go

Set Shell "bash"
Set Width 1400
Set Height 936
Set FontSize 15
Set LetterSpacing 0
Set Padding 0
Set TypingSpeed 0
Set CursorBlink false

Type "go run -tags screenshots ./cmd/screenshot-demo decoder"
Enter
Wait+Screen /CQ CQ CQ DE HA5LA/
Sleep 500ms
Screenshot docs/screenshots/morse-decoder.png
Sleep 1s
Ctrl+C
Sleep 500ms

Type "go run -tags screenshots ./cmd/screenshot-demo logbook"
Enter
Wait+Screen /Budapest, Hungary/
Sleep 500ms
Screenshot docs/screenshots/logbook.png
Sleep 1s
Ctrl+C
Sleep 500ms

Type "go run -tags screenshots ./cmd/screenshot-demo new-qso"
Enter
Wait+Screen /Relaxed evening QSO/
Sleep 500ms
Screenshot docs/screenshots/new-qso.png
Sleep 1s
Ctrl+C
