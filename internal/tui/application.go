package tui

import (
	"context"
	"errors"

	"ditdah/internal/tui/components"
	"ditdah/internal/tui/keybinding"
	"ditdah/internal/tui/modal"
	"ditdah/internal/tui/overlay"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/sync/errgroup"
)

// Application owns the shared TUI infrastructure.
type Application interface {
	PageHost
	OpenModalForCurrentLayer(dialog modal.Dialog) modal.Handle
	AddMenuItem(label string, binding keybinding.Binding)
	AddKeyBinding(binding keybinding.Binding)
	Show(page Page) error
	Run(ctx context.Context, initialPage Page) error
}

type application struct {
	engine              *tview.Application
	layout              Layout
	overlays            overlay.Host
	controls            components.Factory
	theme               colorTheme
	activePage          Page
	globalBindings      []keybinding.Binding
	applicationBindings []keybinding.Binding
	menuItems           []components.MenuItem
	exitBinding         keybinding.Binding
	exitBindingSet      bool
	modals              []*openedModal
	root                tview.Primitive
	appFocusables       []tview.Primitive
	running             bool
	layers              layerRuntime
}

var errPageStopped = errors.New("active page stopped while visible")

type openedModal struct {
	dialog       modal.Dialog
	layer        *modalLayer
	overlay      components.Overlay
	closeBinding keybinding.Binding
	bindings     []keybinding.Binding
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
		layers:   newLayerRuntime(),
	}
	app.root = newMouseFocusGuard(overlays.Root(), app.mousePrimitiveAllowed)
	overlays.SetChangedFunc(app.Refresh)
	return app
}

func (a *application) AddMenuItem(
	label string,
	binding keybinding.Binding,
) {
	a.menuItems = append(a.menuItems, components.MenuItem{
		Label:   label,
		Binding: binding,
	})
}

func (a *application) AddKeyBinding(binding keybinding.Binding) {
	a.applicationBindings = append(a.applicationBindings, binding)
}

func (a *application) buildApplicationMenu() {
	items := append([]components.MenuItem(nil), a.menuItems...)
	if a.running && a.activePage != nil {
		items = append(items, a.activePage.MenuItems()...)
	}
	if a.exitBindingSet {
		items = append(items, components.MenuItem{Separator: true})
		items = append(items, components.MenuItem{
			Label:   "Exit",
			Binding: a.exitBinding,
		})
	}
	functionKeys, _ := keybinding.SplitFunctionBindings(a.applicationBindings)
	menu := newApplicationMenu(a.controls, items, functionKeys)
	a.layout.Header().SetMenu(menu, menu.Width())
	a.appFocusables = []tview.Primitive{menu.button}

	a.globalBindings = append(
		a.globalBindings[:0],
		a.applicationBindings...,
	)
	for _, item := range items {
		if !item.Separator {
			a.globalBindings = append(a.globalBindings, item.Binding)
		}
	}
}

func (a *application) Components() components.Factory {
	return a.controls
}

func (a *application) Show(page Page) error {
	if page == nil {
		return errors.New("show page: nil page")
	}
	if !a.running {
		return errors.New("show page: application is not running")
	}
	a.requestPage(page)
	return nil
}

func (a *application) showPage(page Page) {
	a.activePage = page
	a.layout.Show(page)
	a.buildApplicationMenu()
	a.SetFocus(page.Content())
}

func (a *application) SetFocus(primitive tview.Primitive) {
	a.engine.SetFocus(primitive)
	a.Refresh()
}

func (a *application) OpenModal(
	owner modal.Owner,
	dialog modal.Dialog,
) modal.Handle {
	return a.openModal(owner, true, dialog)
}

// Background starts lifecycle-bound work for a requested page or dialog. It
// rejects stale callbacks whose exact owner layer is no longer requested.
func (a *application) Background(
	owner modal.Owner,
	work BackgroundWork,
) bool {
	if owner == nil || work == nil {
		return false
	}

	return a.layers.startTask(owner, func(layer *runningLayer) error {
		work(layer.ctx)
		return nil
	})
}

// OpenModalForCurrentLayer opens an application-owned modal above whichever
// layer is currently on top. Page and dialog callbacks must use OpenModal so a
// stale callback cannot attach a modal to an unrelated layer.
func (a *application) OpenModalForCurrentLayer(
	dialog modal.Dialog,
) modal.Handle {
	return a.openModal(nil, false, dialog)
}

func (a *application) openModal(
	owner modal.Owner,
	requireOwner bool,
	dialog modal.Dialog,
) modal.Handle {
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
	handle := &modalHandle{app: a, modal: opened}
	opened.closeBinding = keybinding.OnKey(
		tcell.KeyEscape,
		"close",
		handle.Close,
	)
	a.requestModal(owner, requireOwner, opened)
	return handle
}

func (a *application) showModal(opened *openedModal) {
	a.modals = append(a.modals, opened)
	opened.overlay = a.overlays.Push(opened.layer)
	focus := opened.dialog.Content()
	if focusables := opened.dialog.Focusables(); len(focusables) > 0 {
		focus = focusables[0]
	}
	a.SetFocus(focus)
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

	bindings := keybinding.Visible(a.focusedBindings())
	if opened, ok := a.topModal(); ok {
		blocked := a.parentBindingsBlocked()
		bindings = keybinding.Merge(bindings, opened.closeBinding)
		if !blocked {
			bindings = keybinding.Merge(bindings, opened.bindings...)
		}
		if len(opened.dialog.Focusables()) > 1 {
			bindings = keybinding.Merge(
				bindings,
				a.focusNavigationBindings(opened.dialog.Focusables())...,
			)
		}
		a.setKeyBindings(bindings)
		return
	}
	if a.overlays.Active() {
		a.setKeyBindings(bindings)
		return
	}
	if !a.parentBindingsBlocked() {
		bindings = keybinding.Merge(bindings, a.activePage.KeyBindings()...)
		bindings = keybinding.Merge(bindings, a.globalBindings...)
	}
	if len(a.activePage.Focusables()) > 1 {
		bindings = keybinding.Merge(
			bindings,
			a.focusNavigationBindings(a.activePage.Focusables())...,
		)
	}
	a.setKeyBindings(bindings)
}

func (a *application) setKeyBindings(bindings []keybinding.Binding) {
	_, footer := keybinding.SplitFunctionBindings(bindings)
	a.layout.Footer().SetKeyBindings(footer)
}

// Update schedules a layer-owned UI change with tview drawing.
func (a *application) Update(
	owner modal.Owner,
	update func(),
) bool {
	if owner == nil || update == nil {
		return false
	}
	return a.layers.startUpdate(owner, func(layer *runningLayer) error {
		a.engine.QueueUpdateDraw(func() {
			if !a.layers.isRequested(layer) {
				return
			}
			update()
			a.Refresh()
		})
		return nil
	})
}

// update schedules application-owned UI work which is not tied to a layer.
func (a *application) update(update func()) {
	a.engine.QueueUpdateDraw(func() {
		if update != nil {
			update()
		}
		a.Refresh()
	})
}

func (a *application) Run(ctx context.Context, initialPage Page) error {
	if initialPage == nil {
		return errors.New("run terminal application: nil initial page")
	}
	if ctx.Err() != nil {
		return nil
	}
	initialLayer := newPageLayer(initialPage)
	a.layers.initialize(initialLayer)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	a.initializeRuntime(cancel)
	defer func() { a.running = false }()

	var group errgroup.Group
	group.Go(func() error {
		defer cancel()
		return a.engine.
			SetRoot(a.root, true).
			EnableMouse(true).
			Run()
	})

	layerErr := a.layers.Run(runCtx, a)
	cancel()
	a.engine.Stop()
	return errors.Join(layerErr, group.Wait())
}

func (a *application) initializeRuntime(
	cancel context.CancelFunc,
) {
	a.running = true
	a.exitBinding = keybinding.OnRune('q', "quit", cancel)
	a.exitBindingSet = true
	a.buildApplicationMenu()
	a.engine.SetInputCapture(a.runtimeInputCapture(cancel))
}

func (a *application) runtimeInputCapture(
	cancel context.CancelFunc,
) func(*tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			cancel()
			return nil
		}
		return a.captureKey(event)
	}
}

func (a *application) captureKey(event *tcell.EventKey) *tcell.EventKey {
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
	if a.activePage == nil {
		return nil
	}
	return a.activePage.Focusables()
}

func (a *application) mousePrimitiveAllowed(primitive tview.Primitive) bool {
	if primitive == nil {
		return false
	}
	if a.overlays.Active() && primitive == a.overlays.Top() {
		return true
	}
	for _, focusable := range a.activeFocusables() {
		if primitive == focusable {
			return true
		}
	}
	if !a.overlays.Active() {
		for _, focusable := range a.appFocusables {
			if primitive == focusable {
				return true
			}
		}
	}
	return false
}

func (a *application) topModal() (*openedModal, bool) {
	if len(a.modals) == 0 {
		return nil, false
	}
	opened := a.modals[len(a.modals)-1]
	return opened, a.overlays.Top() == opened.layer
}

func (h *modalHandle) Close() {
	h.app.layers.requestModalClose(h.modal)
}

func (a *application) requestPage(page Page) {
	a.layers.requestPage(page)
}

func (a *application) requestModal(
	owner modal.Owner,
	requireOwner bool,
	opened *openedModal,
) {
	a.layers.requestModal(owner, requireOwner, opened)
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
