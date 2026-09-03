package tui

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"ditdah/internal/tui/components"
	"ditdah/internal/tui/modal"
)

func (p *page) newCallsignList(controls components.Factory) components.Table {
	table := controls.Table(" Callsigns ")
	table.SetFixedRows(1)
	table.SetSelectionChangedFunc(func(row, _ int) {
		p.selectCallsign(row - 1)
	})
	return table
}

func (p *page) renderCallsigns() {
	p.callsignList.Clear()
	p.callsignList.SetCell(0, 0, components.TableCell{
		Text:     "Callsign",
		Style:    components.TableCellHeader,
		Disabled: true,
	})
	p.callsignList.SetCell(0, 1, components.TableCell{
		Text:      "Logbook",
		Style:     components.TableCellHeader,
		Disabled:  true,
		Expansion: 1,
	})
	selectedRow := -1
	for index, value := range p.callsigns {
		row := index + 1
		p.callsignList.SetCell(row, 0, components.TableCell{
			Text: value,
		})
		marker := ""
		if _, logged := p.loggedCallsigns[value]; logged {
			marker = "✓"
		}
		p.callsignList.SetCell(row, 1, components.TableCell{
			Text:      marker,
			Expansion: 1,
		})
		if value == p.selectedCallsign {
			selectedRow = row
		}
	}
	if len(p.callsigns) == 0 {
		p.callsignList.SetCell(1, 0, components.TableCell{
			Text:      "No callsigns.",
			Style:     components.TableCellMuted,
			Disabled:  true,
			Expansion: 1,
		})
		p.selectedCallsign = ""
		p.callsignList.Select(1, 0)
		return
	}
	if selectedRow < 0 {
		selectedRow = 1
		p.selectedCallsign = p.callsigns[0]
	}
	p.callsignList.Select(selectedRow, 0)
}

func (p *page) selectCallsign(index int) {
	value := ""
	if index >= 0 && index < len(p.callsigns) {
		value = p.callsigns[index]
	}
	if value == p.selectedCallsign {
		return
	}
	p.selectedCallsign = value
	p.requestSelectedCallsign()
}

func (p *page) requestSelectedCallsign() {
	p.renderDecodedText()
	p.lookupGeneration++
	request := lookupRequest{
		generation: p.lookupGeneration,
		callsign:   p.selectedCallsign,
	}
	if request.callsign == "" {
		p.details.SetStyle(components.TextViewPrimary)
		p.details.setMessage("")
	} else {
		p.details.SetStyle(components.TextViewMuted)
		p.details.setMessage("Loading " + request.callsign + "...")
	}
	p.lookups.Send(request)
	p.host.Refresh()
}

func (p *page) openAddCallsign() {
	dialog := newCallsignDialog(
		p.host.Components(),
		"Add callsign",
		"Add",
		"",
		p.addCallsign,
	)
	dialog.setHandle(p.host.OpenModal(p, dialog))
}

func (p *page) openEditCallsign() {
	value := p.selectedCallsign
	if !slices.Contains(p.callsigns, value) {
		return
	}
	dialog := newCallsignDialog(
		p.host.Components(),
		"Edit callsign",
		"Save",
		value,
		func(updated string) error { return p.updateCallsign(value, updated) },
	)
	dialog.setHandle(p.host.OpenModal(p, dialog))
}

func (p *page) addCallsign(value string) error {
	value, err := normalizeListCallsign(value)
	if err != nil {
		return err
	}
	if slices.Contains(p.callsigns, value) {
		return fmt.Errorf("callsign %s is already in the list", value)
	}
	p.callsigns = append(p.callsigns, value)
	p.selectedCallsign = value
	p.renderCallsigns()
	p.requestSelectedCallsign()
	return nil
}

func (p *page) updateCallsign(previous, value string) error {
	value, err := normalizeListCallsign(value)
	if err != nil {
		return err
	}
	index := slices.Index(p.callsigns, previous)
	if index < 0 {
		return errors.New("selected callsign is no longer in the list")
	}
	if value != previous && slices.Contains(p.callsigns, value) {
		return fmt.Errorf("callsign %s is already in the list", value)
	}
	p.callsigns[index] = value
	p.selectedCallsign = value
	p.renderCallsigns()
	p.requestSelectedCallsign()
	return nil
}

func normalizeListCallsign(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "", errors.New("callsign is required")
	}
	return value, nil
}

func (p *page) confirmDeleteSelectedCallsign() {
	value := p.selectedCallsign
	if !slices.Contains(p.callsigns, value) {
		return
	}
	modal.OpenDangerConfirm(
		p.host,
		p,
		" Delete callsign ",
		fmt.Sprintf("Delete %s from the callsign list?", value),
		"",
		"Delete",
		func() { p.deleteCallsign(value) },
	)
}

func (p *page) deleteSelectedCallsign() {
	p.deleteCallsign(p.selectedCallsign)
}

func (p *page) deleteCallsign(value string) {
	index := slices.Index(p.callsigns, value)
	if index < 0 {
		return
	}
	p.callsigns = slices.Delete(p.callsigns, index, index+1)
	if len(p.callsigns) == 0 {
		p.selectedCallsign = ""
	} else if index < len(p.callsigns) {
		p.selectedCallsign = p.callsigns[index]
	} else {
		p.selectedCallsign = p.callsigns[len(p.callsigns)-1]
	}
	p.renderCallsigns()
	p.requestSelectedCallsign()
}
