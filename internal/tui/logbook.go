// Package tui implements the terminal user interface.
package tui

import (
	"context"
	"fmt"
	"strings"

	"morsemanual/internal/logbook"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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

type logbookPage struct {
	ctx   context.Context
	host  PageHost
	store logbook.Store
	theme colorTheme

	content tview.Primitive
	search  components.InputField
	table   components.Table
	details components.TextView

	qsos         []logbook.QSO
	filteredQsos []logbook.QSO
	selectedID   string
}

var _ Page = (*logbookPage)(nil)

// Run opens the logbook screen and blocks until the user quits or ctx is
// cancelled.
func Run(ctx context.Context, store logbook.Store) error {
	app := NewApplication(nordTheme)
	page := newLogbookPage(ctx, app, store)
	if err := page.load(); err != nil {
		return err
	}

	if err := app.Register(page); err != nil {
		return err
	}
	if err := app.Show(page.ID()); err != nil {
		return err
	}
	return app.Run(ctx)
}

func newLogbookPage(
	ctx context.Context,
	host PageHost,
	store logbook.Store,
) *logbookPage {
	controls := host.Components()
	page := &logbookPage{
		ctx:   ctx,
		host:  host,
		store: store,
		theme: host.Theme(),
	}

	page.search = controls.InputField(" Search  ", "")
	page.search.SetPlaceholder("callsign, date, frequency, mode, name, QTH...")
	page.search.SetChangedFunc(func(string) {
		page.applyFilter()
	})
	page.search.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			page.leaveSearch(false)
		case tcell.KeyEscape:
			page.leaveSearch(true)
		}
	})

	page.table = controls.Table(" QSOs ")
	page.table.SetSelectionChangedFunc(func(row, _ int) {
		page.selectionChanged(row - 1)
	})

	page.details = controls.TextView()
	page.details.SetDynamicColors(true)
	page.details.SetWordWrap(true)
	page.details.SetBorder(" QSO info ")

	content := tview.NewFlex().SetDirection(tview.FlexRow)
	content.
		AddItem(page.search, 1, 0, false).
		AddItem(page.table, 0, 2, true).
		AddItem(page.details, 9, 0, false)
	page.content = content
	return page
}

func (v *logbookPage) ID() string {
	return "logbook"
}

func (v *logbookPage) Title() string {
	return "Logbook"
}

func (v *logbookPage) Content() tview.Primitive {
	return v.content
}

func (v *logbookPage) KeyBindings() []keybinding.Binding {
	return []keybinding.Binding{
		{
			Hint: keybinding.Hint{Keys: "/", Description: "search"},
			Handler: func(event *tcell.EventKey) bool {
				if event.Key() != tcell.KeyRune || event.Rune() != '/' {
					return false
				}
				v.host.SetFocus(v.search)
				return true
			},
		},
	}
}

func (v *logbookPage) Status() string {
	return fmt.Sprintf("(%d/%d)", len(v.filteredQsos), len(v.qsos))
}

func (v *logbookPage) leaveSearch(clear bool) {
	if clear {
		v.search.SetValue("")
	}
	v.host.SetFocus(v.table)
}

func (v *logbookPage) load() error {
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

func (v *logbookPage) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(v.search.Value()))

	v.filteredQsos = v.filteredQsos[:0]
	for _, qso := range v.qsos {
		if query == "" || strings.Contains(searchableText(qso), query) {
			v.filteredQsos = append(v.filteredQsos, qso)
		}
	}

	v.refreshView()
}

func (v *logbookPage) refreshView() {
	selectedIndex := v.renderTable(v.selectedID)
	if selectedIndex >= 0 {
		v.selectedID = v.filteredQsos[selectedIndex].ID
		v.table.Select(selectedIndex+1, 0)
	} else {
		v.selectedID = ""
	}
	v.renderDetails(selectedIndex)
	v.host.Refresh()
}

func (v *logbookPage) selectionChanged(index int) {
	selectedID := ""
	if index >= 0 && index < len(v.filteredQsos) {
		selectedID = v.filteredQsos[index].ID
	}
	if selectedID == v.selectedID {
		return
	}
	v.selectedID = selectedID
	v.refreshView()
}

func (v *logbookPage) renderTable(selectedID string) int {
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
	for index, qso := range v.filteredQsos {
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

	if len(v.filteredQsos) == 0 {
		v.table.SetCell(1, 0, components.TableCell{
			Text:     "No matching QSOs.",
			Style:    components.TableCellMuted,
			Disabled: true,
		})
		return -1
	}

	return selectedRow - 1
}

func (v *logbookPage) renderDetails(index int) {
	v.details.Clear()
	if index < 0 || index >= len(v.filteredQsos) {
		fmt.Fprint(v.details, colorTag(v.theme.muted)+"No QSO is selected.[-]")
		return
	}

	qso := v.filteredQsos[index]
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
