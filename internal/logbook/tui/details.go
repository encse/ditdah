package tui

import (
	"strings"

	"ditdah/internal/tui/components"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

const detailColumnGap = 2

type detailField struct {
	label string
	value string
}

type detailRow struct {
	left  detailField
	right detailField
}

type detailsView struct {
	components.TextView
	rows          []detailRow
	renderedWidth int
}

func newDetailsView(controls components.Factory, title string) *detailsView {
	text := controls.TextView()
	text.SetDynamicColors(true)
	text.SetScrollable(true)
	text.SetWrap(false)
	text.SetBorder(title)
	return &detailsView{TextView: text, renderedWidth: -1}
}

func (v *detailsView) setRows(rows []detailRow) {
	v.rows = append(v.rows[:0], rows...)
	v.renderedWidth = -1
	v.ScrollToStart()
}

func (v *detailsView) Draw(screen tcell.Screen) {
	_, _, width, _ := v.InnerRect()
	if width != v.renderedWidth {
		v.SetText(renderDetailRows(v.rows, width))
		v.renderedWidth = width
	}
	v.TextView.Draw(screen)
}

func (p *page) renderDetails(index int) {
	if index < 0 || index >= len(p.filteredQsos) {
		p.details.setRows([]detailRow{{
			left: detailField{value: "No QSO is selected."},
		}})
		return
	}

	qso := p.filteredQsos[index]
	localTime := qso.StartedAt.Local()
	synced := "No"
	if syncedAt, present := qso.QRZSyncedAt.Get(); present {
		synced = syncedAt.Local().Format("2006-01-02 15:04")
	}

	rows := []detailRow{
		{
			left:  detailField{label: "Callsign", value: qso.Callsign},
			right: detailField{label: "Date and time", value: localTime.Format("2006-01-02 15:04:05")},
		},
		{
			left:  detailField{label: "Frequency", value: detailedFrequency(qso)},
			right: detailField{label: "Mode", value: modeName(qso)},
		},
		{
			left:  detailField{label: "RST sent", value: qso.RSTSent},
			right: detailField{label: "RST received", value: qso.RSTReceived},
		},
		{
			left:  detailField{label: "TX exchange", value: qso.ExchangeSent},
			right: detailField{label: "RX exchange", value: qso.ExchangeReceived},
		},
		{
			left:  detailField{label: "Name", value: qso.Name},
			right: detailField{label: "QTH", value: qso.QTH},
		},
		{
			left:  detailField{label: "My callsign", value: qso.StationCallsign},
			right: detailField{label: "QRZ synced", value: synced},
		},
		{
			left: detailField{label: "Notes", value: qso.Notes},
		},
	}
	visibleRows := rows[:0]
	for _, row := range rows {
		if row.left.value != "" || row.right.value != "" {
			visibleRows = append(visibleRows, row)
		}
	}
	p.details.setRows(visibleRows)
}

func renderDetailRows(rows []detailRow, width int) string {
	if width < 1 {
		return ""
	}
	columnWidth := (width - detailColumnGap) / 2
	if columnWidth < 1 {
		columnWidth = width
	}

	var text strings.Builder
	for _, row := range rows {
		left := renderDetailField(row.left, columnWidth)
		right := renderDetailField(row.right, columnWidth)
		height := max(len(left), len(right))
		for line := 0; line < height; line++ {
			text.WriteString(detailLine(left, line, columnWidth))
			if columnWidth < width {
				text.WriteString(strings.Repeat(" ", detailColumnGap))
				text.WriteString(detailLine(right, line, columnWidth))
			}
			text.WriteByte('\n')
		}
	}
	return text.String()
}

func renderDetailField(field detailField, width int) []string {
	if width < 1 {
		return nil
	}
	if field.label == "" {
		return wrapDetailValue(field.value, width)
	}

	labelWidth := min(14, max(4, width/3))
	valueWidth := width - labelWidth - 2
	if valueWidth < 1 {
		return wrapDetailValue(field.label+" "+field.value, width)
	}
	values := tview.WordWrap(field.value, valueWidth)
	if len(values) == 0 {
		values = []string{""}
	}
	lines := make([]string, len(values))
	for index, value := range values {
		label := ""
		if index == 0 {
			label = runewidth.Truncate(field.label, labelWidth, "")
			label = runewidth.FillRight(label, labelWidth)
			label = "[::b]" + tview.Escape(label) + "[-:-:-]"
		} else {
			label = strings.Repeat(" ", labelWidth)
		}
		value = runewidth.FillRight(value, valueWidth)
		lines[index] = " " + label + " " + tview.Escape(value)
	}
	return lines
}

func wrapDetailValue(value string, width int) []string {
	values := tview.WordWrap(value, max(1, width-1))
	if len(values) == 0 {
		values = []string{""}
	}
	for index, line := range values {
		line = runewidth.FillRight(line, max(1, width-1))
		values[index] = " " + tview.Escape(line)
	}
	return values
}

func detailLine(lines []string, index int, width int) string {
	if index < len(lines) {
		return lines[index]
	}
	return strings.Repeat(" ", width)
}
