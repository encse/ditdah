// Package logbook implements the logbook terminal page.
package logbook

import (
	"context"
	"fmt"

	domain "morsemanual/internal/logbook"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"

	"github.com/rivo/tview"
)

const qsoPageSize = 500

// Host is the part of the terminal application used by the logbook page.
type Host interface {
	SetFocus(primitive tview.Primitive)
	Refresh()
	Components() components.Factory
}

// Page is the logbook page exposed to the terminal application.
type Page interface {
	ID() string
	Title() string
	Content() tview.Primitive
	KeyBindings() []keybinding.Binding
	Status() string
}

type page struct {
	ctx   context.Context
	host  Host
	store domain.Store

	content tview.Primitive
	search  components.InputField
	table   components.Table
	details *detailsView

	qsos         []domain.QSO
	filteredQsos []domain.QSO
	selectedID   string
}

// New creates and loads a logbook page.
func New(ctx context.Context, host Host, store domain.Store) (Page, error) {
	page := newPage(ctx, host, store)
	if err := page.load(); err != nil {
		return nil, err
	}
	return page, nil
}

func newPage(ctx context.Context, host Host, store domain.Store) *page {
	controls := host.Components()
	page := &page{
		ctx:   ctx,
		host:  host,
		store: store,
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

func (p *page) KeyBindings() []keybinding.Binding {
	return p.searchBindings()
}

func (p *page) Status() string {
	return fmt.Sprintf("(%d/%d)", len(p.filteredQsos), len(p.qsos))
}

func (p *page) load() error {
	var qsos []domain.QSO
	for offset := 0; ; offset += qsoPageSize {
		result, err := p.store.List(p.ctx, domain.Filter{
			Limit:  qsoPageSize,
			Offset: offset,
		})
		if err != nil {
			return fmt.Errorf("load logbook: %w", err)
		}
		qsos = append(qsos, result...)
		if len(result) < qsoPageSize {
			break
		}
	}

	p.qsos = qsos
	p.applyFilter()
	return nil
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
