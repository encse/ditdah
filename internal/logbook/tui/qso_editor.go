package tui

import (
	domain "morsemanual/internal/logbook"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type qsoEditor struct {
	content components.TextView
	qso     domain.QSO
}

func newQSOEditor(controls components.Factory, qso domain.QSO) *qsoEditor {
	controls = controls.Modal()
	content := controls.TextView()
	content.SetBorder(" Edit QSO ")
	return &qsoEditor{content: content, qso: qso}
}

func (e *qsoEditor) Content() tview.Primitive {
	return e.content
}

func (e *qsoEditor) Focusables() []tview.Primitive {
	return nil
}

func (e *qsoEditor) KeyBindings() []keybinding.Binding {
	return nil
}

func (e *qsoEditor) Size() modal.Size {
	return modal.Size{Width: 84, Height: 20}
}

func (p *page) editBinding() keybinding.Binding {
	return keybinding.Binding{
		Hint: keybinding.Hint{Keys: "Enter", Description: "edit QSO"},
		Handler: func(event *tcell.EventKey) bool {
			if event.Key() != tcell.KeyEnter {
				return false
			}
			qso, ok := p.selectedQSO()
			if !ok {
				return false
			}
			p.host.OpenModal(newQSOEditor(p.host.Components(), qso))
			return true
		},
	}
}

func (p *page) selectedQSO() (domain.QSO, bool) {
	for _, qso := range p.filteredQsos {
		if qso.ID == p.selectedID {
			return qso, true
		}
	}
	return domain.QSO{}, false
}
