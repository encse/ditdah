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

// PageHost is the part of the TUI application available to pages.
type PageHost interface {
	SetFocus(primitive tview.Primitive)
	Refresh()
	Components() components.Factory
	Theme() colorTheme
	OpenModal(dialog modal.Dialog) modal.Handle
}

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
	dialog  modal.Dialog
	layer   *modalLayer
	overlay components.Overlay
	closed  bool
}

type modalHandle struct {
	app   *application
	modal *openedModal
}

// NewApplication creates the terminal application infrastructure.
func NewApplication(theme colorTheme) Application {
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

func (a *application) Theme() colorTheme {
	return a.theme
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
	opened := &openedModal{dialog: dialog, layer: layer}
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

	hints := a.focusedHints()
	if opened, ok := a.topModal(); ok {
		if !a.parentBindingsBlocked() {
			hints = append(hints, keybinding.Hints(opened.dialog.KeyBindings())...)
		}
		hints = append(hints, keybinding.Hint{Keys: "Esc", Description: "close"})
		if len(opened.dialog.Focusables()) > 1 {
			hints = append(hints, focusNavigationHint())
		}
		a.layout.Footer().SetKeyHints(hints)
		return
	}
	if a.overlays.Active() {
		a.layout.Footer().SetKeyHints(hints)
		return
	}
	if !a.parentBindingsBlocked() {
		hints = append(hints, keybinding.Hints(a.activePage.KeyBindings())...)
		hints = append(hints, keybinding.Hints(a.globalBindings)...)
	}
	if len(a.activePage.Focusables()) > 1 {
		hints = append(hints, focusNavigationHint())
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
	return keybinding.Binding{
		Hint: keybinding.Hint{Keys: "q", Description: "quit"},
		Handler: func(event *tcell.EventKey) bool {
			if event.Key() != tcell.KeyRune || event.Rune() != 'q' {
				return false
			}
			a.Stop()
			return true
		},
	}
}

func (a *application) captureKey(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyCtrlC {
		a.Stop()
		return nil
	}
	if a.activePage == nil {
		return event
	}
	if a.overlays.Active() {
		opened, ok := a.topModal()
		if !ok {
			return event
		}
		return a.captureModal(opened, event)
	}
	if a.moveFocus(event, a.activePage.Focusables()) {
		return nil
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
	if event.Key() == tcell.KeyEscape {
		a.closeModal(opened)
		return nil
	}
	if a.moveFocus(event, opened.dialog.Focusables()) {
		return nil
	}
	if a.parentBindingsBlocked() {
		return event
	}
	for _, binding := range opened.dialog.KeyBindings() {
		if binding.Handle(event) {
			a.Refresh()
			return nil
		}
	}
	return event
}

func (a *application) moveFocus(
	event *tcell.EventKey,
	focusables []tview.Primitive,
) bool {
	direction := 0
	switch event.Key() {
	case tcell.KeyTab:
		direction = 1
	case tcell.KeyBacktab:
		direction = -1
	default:
		return false
	}

	if len(focusables) == 0 {
		return false
	}
	current := a.engine.GetFocus()
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

func focusNavigationHint() keybinding.Hint {
	return keybinding.Hint{
		Keys:        "Tab/Shift+Tab",
		Description: "next/previous",
	}
}

func (a *application) focusedHints() []keybinding.Hint {
	provider, ok := a.engine.GetFocus().(keybinding.HintProvider)
	if !ok {
		return nil
	}
	return provider.KeyHints()
}

func (a *application) parentBindingsBlocked() bool {
	blocker, ok := a.engine.GetFocus().(keybinding.ParentBindingBlocker)
	return ok && blocker.BlocksParentBindings()
}
