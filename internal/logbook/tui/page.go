// Package tui implements the logbook terminal user interface.
package tui

import (
	"context"
	"fmt"

	domain "morsemanual/internal/logbook"
	ui "morsemanual/internal/tui"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"

	"github.com/rivo/tview"
)

const qsoPageSize = 500

type page struct {
	host             ui.PageHost
	store            domain.Store
	syncer           qrzSynchronizer
	showNewQSOEditor func(tview.Primitive, string)
	showQSOEditor    func(tview.Primitive, domain.QSO)

	content tview.Primitive
	search  components.InputField
	table   components.Table
	details *detailsView

	qsos         []domain.QSO
	filteredQsos []domain.QSO
	selectedID   string
}

type Page interface {
	ui.Page
	QSOChanged(domain.QSO)
}

// New creates the logbook page.
func New(
	host ui.PageHost,
	store domain.Store,
	syncer qrzSynchronizer,
	showNewQSOEditor func(tview.Primitive, string),
	showQSOEditor func(tview.Primitive, domain.QSO),
) Page {
	return newPage(
		host,
		store,
		syncer,
		showNewQSOEditor,
		showQSOEditor,
	)
}

func newPage(
	host ui.PageHost,
	store domain.Store,
	syncer qrzSynchronizer,
	showNewQSOEditor func(tview.Primitive, string),
	showQSOEditor func(tview.Primitive, domain.QSO),
) *page {
	controls := host.Components()
	page := &page{
		host:             host,
		store:            store,
		syncer:           syncer,
		showNewQSOEditor: showNewQSOEditor,
		showQSOEditor:    showQSOEditor,
	}
	page.search = page.newSearch(controls)
	page.table = page.newTable(controls)
	page.details = newDetailsView(controls, " QSO info ")
	page.content = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(page.search, 1, 0, false).
		AddItem(page.table, 0, 2, true).
		AddItem(page.details, 9, 0, false)
	return page
}

func (p *page) ID() string {
	return "logbook"
}

func (p *page) Title() string {
	return "Logbook"
}

func (p *page) Content() tview.Primitive {
	return p.content
}

func (p *page) Focusables() []tview.Primitive {
	return []tview.Primitive{p.search, p.table, p.details.TextView}
}

func (p *page) KeyBindings() []keybinding.Binding {
	return append(
		p.searchBindings(),
		p.createBinding(),
		p.editBinding(),
		p.deleteBinding(),
		p.syncBinding(),
	)
}

func (p *page) MenuItems() []components.MenuItem { return nil }

func (p *page) SettingsChanged() {}

func (p *page) Run(ctx context.Context) {
	qsos, err := p.list(ctx)
	if ctx.Err() != nil {
		return
	}
	p.host.Update(p.Content(), func() {
		if err != nil {
			p.details.setRows([]detailRow{{
				left: detailField{value: "Error: " + err.Error()},
			}})
			return
		}
		p.qsos = qsos
		p.applyFilter()
	})
	<-ctx.Done()
}

func (p *page) Status() string {
	return fmt.Sprintf("(%d/%d)", len(p.filteredQsos), len(p.qsos))
}

func (p *page) list(ctx context.Context) ([]domain.QSO, error) {
	qsos := make([]domain.QSO, 0)
	for offset := 0; ; offset += qsoPageSize {
		result, err := p.store.List(ctx, domain.Filter{
			Limit:  qsoPageSize,
			Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("load logbook: %w", err)
		}
		qsos = append(qsos, result...)
		if len(result) < qsoPageSize {
			break
		}
	}

	return qsos, nil
}

func (p *page) refreshView() {
	selectedIndex := p.renderTable(p.selectedID)
	if selectedIndex >= 0 {
		p.selectedID = p.filteredQsos[selectedIndex].ID
		p.table.Select(selectedIndex+1, 0)
	} else {
		p.selectedID = ""
	}
	p.renderDetails(selectedIndex)
	p.host.Refresh()
}
