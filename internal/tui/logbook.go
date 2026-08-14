// Package tui implements the terminal user interface.
package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"morsemanual/internal/logbook"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/overlay"
)

const qsoPageSize = 500

type logbookColumn struct {
	heading  string
	width    int
	expanded bool
}

var logbookColumns = []logbookColumn{
	{heading: "Date", width: 11},
	{heading: "Time", width: 6},
	{heading: "Callsign", width: 13},
	{heading: "Frequency", width: 11},
	{heading: "Mode", width: 7},
	{heading: "Sent", width: 6},
	{heading: "Received", width: 9},
	{heading: "TX exch", width: 11},
	{heading: "RX exch", width: 11},
	{heading: "Name", width: 15},
	{heading: "QTH", expanded: true},
}

type logbookView struct {
	ctx   context.Context
	app   *tview.Application
	store logbook.Store
	theme colorTheme

	header   components.TextView
	search   *tview.InputField
	table    components.Table
	details  components.TextView
	footer   components.TextView
	overlays overlay.Host

	qsos      []logbook.QSO
	visible   []logbook.QSO
	searching bool
}

// Run opens the logbook screen and blocks until the user quits or ctx is
// cancelled.
func Run(ctx context.Context, store logbook.Store) error {
	tview.Styles = nordTheme.styles

	view := newLogbookView(ctx, store, nordTheme)
	if err := view.load(); err != nil {
		return err
	}

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			view.app.Stop()
		case <-done:
		}
	}()

	if err := view.app.SetRoot(view.overlays.Root(), true).EnableMouse(true).Run(); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}

	return nil
}

func newLogbookView(
	ctx context.Context,
	store logbook.Store,
	theme colorTheme,
) *logbookView {
	view := &logbookView{
		ctx:   ctx,
		app:   tview.NewApplication(),
		store: store,
		theme: theme,
	}
	layout := tview.NewFlex().SetDirection(tview.FlexRow)
	view.overlays = overlay.New(view.app, layout)
	controls := components.New(components.Dependencies{
		Theme:    theme.components(),
		Overlays: view.overlays,
	})

	view.header = controls.TextView()
	view.header.SetDynamicColors(true)
	view.header.SetTextColor(theme.accent)

	view.search = tview.NewInputField().
		SetLabel(" Search  ").
		SetPlaceholder("callsign, date, frequency, mode, name, QTH...").
		SetLabelColor(theme.accent).
		SetFieldTextColor(theme.styles.PrimaryTextColor).
		SetFieldBackgroundColor(theme.styles.ContrastBackgroundColor).
		SetPlaceholderTextColor(theme.muted)
	view.search.SetChangedFunc(func(string) {
		view.applyFilter()
	})
	view.search.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			view.leaveSearch(false)
		case tcell.KeyEscape:
			view.leaveSearch(true)
		}
	})

	view.table = controls.Table(" QSOs ")
	view.table.SetSelectionChangedFunc(func(row, _ int) {
		view.renderDetails(row - 1)
	})

	view.details = controls.TextView()
	view.details.SetDynamicColors(true)
	view.details.SetWordWrap(true)
	view.details.SetBorder(" QSO info ")

	view.footer = controls.TextView()
	view.footer.SetDynamicColors(true)
	view.footer.SetTextAlign(tview.AlignCenter)
	view.footer.SetTextColor(theme.muted)

	layout.
		AddItem(view.header, 1, 0, false).
		AddItem(view.search, 1, 0, false).
		AddItem(view.table, 0, 2, true).
		AddItem(view.details, 9, 0, false).
		AddItem(view.footer, 1, 0, false)

	view.app.SetInputCapture(view.captureKey)
	return view
}

func (v *logbookView) captureKey(event *tcell.EventKey) *tcell.EventKey {
	if v.overlays != nil && v.overlays.Active() {
		return event
	}
	if v.searching {
		return event
	}

	switch {
	case event.Key() == tcell.KeyCtrlC:
		v.app.Stop()
		return nil
	case event.Key() == tcell.KeyRune && event.Rune() == 'q':
		v.app.Stop()
		return nil
	case event.Key() == tcell.KeyRune && event.Rune() == '/':
		v.searching = true
		v.app.SetFocus(v.search)
		v.renderFooter()
		return nil
	case event.Key() == tcell.KeyRune && event.Rune() == 'j':
		return tcell.NewEventKey(tcell.KeyDown, 0, event.Modifiers())
	case event.Key() == tcell.KeyRune && event.Rune() == 'k':
		return tcell.NewEventKey(tcell.KeyUp, 0, event.Modifiers())
	}

	return event
}

func (v *logbookView) leaveSearch(clear bool) {
	v.searching = false
	if clear {
		v.search.SetText("")
	}
	v.app.SetFocus(v.table)
	v.renderFooter()
}

func (v *logbookView) load() error {
	var qsos []logbook.QSO
	for offset := 0; ; offset += qsoPageSize {
		page, err := v.store.List(v.ctx, logbook.Filter{
			Limit:  qsoPageSize,
			Offset: offset,
		})
		if err != nil {
			return fmt.Errorf("load logbook: %w", err)
		}
		qsos = append(qsos, page...)
		if len(page) < qsoPageSize {
			break
		}
	}

	v.qsos = qsos
	v.applyFilter()
	return nil
}

func (v *logbookView) applyFilter() {
	selectedID := v.selectedID()
	query := strings.ToLower(strings.TrimSpace(v.search.GetText()))

	v.visible = v.visible[:0]
	for _, qso := range v.qsos {
		if query == "" || strings.Contains(searchableText(qso), query) {
			v.visible = append(v.visible, qso)
		}
	}

	v.renderTable(selectedID)
	v.renderHeader()
	v.renderFooter()
}

func (v *logbookView) renderTable(selectedID string) {
	v.table.Clear()
	for column, definition := range logbookColumns {
		heading := definition.heading
		if definition.width > 0 {
			heading = fmt.Sprintf("%-*s", definition.width, heading)
		}
		cell := components.TableCell{
			Text:     heading,
			Style:    components.TableCellHeader,
			Disabled: true,
			MaxWidth: definition.width,
		}
		if definition.expanded {
			cell.Expansion = 1
		}
		v.table.SetCell(0, column, cell)
	}

	selectedRow := 1
	for index, qso := range v.visible {
		localTime := qso.StartedAt.Local()
		values := []string{
			localTime.Format("2006-01-02"),
			localTime.Format("15:04"),
			qso.Callsign,
			formatFrequency(qso),
			modeName(qso),
			qso.RSTSent,
			qso.RSTReceived,
			qso.ExchangeSent,
			qso.ExchangeReceived,
			qso.Name,
			qso.QTH,
		}
		for column, value := range values {
			definition := logbookColumns[column]
			cell := components.TableCell{
				Text:     value,
				MaxWidth: definition.width,
			}
			if definition.expanded {
				cell.Expansion = 1
			}
			v.table.SetCell(index+1, column, cell)
		}
		if qso.ID == selectedID {
			selectedRow = index + 1
		}
	}

	if len(v.visible) == 0 {
		v.table.SetCell(1, 0, components.TableCell{
			Text:     "No matching QSOs.",
			Style:    components.TableCellMuted,
			Disabled: true,
		})
		v.renderDetails(-1)
		return
	}

	v.table.Select(selectedRow, 0)
	v.renderDetails(selectedRow - 1)
}

func (v *logbookView) renderHeader() {
	text := fmt.Sprintf(
		"[::b] Logbook[-:-:-]  %s(%d/%d)[-]",
		colorTag(v.theme.muted),
		len(v.visible),
		len(v.qsos),
	)
	v.header.SetText(text)
}

func (v *logbookView) renderFooter() {
	if v.searching {
		v.footer.SetText("[::b]Enter[-:-:-] apply   [::b]Esc[-:-:-] clear search")
		return
	}
	v.footer.SetText(
		"[::b]↑/k ↓/j ←/→[-:-:-] move   [::b]PgUp/PgDn[-:-:-] page   " +
			"[::b]/[-:-:-] search   [::b]q[-:-:-] quit",
	)
}

func (v *logbookView) renderDetails(index int) {
	v.details.Clear()
	if index < 0 || index >= len(v.visible) {
		fmt.Fprint(v.details, colorTag(v.theme.muted)+"No QSO is selected.[-]")
		return
	}

	qso := v.visible[index]
	localTime := qso.StartedAt.Local()
	synced := "No"
	if syncedAt, present := qso.QRZSyncedAt.Get(); present {
		synced = syncedAt.Local().Format("2006-01-02 15:04")
	}

	rows := [][4]string{
		{"Callsign", qso.Callsign, "Date and time", localTime.Format("2006-01-02 15:04:05")},
		{"Frequency", detailedFrequency(qso), "Mode", modeName(qso)},
		{"RST sent", qso.RSTSent, "RST received", qso.RSTReceived},
		{"TX exchange", qso.ExchangeSent, "RX exchange", qso.ExchangeReceived},
		{"Name", qso.Name, "QTH", qso.QTH},
		{"My callsign", qso.StationCallsign, "QRZ synced", synced},
		{"Notes", qso.Notes, "", ""},
	}
	for _, row := range rows {
		if row[1] == "" && row[3] == "" {
			continue
		}
		fmt.Fprintln(v.details, detailPair(row[0], row[1], row[2], row[3]))
	}
}

func (v *logbookView) selectedID() string {
	row, _ := v.table.Selection()
	index := row - 1
	if index < 0 || index >= len(v.visible) {
		return ""
	}
	return v.visible[index].ID
}

func searchableText(qso logbook.QSO) string {
	values := []string{
		qso.Callsign,
		qso.StationCallsign,
		qso.StartedAt.Local().Format("2006-01-02 15:04"),
		formatFrequency(qso),
		qso.Mode,
		qso.Submode,
		qso.RSTSent,
		qso.RSTReceived,
		qso.ExchangeSent,
		qso.ExchangeReceived,
		qso.Name,
		qso.QTH,
		qso.Notes,
	}
	return strings.ToLower(strings.Join(values, " "))
}

func formatFrequency(qso logbook.QSO) string {
	frequency, present := qso.FrequencyHz.Get()
	if !present {
		return ""
	}
	formatted := fmt.Sprintf("%.6f", float64(frequency)/1_000_000)
	return strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
}

func detailedFrequency(qso logbook.QSO) string {
	frequency, present := qso.FrequencyHz.Get()
	if !present {
		return ""
	}
	return fmt.Sprintf("%s MHz (%d Hz)", formatFrequency(qso), frequency)
}

func modeName(qso logbook.QSO) string {
	if qso.Submode == "" {
		return qso.Mode
	}
	return qso.Mode + "/" + qso.Submode
}

func detailPair(leftLabel, leftValue, rightLabel, rightValue string) string {
	left := fmt.Sprintf(
		"[::b]%-14s[-:-:-] %-24s",
		tview.Escape(leftLabel),
		tview.Escape(leftValue),
	)
	if rightLabel == "" {
		return left
	}
	return left + fmt.Sprintf(
		"  [::b]%-14s[-:-:-] %s",
		tview.Escape(rightLabel),
		tview.Escape(rightValue),
	)
}
