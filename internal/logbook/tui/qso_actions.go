package tui

import (
	"context"
	"errors"
	"fmt"
	"slices"

	domain "morsemanual/internal/logbook"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

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
	modal.OpenConfirm(
		p.host,
		p,
		" Sync QRZ.com ",
		fmt.Sprintf("Upload %d pending QSO(s) to QRZ.com?", pending),
		"Replacement may reset the QRZ confirmation.",
		"Sync",
		p.startQRZSync,
	)
}

func (p *page) startQRZSync() {
	p.host.Background(p, func(ctx context.Context) {
		if p.syncer == nil {
			p.showActionError(errors.New("QRZ.com synchronization is unavailable"))
			return
		}
		_, syncErr := p.syncer.Sync(ctx)
		qsos, refreshErr := p.list(ctx)
		if refreshErr != nil {
			refreshErr = fmt.Errorf("refresh logbook after QRZ.com sync: %w", refreshErr)
		}
		if ctx.Err() != nil {
			return
		}
		p.host.Update(p, func() {
			if err := errors.Join(syncErr, refreshErr); err != nil {
				p.openActionError(err)
				return
			}
			p.qsos = qsos
			p.applyFilter()
		})
	})
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
		p.showNewQSOEditor(p, "")
	}
}

func (p *page) openSelectedQSO() {
	qso, ok := p.selectedQSO()
	if !ok {
		return
	}
	if p.showQSOEditor != nil {
		p.showQSOEditor(p, qso)
	}
}

func (p *page) confirmDeleteQSO() {
	qso, ok := p.selectedQSO()
	if !ok {
		return
	}
	nextSelectedID := p.selectionAfterDelete(qso.ID)
	modal.OpenDangerConfirm(
		p.host,
		p,
		" Delete QSO ",
		fmt.Sprintf("Delete QSO with %s?", qso.Callsign),
		"This action cannot be undone.",
		"Delete",
		func() {
			p.startDeleteQSO(qso.ID, nextSelectedID)
		},
	)
}

func (p *page) startDeleteQSO(
	id string,
	nextSelectedID string,
) {
	p.host.Background(p, func(ctx context.Context) {
		err := p.store.Delete(ctx, id)
		if ctx.Err() != nil {
			return
		}
		p.host.Update(p, func() {
			if err != nil {
				p.openActionError(fmt.Errorf("delete QSO: %w", err))
				return
			}
			p.qsos = slices.DeleteFunc(p.qsos, func(qso domain.QSO) bool {
				return qso.ID == id
			})
			p.selectedID = nextSelectedID
			p.applyFilter()
		})
	})
}

func (p *page) showActionError(err error) {
	if err == nil {
		return
	}
	p.host.Update(p, func() { p.openActionError(err) })
}

func (p *page) openActionError(err error) {
	modal.OpenError(
		p.host,
		p,
		" Error ",
		"Error: "+err.Error(),
	)
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
