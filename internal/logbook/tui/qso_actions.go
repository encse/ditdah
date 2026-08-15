package tui

import (
	"fmt"
	"slices"
	"time"

	domain "morsemanual/internal/logbook"
	"morsemanual/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
)

func (p *page) createBinding() keybinding.Binding {
	return keybinding.OnRune('n', "new QSO", p.openCreateQSO)
}

func (p *page) editBinding() keybinding.Binding {
	return keybinding.OnKey(tcell.KeyEnter, "edit QSO", p.openSelectedQSO)
}

func (p *page) deleteBinding() keybinding.Binding {
	return keybinding.OnRune('d', "delete QSO", p.confirmDeleteQSO)
}

func (p *page) openCreateQSO() {
	qso := domain.QSO{
		StartedAt: time.Now(),
		Mode:      "CW",
	}
	if selected, ok := p.selectedQSO(); ok {
		qso.StationCallsign = selected.StationCallsign
	}
	editor := newQSOEditor(p.host.Components(), qso, p.addQSO)
	editor.setHandle(p.host.OpenModal(editor))
}

func (p *page) openSelectedQSO() {
	qso, ok := p.selectedQSO()
	if !ok {
		return
	}
	editor := newQSOEditor(p.host.Components(), qso, p.updateQSO)
	editor.setHandle(p.host.OpenModal(editor))
}

func (p *page) confirmDeleteQSO() {
	qso, ok := p.selectedQSO()
	if !ok {
		return
	}
	dialog := newConfirmDialog(
		p.host.Components(),
		" Delete QSO ",
		fmt.Sprintf("Delete QSO with %s?", qso.Callsign),
		"This action cannot be undone.",
		"Delete",
		func() error { return p.deleteQSO(qso.ID) },
	)
	dialog.setHandle(p.host.OpenModal(dialog))
}

func (p *page) addQSO(qso domain.QSO) (domain.QSO, error) {
	created, err := p.store.Add(p.ctx, qso)
	if err != nil {
		return domain.QSO{}, fmt.Errorf("add QSO: %w", err)
	}
	p.qsos = append(p.qsos, created)
	p.selectedID = created.ID
	p.applyFilter()
	return created, nil
}

func (p *page) updateQSO(qso domain.QSO) (domain.QSO, error) {
	updated, err := p.store.Update(p.ctx, qso)
	if err != nil {
		return domain.QSO{}, fmt.Errorf("update QSO: %w", err)
	}
	for index := range p.qsos {
		if p.qsos[index].ID == updated.ID {
			p.qsos[index] = updated
			break
		}
	}
	p.selectedID = updated.ID
	p.applyFilter()
	return updated, nil
}

func (p *page) deleteQSO(id string) error {
	nextSelectedID := p.selectionAfterDelete(id)
	if err := p.store.Delete(p.ctx, id); err != nil {
		return fmt.Errorf("delete QSO: %w", err)
	}
	p.qsos = slices.DeleteFunc(p.qsos, func(qso domain.QSO) bool {
		return qso.ID == id
	})
	p.selectedID = nextSelectedID
	p.applyFilter()
	return nil
}

func (p *page) selectionAfterDelete(id string) string {
	for index, qso := range p.filteredQsos {
		if qso.ID != id {
			continue
		}
		if index+1 < len(p.filteredQsos) {
			return p.filteredQsos[index+1].ID
		}
		if index > 0 {
			return p.filteredQsos[index-1].ID
		}
		return ""
	}
	return p.selectedID
}

func (p *page) selectedQSO() (domain.QSO, bool) {
	for _, qso := range p.filteredQsos {
		if qso.ID == p.selectedID {
			return qso, true
		}
	}
	return domain.QSO{}, false
}
