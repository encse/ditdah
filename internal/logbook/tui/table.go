package tui

import (
	"fmt"
	"strings"

	domain "morsemanual/internal/logbook"
	"morsemanual/internal/tui/components"
)

type column struct {
	heading  string
	width    int
	expanded bool
}

var columns = []column{
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

func (p *page) newTable(controls components.Factory) components.Table {
	table := controls.Table(" QSOs ")
	table.SetSelectionChangedFunc(func(row, _ int) {
		p.selectionChanged(row - 1)
	})
	return table
}

func (p *page) selectionChanged(index int) {
	selectedID := ""
	if index >= 0 && index < len(p.filteredQsos) {
		selectedID = p.filteredQsos[index].ID
	}
	if selectedID == p.selectedID {
		return
	}
	p.selectedID = selectedID
	p.refreshView()
}

func (p *page) renderTable(selectedID string) int {
	p.table.Clear()
	for index, definition := range columns {
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
		p.table.SetCell(0, index, cell)
	}

	selectedRow := 1
	for index, qso := range p.filteredQsos {
		p.renderTableRow(index+1, qso)
		if qso.ID == selectedID {
			selectedRow = index + 1
		}
	}

	if len(p.filteredQsos) == 0 {
		p.table.SetCell(1, 0, components.TableCell{
			Text:     "No matching QSOs.",
			Style:    components.TableCellMuted,
			Disabled: true,
		})
		return -1
	}
	return selectedRow - 1
}

func (p *page) renderTableRow(row int, qso domain.QSO) {
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
	for index, value := range values {
		definition := columns[index]
		cell := components.TableCell{Text: value, MaxWidth: definition.width}
		if definition.expanded {
			cell.Expansion = 1
		}
		p.table.SetCell(row, index, cell)
	}
}

func formatFrequency(qso domain.QSO) string {
	frequency, present := qso.FrequencyHz.Get()
	if !present {
		return ""
	}
	formatted := fmt.Sprintf("%.6f", float64(frequency)/1_000_000)
	return strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
}

func detailedFrequency(qso domain.QSO) string {
	frequency, present := qso.FrequencyHz.Get()
	if !present {
		return ""
	}
	return fmt.Sprintf("%s MHz (%d Hz)", formatFrequency(qso), frequency)
}

func modeName(qso domain.QSO) string {
	if qso.Submode == "" {
		return qso.Mode
	}
	return qso.Mode + "/" + qso.Submode
}
