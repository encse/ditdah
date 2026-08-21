package tui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	domain "morsemanual/internal/logbook"
	"morsemanual/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
)

type qrzSynchronizer interface {
	Sync(ctx context.Context) (int, error)
}

func (p *page) createBinding() keybinding.Binding {
	return keybinding.OnRune('n', "new QSO", p.openCreateQSO)
}

func (p *page) editBinding() keybinding.Binding {
	return keybinding.OnKey(tcell.KeyEnter, "edit QSO", p.openSelectedQSO)
}

func (p *page) deleteBinding() keybinding.Binding {
	return keybinding.OnRune('d', "delete QSO", p.confirmDeleteQSO)
}

func (p *page) syncBinding() keybinding.Binding {
	return keybinding.OnRune('u', "sync QRZ", p.confirmQRZSync)
}

func (p *page) confirmQRZSync() {
	pending := p.pendingQRZCount()
	dialog := newActionDialog(
		p.host.Components(),
		" Sync QRZ.com ",
		fmt.Sprintf("Upload %d pending QSO(s) to QRZ.com?", pending),
		"Replacement may reset the QRZ confirmation.",
		"Sync",
		p.syncQRZ,
		false,
	)
	dialog.setHandle(p.host.OpenModal(p.Content(), dialog))
}

func (p *page) syncQRZ() error {
	if p.syncer == nil {
		return fmt.Errorf("QRZ.com synchronization is unavailable")
	}
	_, syncErr := p.syncer.Sync(p.ctx)
	refreshErr := p.load()
	if refreshErr != nil {
		refreshErr = fmt.Errorf("refresh logbook after QRZ.com sync: %w", refreshErr)
	}
	return errors.Join(syncErr, refreshErr)
}

func (p *page) pendingQRZCount() int {
	pending := 0
	for _, qso := range p.qsos {
		if !qso.QRZSyncedAt.IsSome() {
			pending++
		}
	}
	return pending
}

func (p *page) openCreateQSO() {
	p.OpenCreateQSO("")
}

// OpenCreateQSO opens the editor for a new QSO with the contacted station.
// Creation defaults and available station data are resolved here so callers do
// not need to construct a partially populated domain model.
func (p *page) OpenCreateQSO(callsign string) {
	qso, err := p.newQSODraft(callsign)
	editor := newQSOEditor(p.host, qso, p.addQSO, p.lookup)
	if err != nil {
		editor.showError(err)
	}
	editor.setHandle(p.host.OpenModal(p.Content(), editor))
}

func (p *page) newQSODraft(contactedCallsign string) (domain.QSO, error) {
	qso := domain.QSO{
		Callsign:  strings.ToUpper(strings.TrimSpace(contactedCallsign)),
		StartedAt: time.Now(),
		Mode:      "CW",
	}
	if selected, ok := p.selectedQSO(); ok {
		qso.StationCallsign = selected.StationCallsign
	}

	var resultErr error
	if p.settings != nil {
		configured, err := p.settings.Load(p.ctx)
		if err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("load settings: %w", err))
		} else {
			qso.StationCallsign = configured.StationCallsign
		}
	}
	if qso.Callsign == "" || p.lookup == nil {
		return qso, resultErr
	}

	entry, err := p.lookup.Lookup(p.ctx, qso.Callsign)
	if err != nil {
		return qso, errors.Join(
			resultErr,
			fmt.Errorf("look up %s: %w", qso.Callsign, err),
		)
	}
	if record, present := entry.Record.Get(); present {
		qso.Name, _ = record.Name.Get()
		if qso.Name == "" {
			qso.Name, _ = record.Nickname.Get()
		}
		qso.QTH, _ = record.QTH.Get()
	}
	return qso, resultErr
}

func (p *page) openSelectedQSO() {
	qso, ok := p.selectedQSO()
	if !ok {
		return
	}
	editor := newQSOEditor(p.host, qso, p.updateQSO, p.lookup)
	editor.setHandle(p.host.OpenModal(p.Content(), editor))
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
	dialog.setHandle(p.host.OpenModal(p.Content(), dialog))
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
