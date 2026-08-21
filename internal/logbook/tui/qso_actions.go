package tui

import (
	"context"
	"errors"
	"fmt"
	"slices"

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
		p.host,
		" Sync QRZ.com ",
		fmt.Sprintf("Upload %d pending QSO(s) to QRZ.com?", pending),
		"Replacement may reset the QRZ confirmation.",
		"Sync",
		p.syncQRZ,
		false,
	)
	dialog.setHandle(p.host.OpenModal(p.Content(), dialog))
}

func (p *page) syncQRZ(ctx context.Context) error {
	if p.syncer == nil {
		return fmt.Errorf("QRZ.com synchronization is unavailable")
	}
	_, syncErr := p.syncer.Sync(ctx)
	qsos, refreshErr := p.list(ctx)
	if refreshErr != nil {
		refreshErr = fmt.Errorf("refresh logbook after QRZ.com sync: %w", refreshErr)
	} else if ctx.Err() == nil {
		p.host.Update(func() {
			p.qsos = qsos
			p.applyFilter()
		})
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
	if p.showNewQSOEditor != nil {
		p.showNewQSOEditor(p.Content(), "")
	}
}

func (p *page) openSelectedQSO() {
	qso, ok := p.selectedQSO()
	if !ok {
		return
	}
	if p.showQSOEditor != nil {
		p.showQSOEditor(p.Content(), qso)
	}
}

func (p *page) confirmDeleteQSO() {
	qso, ok := p.selectedQSO()
	if !ok {
		return
	}
	nextSelectedID := p.selectionAfterDelete(qso.ID)
	dialog := newConfirmDialog(
		p.host,
		" Delete QSO ",
		fmt.Sprintf("Delete QSO with %s?", qso.Callsign),
		"This action cannot be undone.",
		"Delete",
		func(ctx context.Context) error {
			return p.deleteQSO(ctx, qso.ID, nextSelectedID)
		},
	)
	dialog.setHandle(p.host.OpenModal(p.Content(), dialog))
}

func (p *page) QSOChanged(qso domain.QSO) {
	for index := range p.qsos {
		if p.qsos[index].ID == qso.ID {
			p.qsos[index] = qso
			p.selectedID = qso.ID
			p.applyFilter()
			return
		}
	}
	p.qsos = append(p.qsos, qso)
	p.selectedID = qso.ID
	p.applyFilter()
}

func (p *page) deleteQSO(
	ctx context.Context,
	id string,
	nextSelectedID string,
) error {
	if err := p.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete QSO: %w", err)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	p.host.Update(func() {
		p.qsos = slices.DeleteFunc(p.qsos, func(qso domain.QSO) bool {
			return qso.ID == id
		})
		p.selectedID = nextSelectedID
		p.applyFilter()
	})
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
