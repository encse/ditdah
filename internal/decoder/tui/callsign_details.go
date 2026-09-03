package tui

import (
	"strings"

	"ditdah/internal/callsign"
	"ditdah/internal/optional"
	"ditdah/internal/tui/components"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

const (
	callsignDetailLabelWidth = 8
	callsignDetailColumnGap  = 2
)

type callsignDetailField struct {
	label string
	value string
}

type callsignDetailsView struct {
	components.TextView
	fields        []callsignDetailField
	message       string
	renderedWidth int
}

func newCallsignDetailsView(controls components.Factory) *callsignDetailsView {
	text := controls.TextView()
	text.SetDynamicColors(true)
	text.SetScrollable(true)
	text.SetWrap(false)
	text.SetBorder(" QRZ.com ")
	return &callsignDetailsView{TextView: text, renderedWidth: -1}
}

func (v *callsignDetailsView) setMessage(message string) {
	v.fields = nil
	v.message = message
	v.renderedWidth = -1
	v.SetText(message)
	v.ScrollToStart()
}

func (v *callsignDetailsView) setEntry(entry callsign.Entry) {
	if entry.Status == callsign.StatusError {
		v.setMessage(strings.TrimSpace(entry.Error))
		return
	}
	record, present := entry.Record.Get()
	if !present {
		v.setMessage("No QRZ.com details available.")
		return
	}

	v.fields = []callsignDetailField{
		{label: "Callsign", value: record.Callsign},
		optionalCallsignDetail("Name", record.Name),
		optionalCallsignDetail("Nickname", record.Nickname),
		optionalCallsignDetail("QTH", record.QTH),
		optionalCallsignDetail("State", record.State),
		optionalCallsignDetail("Country", record.Country),
		optionalCallsignDetail("Grid", record.Grid),
		optionalCallsignDetail("CQ zone", record.CQZone),
		optionalCallsignDetail("ITU zone", record.ITUZone),
		optionalCallsignDetail("QRZ", record.QRZURL),
	}
	v.fields = visibleCallsignDetails(v.fields)
	v.message = ""
	v.renderedWidth = -1
	v.SetText("")
	v.ScrollToStart()
}

func (v *callsignDetailsView) Draw(screen tcell.Screen) {
	_, _, width, _ := v.InnerRect()
	if width != v.renderedWidth {
		if v.fields != nil {
			v.SetText(renderCallsignDetails(v.fields, width))
		} else {
			v.SetText(strings.Join(tview.WordWrap(v.message, width), "\n"))
		}
		v.renderedWidth = width
	}
	v.TextView.Draw(screen)
}

func optionalCallsignDetail(
	label string,
	value optional.Value[string],
) callsignDetailField {
	text, present := value.Get()
	if !present {
		return callsignDetailField{}
	}
	return callsignDetailField{label: label, value: strings.TrimSpace(text)}
}

func visibleCallsignDetails(
	fields []callsignDetailField,
) []callsignDetailField {
	visible := fields[:0]
	for _, field := range fields {
		if field.value != "" {
			visible = append(visible, field)
		}
	}
	return visible
}

func renderCallsignDetails(fields []callsignDetailField, width int) string {
	if width < 1 {
		return ""
	}
	labelWidth := min(callsignDetailLabelWidth, max(1, width-3))
	valueWidth := width - labelWidth - callsignDetailColumnGap

	var text strings.Builder
	for _, field := range fields {
		if valueWidth < 1 {
			for _, line := range tview.WordWrap(field.label+" "+field.value, width) {
				text.WriteString(tview.Escape(line))
				text.WriteByte('\n')
			}
			continue
		}

		values := tview.WordWrap(field.value, valueWidth)
		if len(values) == 0 {
			values = []string{""}
		}
		for index, value := range values {
			label := ""
			if index == 0 {
				label = runewidth.Truncate(field.label, labelWidth, "")
				label = runewidth.FillRight(label, labelWidth)
				label = "[::b]" + tview.Escape(label) + "[-:-:-]"
			} else {
				label = strings.Repeat(" ", labelWidth)
			}
			text.WriteString(label)
			text.WriteString(strings.Repeat(" ", callsignDetailColumnGap))
			text.WriteString(tview.Escape(value))
			text.WriteByte('\n')
		}
	}
	return text.String()
}
