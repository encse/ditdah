package tui

import (
	"context"
	"errors"
	"fmt"
	"sync"

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
	OpenModalForCurrentLayer(dialog modal.Dialog) modal.Handle
	AddMenuItem(label string, binding keybinding.Binding)
	AddKeyBinding(binding keybinding.Binding)
	Register(page Page) error
	Show(pageID string) error
	NotifySettingsChanged()
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
	runtimeContext      context.Context
	layerChanges        mailbox.Mailbox[layerState]
	layerMu             sync.Mutex
	requestedLayers     []requestedLayer
}

var errPageStopped = errors.New("active page stopped while visible")

type requestedLayer struct {
	instance *layerInstance
	owner    *layerInstance
	page     Page
	modal    *openedModal
}

// layerInstance is deliberately non-zero-sized so distinct allocations always
// have distinct pointer identity.
type layerInstance struct {
	identity byte
}

type layerState struct {
	layers        []requestedLayer
	stoppedPageID string
}

type runningLayer struct {
	requestedLayer
	ctx    context.Context
	cancel context.CancelFunc
	group  *errgroup.Group
}

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
	if a.runtimeContext != nil {
		for _, page := range a.pageOrder {
			items = append(items, page.MenuItems(a.runtimeContext)...)
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
	if a.runtimeContext == nil {
		return errors.New("show page: application is not running")
	}
	a.requestPage(page)
	return nil
}

// NotifySettingsChanged forwards a settings change to the currently visible
// page.
func (a *application) NotifySettingsChanged() {
	if a.activePage != nil {
		a.activePage.SettingsChanged()
	}
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

func (a *application) OpenModal(
	owner tview.Primitive,
	dialog modal.Dialog,
) modal.Handle {
	return a.openModal(owner, true, dialog)
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
	owner tview.Primitive,
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
	initialLayer := newPageLayer(initialPage)
	a.layerChanges = mailbox.New(layerState{
		layers: []requestedLayer{initialLayer},
	})
	a.layerMu.Lock()
	a.requestedLayers = []requestedLayer{initialLayer}
	a.layerMu.Unlock()

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

	layerErr := a.runLayers(runCtx)
	cancel()
	a.engine.Stop()
	return errors.Join(layerErr, group.Wait())
}

func (a *application) initializeRuntime(
	ctx context.Context,
	cancel context.CancelFunc,
) {
	a.runtimeContext = ctx
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

func (a *application) runLayers(ctx context.Context) error {
	// The requested stack always starts with its page. Every following modal
	// runs under a context derived from the layer immediately below it.
	var running []*runningLayer
	for {
		state, err := a.layerChanges.Receive(ctx)
		if err != nil {
			a.stopLayers(running, 0)
			return nil
		}

		common := commonLayerPrefix(running, state.layers)
		a.stopLayers(running, common)
		running = running[:common]
		running = a.startLayers(ctx, running, state.layers[common:])

		if state.stoppedPageID != "" {
			a.stopLayers(running, 0)
			return fmt.Errorf(
				"run page %q: %w",
				state.stoppedPageID,
				errPageStopped,
			)
		}
	}
}

func (a *application) startLayers(
	rootCtx context.Context,
	running []*runningLayer,
	requested []requestedLayer,
) []*runningLayer {
	parentCtx := rootCtx
	if len(running) > 0 {
		parentCtx = running[len(running)-1].ctx
	}
	for _, request := range requested {
		if !a.layerRequestIsCurrent(request) {
			break
		}
		if request.owner != nil &&
			(len(running) == 0 ||
				running[len(running)-1].instance != request.owner) {
			break
		}
		layerCtx, cancel := context.WithCancel(parentCtx)
		group, runCtx := errgroup.WithContext(layerCtx)
		layer := &runningLayer{
			requestedLayer: request,
			ctx:            runCtx,
			cancel:         cancel,
			group:          group,
		}
		a.Update(func() { a.showLayer(request) })
		group.Go(func() error {
			request.run(runCtx)
			if runCtx.Err() == nil {
				a.layerReturned(request)
			}
			return nil
		})
		running = append(running, layer)
		parentCtx = runCtx
	}
	return running
}

func (a *application) stopLayers(running []*runningLayer, from int) {
	if from >= len(running) {
		return
	}
	for _, layer := range running[from:] {
		layer.cancel()
	}
	for index := len(running) - 1; index >= from; index-- {
		_ = running[index].group.Wait()
	}
	a.Update(func() {
		for index := len(running) - 1; index >= from; index-- {
			a.hideLayer(running[index].requestedLayer)
		}
	})
}

func (a *application) showLayer(layer requestedLayer) {
	if layer.modal != nil {
		a.showModal(layer.modal)
		return
	}
	a.showPage(layer.page)
}

func (a *application) hideLayer(layer requestedLayer) {
	if layer.modal == nil {
		return
	}
	for index := len(a.modals) - 1; index >= 0; index-- {
		if a.modals[index] != layer.modal {
			continue
		}
		if layer.modal.overlay != nil {
			layer.modal.overlay.Close()
		}
		a.modals = append(a.modals[:index], a.modals[index+1:]...)
		return
	}
}

func (a *application) layerReturned(layer requestedLayer) {
	if layer.modal != nil {
		a.requestModalClose(layer.modal)
		return
	}
	a.requestLayerStopped(layer)
}

func (r requestedLayer) run(ctx context.Context) {
	if r.modal != nil {
		r.modal.dialog.Run(ctx)
		return
	}
	r.page.Run(ctx)
}

func commonLayerPrefix(
	running []*runningLayer,
	requested []requestedLayer,
) int {
	limit := min(len(running), len(requested))
	for index := 0; index < limit; index++ {
		if !sameLayer(running[index].requestedLayer, requested[index]) {
			return index
		}
	}
	return limit
}

func sameLayer(left, right requestedLayer) bool {
	return left.instance == right.instance
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
	h.app.requestModalClose(h.modal)
}

func (a *application) requestPage(page Page) {
	a.layerMu.Lock()
	a.requestedLayers = []requestedLayer{newPageLayer(page)}
	state := a.layerStateLocked("")
	a.layerMu.Unlock()
	a.layerChanges.Send(state)
}

func (a *application) requestModal(
	owner tview.Primitive,
	requireOwner bool,
	opened *openedModal,
) {
	a.layerMu.Lock()
	if len(a.requestedLayers) == 0 {
		a.layerMu.Unlock()
		return
	}
	parent := a.requestedLayers[len(a.requestedLayers)-1]
	if requireOwner && parent.content() != owner {
		a.layerMu.Unlock()
		return
	}
	a.requestedLayers = append(
		a.requestedLayers,
		requestedLayer{
			instance: &layerInstance{},
			owner:    parent.instance,
			modal:    opened,
		},
	)
	state := a.layerStateLocked("")
	a.layerMu.Unlock()
	a.layerChanges.Send(state)
}

func (a *application) requestModalClose(target *openedModal) {
	a.layerMu.Lock()
	index := -1
	for layerIndex, layer := range a.requestedLayers {
		if layer.modal == target {
			index = layerIndex
			break
		}
	}
	if index < 0 {
		a.layerMu.Unlock()
		return
	}
	a.requestedLayers = a.requestedLayers[:index]
	state := a.layerStateLocked("")
	a.layerMu.Unlock()
	a.layerChanges.Send(state)
}

func (a *application) requestLayerStopped(stopped requestedLayer) {
	a.layerMu.Lock()
	if len(a.requestedLayers) == 0 ||
		a.requestedLayers[0].page == nil ||
		a.requestedLayers[0].instance != stopped.instance {
		a.layerMu.Unlock()
		return
	}
	a.requestedLayers = nil
	state := a.layerStateLocked(stopped.page.ID())
	a.layerMu.Unlock()
	a.layerChanges.Send(state)
}

func newPageLayer(page Page) requestedLayer {
	return requestedLayer{instance: &layerInstance{}, page: page}
}

func (r requestedLayer) content() tview.Primitive {
	if r.modal != nil {
		return r.modal.dialog.Content()
	}
	return r.page.Content()
}

func (a *application) layerRequestIsCurrent(target requestedLayer) bool {
	a.layerMu.Lock()
	defer a.layerMu.Unlock()
	for _, layer := range a.requestedLayers {
		if layer.instance == target.instance {
			return true
		}
	}
	return false
}

func (a *application) layerStateLocked(stoppedPageID string) layerState {
	return layerState{
		layers:        append([]requestedLayer(nil), a.requestedLayers...),
		stoppedPageID: stoppedPageID,
	}
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
