package main

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/overlay"
)

const labelWidth = 24

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	theme := demoTheme()
	app := tview.NewApplication()
	content := tview.NewFlex().SetDirection(tview.FlexRow)
	overlays := overlay.New(app)
	overlays.SetContent(content)
	controls := components.New(components.Dependencies{
		Theme:    theme,
		Overlays: overlays,
	})

	fields := []components.SelectField{
		controls.SelectField(
			"Short list",
			[]string{"CW", "SSB", "AM", "FM"},
			0,
			labelWidth,
			0,
		),
		controls.SelectField(
			"Long scrollable list",
			longOptions(),
			0,
			labelWidth,
			0,
		),
		controls.SelectField(
			"Initially in the middle",
			[]string{
				"Option 01", "Option 02", "Option 03", "Option 04", "Option 05",
				"Option 06", "Option 07", "Option 08", "Option 09", "Option 10",
				"Option 11", "Option 12", "Option 13", "Option 14", "Option 15",
			},
			7,
			labelWidth,
			0,
		),
		controls.SelectField(
			"Opens upward",
			[]string{
				"Bottom 01", "Bottom 02", "Bottom 03", "Bottom 04", "Bottom 05",
				"Bottom 06", "Bottom 07", "Bottom 08", "Bottom 09", "Bottom 10",
			},
			0,
			labelWidth,
			0,
		),
	}

	for _, field := range fields {
		field.SetFormAttributes(
			labelWidth,
			theme.LabelColor,
			tcell.GetColor("#2e3440"),
			theme.FieldTextColor,
			theme.FieldBackground,
		)
	}
	wireFocus(app, fields)

	title := controls.TextView()
	title.SetTextAlign(tview.AlignCenter)
	title.SetTextColor(tcell.GetColor("#88c0d0"))
	title.SetText("SelectField playground")
	instructions := controls.TextView()
	instructions.SetTextAlign(tview.AlignCenter)
	instructions.SetTextColor(tcell.GetColor("#d8dee9"))
	instructions.SetText(
		"Enter/Space: open/select   Esc: close   Tab: next   Mouse and wheel enabled",
	)
	footer := controls.TextView()
	footer.SetTextAlign(tview.AlignCenter)
	footer.SetTextColor(tcell.GetColor("#81a1c1"))
	footer.SetText("Press q or Ctrl-C to quit")

	content.SetBackgroundColor(tcell.GetColor("#2e3440"))
	content.
		AddItem(title, 1, 0, false).
		AddItem(instructions, 2, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(fields[0], 1, 0, true).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(fields[1], 1, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(fields[2], 1, 0, false).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(fields[3], 1, 0, false).
		AddItem(footer, 2, 0, false)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if !overlays.Active() && event.Key() == tcell.KeyRune && event.Rune() == 'q' {
			app.Stop()
			return nil
		}
		return event
	})

	return app.
		SetRoot(overlays.Root(), true).
		EnableMouse(true).
		Run()
}

func wireFocus(app *tview.Application, fields []components.SelectField) {
	for index, field := range fields {
		index := index
		field.SetFinishedFunc(func(key tcell.Key) {
			switch key {
			case tcell.KeyTab:
				app.SetFocus(fields[(index+1)%len(fields)])
			case tcell.KeyBacktab:
				app.SetFocus(fields[(index+len(fields)-1)%len(fields)])
			}
		})
	}
}

func longOptions() []string {
	modes := []string{
		"CW", "SSB", "AM", "FM", "RTTY", "PSK31", "PSK63",
		"FT8", "FT4", "JS8", "WSPR", "MFSK", "Olivia", "Contestia",
		"Hellschreiber", "SSTV", "Packet", "Pactor", "FreeDV", "DV",
	}
	options := make([]string, 0, len(modes)*5)
	for repetition := 0; repetition < 5; repetition++ {
		for _, mode := range modes {
			options = append(options, fmt.Sprintf("%03d  %s", len(options)+1, mode))
		}
	}
	return options
}

func demoTheme() components.Theme {
	return components.Theme{
		Background:             tcell.GetColor("#2e3440"),
		PrimaryText:            tcell.GetColor("#eceff4"),
		SecondaryText:          tcell.GetColor("#d8dee9"),
		MutedText:              tcell.GetColor("#81a1c1"),
		Accent:                 tcell.GetColor("#88c0d0"),
		Border:                 tcell.GetColor("#4c566a"),
		LabelColor:             tcell.GetColor("#d8dee9"),
		FieldTextColor:         tcell.GetColor("#eceff4"),
		FieldBackground:        tcell.GetColor("#3b4252"),
		ActiveFieldBackground:  tcell.GetColor("#434c5e"),
		CursorColor:            tcell.GetColor("#88c0d0"),
		SelectionText:          tcell.GetColor("#2e3440"),
		SelectionBackground:    tcell.GetColor("#88c0d0"),
		PopupBorder:            tcell.GetColor("#81a1c1"),
		ButtonText:             tcell.GetColor("#eceff4"),
		ButtonBackground:       tcell.GetColor("#5e81ac"),
		ActiveButtonText:       tcell.GetColor("#2e3440"),
		ActiveButtonBackground: tcell.GetColor("#88c0d0"),
	}
}
