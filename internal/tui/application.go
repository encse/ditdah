package tui

import (
	"context"
	"errors"
	"fmt"

	"morsemanual/internal/mailbox"
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
	AddMenuItem(label string, binding keybinding.Binding)
	AddKeyBinding(binding keybinding.Binding)
	Register(page Page) error
	Show(pageID string) error
	Run(ctx context.Context, initialPageID string) error
}

type application struct {
	engine              *tview.Application
	layout              Layout
	overlays            overlay.Host
	controls            components.Factory
	theme               colorTheme
	pages               map[string]Page
	pageOrder           []Page
	activePage          Page
	globalBindings      []keybinding.Binding
	applicationBindings []keybinding.Binding
	menuItems           []components.MenuItem
	exitBinding         keybinding.Binding
	exitBindingSet      bool
	modals              []*openedModal
	root                tview.Primitive
	appFocusables       []tview.Primitive
	menuContext         context.Context
	pageChanges         mailbox.Mailbox[Page]
}

var errPageChanged = errors.New("active page changed")
var errPageStopped = errors.New("active page stopped while visible")

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
	if a.menuContext != nil {
		for _, page := range a.pageOrder {
			items = append(items, page.MenuItems(a.menuContext)...)
		}
	}
	if a.exitBindingSet {
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
		a.globalBindings = append(a.globalBindings, item.Binding)
	}
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
	a.pageOrder = append(a.pageOrder, page)
	return nil
}

func (a *application) Show(pageID string) error {
	page, exists := a.pages[pageID]
	if !exists {
		return fmt.Errorf("show page %q: not registered", pageID)
	}
	if a.pageChanges == nil {
		return errors.New("show page: application is not running")
	}
	a.pageChanges.Send(page)
	return nil
}

func (a *application) showPage(page Page) {
	a.activePage = page
	a.layout.Show(page)
	a.SetFocus(page.Content())
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
	// Modals are opened from direct event handling. Drawing here makes the
	// modal visible before the caller starts any following synchronous work.
	a.engine.ForceDraw()
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

// Update serializes a background update with tview drawing.
func (a *application) Update(update func()) {
	a.engine.QueueUpdateDraw(func() {
		if update != nil {
			update()
		}
		a.Refresh()
	})
}

func (a *application) Run(ctx context.Context, initialPageID string) error {
	initialPage, exists := a.pages[initialPageID]
	if !exists {
		return fmt.Errorf(
			"run terminal application: page %q is not registered",
			initialPageID,
		)
	}
	if ctx.Err() != nil {
		return nil
	}
	a.pageChanges = mailbox.New[Page](initialPage)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	a.initializeRuntime(runCtx, cancel)

	var group errgroup.Group
	group.Go(func() error {
		defer cancel()
		return a.engine.
			SetRoot(a.root, true).
			EnableMouse(true).
			Run()
	})

	pageErr := a.runPages(runCtx)
	cancel()
	a.engine.Stop()
	return errors.Join(pageErr, group.Wait())
}

func (a *application) initializeRuntime(
	ctx context.Context,
	cancel context.CancelFunc,
) {
	a.menuContext = ctx
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

func (a *application) runPages(ctx context.Context) error {
	page, err := a.pageChanges.Receive(ctx)
	if err != nil {
		return err
	}
	for ctx.Err() == nil {
		a.Update(func() {
			a.showPage(page)
		})

		var nextPage Page
		group, pageCtx := errgroup.WithContext(ctx)
		group.Go(func() error {
			page.Run(pageCtx)
			return errPageStopped
		})
		group.Go(func() error {
			var err error
			nextPage, err = a.pageChanges.Receive(pageCtx)
			if err != nil {
				return err
			}
			return errPageChanged
		})

		err = group.Wait()
		if ctx.Err() != nil {
			return nil
		}
		if !errors.Is(err, errPageChanged) {
			return fmt.Errorf("run page %q: %w", page.ID(), err)
		}

		page = nextPage
	}
	return nil
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
