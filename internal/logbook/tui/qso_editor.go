package tui

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	domain "morsemanual/internal/logbook"
	"morsemanual/internal/optional"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

	"github.com/rivo/tview"
)

const qsoEditorLabelWidth = 16

const qsoEditorTimeLayout = "2006-01-02 15:04"

type saveQSOFunc func(domain.QSO) (domain.QSO, error)

var qsoEditorModes = []string{
	"CW", "SSB", "AM", "FM", "RTTY", "FT8", "FT4", "PSK", "MFSK", "DATA",
}

type qsoEditor struct {
	content tview.Primitive
	qso     domain.QSO
	save    saveQSOFunc

	stationCallsign  components.InputField
	callsign         components.InputField
	startedAt        components.InputField
	frequency        components.InputField
	mode             components.SelectField
	rstSent          components.InputField
	rstReceived      components.InputField
	exchangeSent     components.InputField
	exchangeReceived components.InputField
	name             components.InputField
	qth              components.InputField
	notes            components.TextArea
	message          components.TextView
	ok               components.Button
	cancel           components.Button
	focusables       []tview.Primitive
	handle           modal.Handle
}

func newQSOEditor(
	controls components.Factory,
	qso domain.QSO,
	save saveQSOFunc,
) *qsoEditor {
	controls = controls.Modal()
	editor := &qsoEditor{qso: qso, save: save}
	editor.stationCallsign = editor.input(
		controls,
		"My callsign",
		qso.StationCallsign,
	)
	editor.callsign = editor.input(controls, "Callsign", qso.Callsign)
	editor.startedAt = editor.input(
		controls,
		"Date and time",
		qso.StartedAt.Local().Format(qsoEditorTimeLayout),
	)
	editor.frequency = editor.input(
		controls,
		"Frequency (MHz)",
		formatFrequency(qso),
	)
	modes, selectedMode := editorModes(qso.Mode)
	editor.mode = controls.SelectField(
		"Mode",
		modes,
		selectedMode,
		qsoEditorLabelWidth,
		0,
	)
	editor.rstSent = editor.input(controls, "RST sent", qso.RSTSent)
	editor.rstReceived = editor.input(controls, "RST received", qso.RSTReceived)
	editor.exchangeSent = editor.input(controls, "TX exchange", qso.ExchangeSent)
	editor.exchangeReceived = editor.input(
		controls,
		"RX exchange",
		qso.ExchangeReceived,
	)
	editor.name = editor.input(controls, "Name", qso.Name)
	editor.qth = editor.input(controls, "QTH", qso.QTH)
	editor.notes = controls.TextArea("", qso.Notes)
	editor.message = controls.TextView()
	editor.ok = controls.Button("OK")
	editor.cancel = controls.Button("Cancel")
	editor.ok.SetSelectedFunc(editor.submit)
	editor.cancel.SetSelectedFunc(editor.close)

	editor.focusables = []tview.Primitive{
		editor.stationCallsign,
		editor.callsign,
		editor.startedAt,
		editor.frequency,
		editor.mode,
		editor.rstSent,
		editor.rstReceived,
		editor.exchangeSent,
		editor.exchangeReceived,
		editor.name,
		editor.qth,
		editor.notes,
		editor.ok,
		editor.cancel,
	}
	editor.content = editor.layout(controls)
	return editor
}

func (e *qsoEditor) Content() tview.Primitive {
	return e.content
}

func (e *qsoEditor) Focusables() []tview.Primitive {
	return e.focusables
}

func (e *qsoEditor) KeyBindings() []keybinding.Binding {
	return nil
}

func (e *qsoEditor) Size() modal.Size {
	return modal.Size{Width: 84, Height: 22}
}

func (e *qsoEditor) setHandle(handle modal.Handle) {
	e.handle = handle
}

func (e *qsoEditor) close() {
	if e.handle != nil {
		e.handle.Close()
	}
}

func (e *qsoEditor) submit() {
	qso, err := e.value()
	if err != nil {
		e.showError(err)
		return
	}
	if e.save != nil {
		qso, err = e.save(qso)
		if err != nil {
			e.showError(err)
			return
		}
	}
	e.qso = qso
	e.close()
}

func (e *qsoEditor) value() (domain.QSO, error) {
	qso := e.qso
	qso.StationCallsign = e.stationCallsign.Value()
	qso.Callsign = e.callsign.Value()
	qso.RSTSent = e.rstSent.Value()
	qso.RSTReceived = e.rstReceived.Value()
	qso.ExchangeSent = e.exchangeSent.Value()
	qso.ExchangeReceived = e.exchangeReceived.Value()
	qso.Name = e.name.Value()
	qso.QTH = e.qth.Value()
	qso.Notes = e.notes.Value()
	_, qso.Mode = e.mode.CurrentOption()

	startedAt, err := e.parseStartedAt()
	if err != nil {
		return domain.QSO{}, err
	}
	qso.StartedAt = startedAt

	frequency, err := parseFrequency(e.frequency.Value())
	if err != nil {
		return domain.QSO{}, err
	}
	qso.FrequencyHz = frequency
	return qso, nil
}

func (e *qsoEditor) parseStartedAt() (time.Time, error) {
	value := strings.TrimSpace(e.startedAt.Value())
	if value == e.qso.StartedAt.Local().Format(qsoEditorTimeLayout) {
		return e.qso.StartedAt, nil
	}
	startedAt, err := time.ParseInLocation(qsoEditorTimeLayout, value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"date and time must use YYYY-MM-DD HH:MM",
		)
	}
	return startedAt, nil
}

func parseFrequency(value string) (optional.Value[int64], error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return optional.None[int64](), nil
	}
	megahertz, err := strconv.ParseFloat(value, 64)
	hertz := megahertz * 1_000_000
	if err != nil || math.IsNaN(hertz) || math.IsInf(hertz, 0) ||
		hertz <= 0 || hertz > float64(math.MaxInt64) {
		return optional.None[int64](), fmt.Errorf(
			"frequency must be a positive number in MHz",
		)
	}
	return optional.Some(int64(math.Round(hertz))), nil
}

func (e *qsoEditor) showError(err error) {
	e.message.SetText("Error: " + err.Error())
}

func (e *qsoEditor) input(
	controls components.Factory,
	label string,
	value string,
) components.InputField {
	input := controls.InputField(label, value)
	input.SetLabelWidth(qsoEditorLabelWidth)
	return input
}

func (e *qsoEditor) layout(controls components.Factory) tview.Primitive {
	fields := tview.NewGrid().
		SetColumns(0, 2, 0).
		SetRows(1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1)
	pairs := [][2]tview.Primitive{
		{e.stationCallsign, e.callsign},
		{e.startedAt, e.frequency},
		{e.rstSent, e.rstReceived},
		{e.exchangeSent, e.exchangeReceived},
		{e.name, e.qth},
	}
	for index, pair := range pairs {
		row := index * 2
		if index >= 2 {
			row += 2
		}
		fields.AddItem(pair[0], row, 0, 1, 1, 0, 0, false)
		fields.AddItem(pair[1], row, 2, 1, 1, 0, 0, false)
	}
	fields.AddItem(e.mode, 4, 0, 1, 1, 0, 0, false)

	notesLabel := controls.TextView()
	notesLabel.SetText("Notes")
	buttons := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(e.ok, 12, 0, false).
		AddItem(nil, 2, 0, false).
		AddItem(e.cancel, 12, 0, false).
		AddItem(nil, 0, 1, false)
	body := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(fields, 11, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(notesLabel, 1, 0, false).
		AddItem(e.notes, 0, 1, false).
		AddItem(e.message, 1, 0, false).
		AddItem(buttons, 1, 0, false)
	padded := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 2, 0, false).
		AddItem(
			tview.NewFlex().
				AddItem(nil, 2, 0, false).
				AddItem(body, 0, 1, false).
				AddItem(nil, 2, 0, false),
			0,
			1,
			false,
		).
		AddItem(nil, 2, 0, false)
	surface := controls.TextView()
	surface.SetBorder(e.title())
	return tview.NewPages().
		AddPage("surface", surface, true, true).
		AddPage("content", padded, true, true)
}

func (e *qsoEditor) title() string {
	if e.qso.ID == "" {
		return " New QSO "
	}
	return " Edit QSO "
}

func editorModes(current string) ([]string, int) {
	modes := slices.Clone(qsoEditorModes)
	selected := slices.Index(modes, current)
	if current != "" && selected < 0 {
		modes = append([]string{current}, modes...)
		selected = 0
	}
	if selected < 0 {
		selected = 0
	}
	return modes, selected
}
