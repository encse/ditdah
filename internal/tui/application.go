package tui

import (
	"context"
	"fmt"

	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"
	"morsemanual/internal/tui/overlay"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/sync/errgroup"
)

// Application owns the shared TUI infrastructure and registered pages.
type Application interface {
	PageHost
	Register(page Page) error
	Show(pageID string) error
	Run(ctx context.Context) error
	Stop()
}

type application struct {
	engine         *tview.Application
	layout         Layout
	overlays       overlay.Host
	controls       components.Factory
	theme          colorTheme
	pages          map[string]Page
	activePage     Page
	globalBindings []keybinding.Binding
	modals         []*openedModal
}

type openedModal struct {
	dialog       modal.Dialog
	layer        *modalLayer
	overlay      components.Overlay
	closeBinding keybinding.Binding
	bindings     []keybinding.Binding
	closed       bool
}

type modalHandle struct {
	app   *application
	modal *openedModal
}

// NewApplication creates the terminal application infrastructure.
func NewApplication() Application {
	return newApplication(nordTheme)
}

func newApplication(theme colorTheme) Application {
	tview.Styles = theme.styles
	engine := tview.NewApplication()
	overlays := overlay.New(engine)
	var app *application
	controls := components.New(components.Dependencies{
		Theme:      theme.components(),
		ModalTheme: theme.modalComponents(),
		Overlays:   overlays,
		FocusChanged: func() {
			if app != nil {
				app.Refresh()
			}
		},
	})
	layout := NewLayout(controls)
	overlays.SetContent(layout)

	app = &application{
		engine:   engine,
		layout:   layout,
		overlays: overlays,
		controls: controls,
		theme:    theme,
		pages:    make(map[string]Page),
	}
	app.globalBindings = []keybinding.Binding{app.quitBinding()}
	engine.SetInputCapture(app.captureKey)
	overlays.SetChangedFunc(app.Refresh)
	return app
}

func (a *application) Components() components.Factory {
	return a.controls
}

func (a *application) Register(page Page) error {
	id := page.ID()
	if id == "" {
		return fmt.Errorf("register page: empty ID")
	}
	if _, exists := a.pages[id]; exists {
		return fmt.Errorf("register page %q: duplicate ID", id)
	}
	a.pages[id] = page
	return nil
}

func (a *application) Show(pageID string) error {
	page, exists := a.pages[pageID]
	if !exists {
		return fmt.Errorf("show page %q: not registered", pageID)
	}
	a.activePage = page
	a.layout.Show(page)
	a.SetFocus(page.Content())
	return nil
}

func (a *application) SetFocus(primitive tview.Primitive) {
	a.engine.SetFocus(primitive)
	a.Refresh()
}

func (a *application) OpenModal(dialog modal.Dialog) modal.Handle {
	size := dialog.Size()
	layer := newModalLayer(
		dialog.Content(),
		size.Width,
		size.Height,
		a.theme.styles.BorderColor,
		a.theme.styles.PrimitiveBackgroundColor,
	)
	opened := &openedModal{
		dialog:   dialog,
		layer:    layer,
		bindings: dialog.KeyBindings(),
	}
	opened.closeBinding = keybinding.OnKey(
		tcell.KeyEscape,
		"close",
		func() { a.closeModal(opened) },
	)
	a.modals = append(a.modals, opened)
	opened.overlay = a.overlays.Push(layer)
	focus := dialog.Content()
	if focusables := dialog.Focusables(); len(focusables) > 0 {
		focus = focusables[0]
	}
	a.SetFocus(focus)
	return &modalHandle{app: a, modal: opened}
}

func (a *application) Refresh() {
	if a.activePage == nil {
		return
	}

	if status, ok := a.activePage.(interface{ Status() string }); ok {
		a.layout.Header().SetStatus(status.Status())
	} else {
		a.layout.Header().SetStatus("")
	}

	hints := keybinding.Hints(a.focusedBindings())
	if opened, ok := a.topModal(); ok {
		blocked := a.parentBindingsBlocked()
		hints = keybinding.MergeBindingHints(hints, opened.closeBinding)
		if !blocked {
			hints = keybinding.MergeBindingHints(hints, opened.bindings...)
		}
		if len(opened.dialog.Focusables()) > 1 {
			hints = keybinding.MergeBindingHints(
				hints,
				a.focusNavigationBindings(opened.dialog.Focusables())...,
			)
		}
		a.layout.Footer().SetKeyHints(hints)
		return
	}
	if a.overlays.Active() {
		a.layout.Footer().SetKeyHints(hints)
		return
	}
	if !a.parentBindingsBlocked() {
		hints = keybinding.MergeBindingHints(hints, a.activePage.KeyBindings()...)
		hints = keybinding.MergeBindingHints(hints, a.globalBindings...)
	}
	if len(a.activePage.Focusables()) > 1 {
		hints = keybinding.MergeBindingHints(
			hints,
			a.focusNavigationBindings(a.activePage.Focusables())...,
		)
	}
	a.layout.Footer().SetKeyHints(hints)
}

func (a *application) Run(ctx context.Context) error {
	if a.activePage == nil {
		return fmt.Errorf("run terminal application: no active page")
	}
	if ctx.Err() != nil {
		return nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	group, groupCtx := errgroup.WithContext(runCtx)
	group.Go(func() error {
		defer cancel()
		return a.engine.
			SetRoot(a.overlays.Root(), true).
			EnableMouse(true).
			Run()
	})
	group.Go(func() error {
		<-groupCtx.Done()
		if ctx.Err() != nil {
			a.Stop()
		}
		return nil
	})

	return group.Wait()
}

func (a *application) Stop() {
	a.engine.Stop()
}

func (a *application) quitBinding() keybinding.Binding {
	return keybinding.OnRune('q', "quit", a.Stop)
}

func (a *application) captureKey(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyCtrlC {
		a.Stop()
		return nil
	}
	if a.activePage == nil {
		return event
	}
	for _, binding := range a.focusNavigationBindings(a.activeFocusables()) {
		if binding.Handle(event) {
			return nil
		}
	}
	if a.overlays.Active() {
		opened, ok := a.topModal()
		if !ok {
			return event
		}
		return a.captureModal(opened, event)
	}
	if a.parentBindingsBlocked() {
		return event
	}

	for _, binding := range a.activePage.KeyBindings() {
		if binding.Handle(event) {
			a.Refresh()
			return nil
		}
	}
	for _, binding := range a.globalBindings {
		if binding.Handle(event) {
			a.Refresh()
			return nil
		}
	}
	return event
}

func (a *application) captureModal(
	opened *openedModal,
	event *tcell.EventKey,
) *tcell.EventKey {
	if opened.closeBinding.Handle(event) {
		a.Refresh()
		return nil
	}
	if a.parentBindingsBlocked() {
		return event
	}
	for index := len(opened.bindings) - 1; index >= 0; index-- {
		if opened.bindings[index].Handle(event) {
			a.Refresh()
			return nil
		}
	}
	return event
}

func (a *application) focusNavigationBindings(
	focusables []tview.Primitive,
) []keybinding.Binding {
	return []keybinding.Binding{
		keybinding.OnKey(
			tcell.KeyTab,
			"next",
			func() { a.focusRelative(focusables, 1) },
		),
		keybinding.OnKey(
			tcell.KeyBacktab,
			"previous",
			func() { a.focusRelative(focusables, -1) },
		),
	}
}

func (a *application) focusRelative(
	focusables []tview.Primitive,
	direction int,
) bool {
	if len(focusables) == 0 {
		return false
	}
	current := a.engine.GetFocus()
	if current == a.overlays.Top() {
		current = a.overlays.FocusBeforeTop()
	}
	currentIndex := -1
	for index, primitive := range focusables {
		if primitive == current {
			currentIndex = index
			break
		}
	}
	nextIndex := currentIndex + direction
	if currentIndex < 0 && direction < 0 {
		nextIndex = len(focusables) - 1
	}
	nextIndex = (nextIndex + len(focusables)) % len(focusables)
	a.SetFocus(focusables[nextIndex])
	return true
}

func (a *application) activeFocusables() []tview.Primitive {
	if len(a.modals) > 0 {
		return a.modals[len(a.modals)-1].dialog.Focusables()
	}
	return a.activePage.Focusables()
}

func (a *application) topModal() (*openedModal, bool) {
	if len(a.modals) == 0 {
		return nil, false
	}
	opened := a.modals[len(a.modals)-1]
	return opened, a.overlays.Top() == opened.layer
}

func (a *application) closeModal(target *openedModal) {
	index := -1
	for modalIndex, opened := range a.modals {
		if opened == target {
			index = modalIndex
			break
		}
	}
	if index < 0 || target.closed {
		return
	}
	for _, opened := range a.modals[index:] {
		opened.closed = true
	}
	a.modals = a.modals[:index]
	target.overlay.Close()
}

func (h *modalHandle) Close() {
	h.app.closeModal(h.modal)
}

func (a *application) focusedBindings() []keybinding.Binding {
	provider, ok := a.engine.GetFocus().(keybinding.BindingProvider)
	if !ok {
		return nil
	}
	return provider.KeyBindings()
}

func (a *application) parentBindingsBlocked() bool {
	blocker, ok := a.engine.GetFocus().(keybinding.ParentBindingBlocker)
	return ok && blocker.BlocksParentBindings()
}
