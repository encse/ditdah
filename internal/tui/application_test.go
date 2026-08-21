package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestApplicationRegistersAndShowsPage(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	content := tview.NewBox()
	page := applicationTestPage{
		id:      "logbook",
		title:   "Logbook",
		status:  "24 QSOs",
		content: content,
	}

	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	cancel, runDone := startTestApplication(t, app, page.ID())
	defer finishTestApplication(t, cancel, runDone)

	if app.activePage.ID() != page.ID() {
		t.Fatalf("active page = %q, want %q", app.activePage.ID(), page.ID())
	}
	if got := app.engine.GetFocus(); got != content {
		t.Fatalf("focus = %T, want page content", got)
	}
}

func TestApplicationNotifiesOnlyActivePageAboutSettingsChanges(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	active := &settingsAwareApplicationTestPage{applicationTestPage: applicationTestPage{
		id: "logbook", title: "Logbook", content: tview.NewBox(),
	}}
	inactive := &settingsAwareApplicationTestPage{applicationTestPage: applicationTestPage{
		id: "decoder", title: "Decoder", content: tview.NewBox(),
	}}
	if err := app.Register(active); err != nil {
		t.Fatal(err)
	}
	if err := app.Register(inactive); err != nil {
		t.Fatal(err)
	}
	app.showPage(active)

	app.NotifySettingsChanged()

	if active.settingsChanges != 1 {
		t.Fatalf("active page notifications = %d, want 1", active.settingsChanges)
	}
	if inactive.settingsChanges != 0 {
		t.Fatalf("inactive page notifications = %d, want 0", inactive.settingsChanges)
	}
}

func TestHeaderStatusClickCannotStealPageFocusOrMouseCapture(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	content := tview.NewBox()
	page := applicationTestPage{
		id:         "logbook",
		title:      "Logbook",
		status:     "(28/28)",
		content:    content,
		focusables: []tview.Primitive{content},
	}
	if err := app.Register(page); err != nil {
		t.Fatal(err)
	}
	app.showPage(page)
	app.Refresh()

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(80, 8)
	app.root.SetRect(0, 0, 80, 8)
	app.root.Draw(screen)

	before := app.engine.GetFocus()
	consumed, capture := app.root.MouseHandler()(
		tview.MouseLeftDown,
		tcell.NewEventMouse(78, 0, tcell.Button1, 0),
		func(primitive tview.Primitive) { app.engine.SetFocus(primitive) },
	)
	if !consumed {
		t.Fatal("header status click was not consumed")
	}
	if capture != nil {
		t.Fatalf("header status capture = %T, want nil", capture)
	}
	if got := app.engine.GetFocus(); got != before {
		t.Fatalf("focus after header click = %T, want previous %T", got, before)
	}
}

func TestApplicationMenuItemHandlesPhysicalMouseClick(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	selected := make(chan struct{}, 1)
	app.AddMenuItem(
		"Settings",
		keybinding.OnRune('s', "settings", func() { selected <- struct{}{} }),
	)
	page := applicationTestPage{
		id:      "logbook",
		title:   "Logbook",
		content: tview.NewBox(),
	}
	if err := app.Register(page); err != nil {
		t.Fatal(err)
	}

	drawn := make(chan struct{}, 8)
	menuDrawn := make(chan struct{}, 1)
	app.engine.SetBeforeDrawFunc(func(tcell.Screen) bool {
		if app.activePage != nil {
			select {
			case drawn <- struct{}{}:
			default:
			}
		}
		if app.overlays.Active() {
			select {
			case menuDrawn <- struct{}{}:
			default:
			}
		}
		return false
	})
	screen := tcell.NewSimulationScreen("UTF-8")
	app.engine.SetScreen(screen)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx, page.ID()) }()
	waitForApplicationSignal(t, drawn, "initial application draw")

	// One physical click on the hamburger.
	screen.InjectMouse(2, 0, tcell.Button1, 0)
	screen.InjectMouse(2, 0, tcell.ButtonNone, 0)
	waitForApplicationSignal(t, menuDrawn, "open menu draw")

	// One physical click on the first menu item.
	screen.InjectMouse(2, 2, tcell.Button1, 0)
	screen.InjectMouse(2, 2, tcell.ButtonNone, 0)
	waitForApplicationSignal(t, selected, "menu item callback")
	finishTestApplication(t, cancel, done)
}

func TestApplicationRejectsDuplicateAndUnknownPages(t *testing.T) {
	app := newApplication(nordTheme)
	page := applicationTestPage{
		id:      "logbook",
		title:   "Logbook",
		content: tview.NewBox(),
	}
	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	if err := app.Register(page); err == nil {
		t.Fatal("duplicate page registration succeeded")
	}
	if err := app.Show("missing"); err == nil {
		t.Fatal("showing unknown page succeeded")
	}
}

func TestApplicationOwnsQuitBinding(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	stopped := 0
	app.initializeRuntime(func() { stopped++ })
	binding := app.exitBinding
	if binding.Hint() != (keybinding.Hint{Keys: "q", Description: "quit"}) {
		t.Fatalf("quit hint = %#v", binding.Hint())
	}
	if binding.Handle(tcell.NewEventKey(tcell.KeyRune, 'x', 0)) {
		t.Fatal("quit binding handled unrelated key")
	}
	if !binding.Handle(tcell.NewEventKey(tcell.KeyRune, 'q', 0)) {
		t.Fatal("quit binding did not handle q")
	}
	if stopped != 1 {
		t.Fatalf("stop callbacks = %d, want 1", stopped)
	}
}

func TestApplicationMenuItemsAlsoRegisterGlobalBindings(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	opened := 0
	app.AddMenuItem(
		"Settings",
		keybinding.OnRune('s', "settings", func() { opened++ }),
	)
	app.buildApplicationMenu()

	if len(app.menuItems) != 1 || app.menuItems[0].Label != "Settings" {
		t.Fatalf("menu items = %#v", app.menuItems)
	}
	if len(app.globalBindings) != 1 {
		t.Fatalf("global binding count = %d, want 1", len(app.globalBindings))
	}
	if !app.globalBindings[0].Handle(
		tcell.NewEventKey(tcell.KeyRune, 's', 0),
	) {
		t.Fatal("settings binding did not handle s")
	}
	if opened != 1 {
		t.Fatalf("settings opens = %d, want 1", opened)
	}
}

func TestApplicationPlacesFunctionBindingsOnlyInHeader(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	page := applicationTestPage{
		id:      "logbook",
		title:   "Logbook",
		content: tview.NewBox(),
	}
	if err := app.Register(page); err != nil {
		t.Fatal(err)
	}
	app.AddKeyBinding(keybinding.OnKey(tcell.KeyF1, "Logbook", func() {}))
	app.AddKeyBinding(keybinding.OnKey(tcell.KeyF2, "Morse decoder", func() {}))
	app.AddKeyBinding(keybinding.OnRune('q', "quit", func() {}))
	app.buildApplicationMenu()
	app.showPage(page)
	app.Refresh()

	header := drawApplicationLine(t, app, 0)
	footer := drawApplicationLine(t, app, 5)
	if !strings.Contains(header, "F1 Logbook") {
		t.Fatalf("header = %q, want F1 Logbook", header)
	}
	if strings.Contains(footer, "F1 Logbook") {
		t.Fatalf("footer = %q, contains F1 Logbook", footer)
	}
	if !strings.Contains(footer, "q quit") {
		t.Fatalf("footer = %q, want q quit", footer)
	}
	if len(app.menuItems) != 0 {
		t.Fatalf("menu items = %#v, want no function-key menu item", app.menuItems)
	}
}

func TestRegisteredPageContributesApplicationMenuItems(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	handled := 0
	page := applicationTestPage{
		id:      "decoder",
		title:   "Decoder",
		content: tview.NewBox(),
		menuItems: []components.MenuItem{{
			Label: "Morse input",
			Binding: keybinding.OnRune('i', "Morse input", func() {
				handled++
			}),
		}},
	}

	if err := app.Register(page); err != nil {
		t.Fatal(err)
	}
	app.running = true
	app.buildApplicationMenu()
	if len(app.globalBindings) != 1 {
		t.Fatalf("global bindings = %d, want 1", len(app.globalBindings))
	}
	if got := app.globalBindings[0].Hint().Description; got != "Morse input" {
		t.Fatalf("menu binding description = %q, want Morse input", got)
	}
	if !app.globalBindings[0].Handle(
		tcell.NewEventKey(tcell.KeyRune, 'i', 0),
	) {
		t.Fatal("page menu binding did not handle i")
	}
	if handled != 1 {
		t.Fatalf("handled = %d, want 1", handled)
	}
}

func TestApplicationDispatchesBindingsByContext(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	pageHandled := 0
	globalHandled := 0
	page := applicationTestPage{
		id:      "logbook",
		title:   "Logbook",
		content: tview.NewBox(),
		bindings: []keybinding.Binding{
			bindingForRune("page", 'p', &pageHandled),
		},
	}
	app.globalBindings = []keybinding.Binding{
		bindingForRune("global", 'q', &globalHandled),
	}
	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	app.showPage(page)

	if got := app.captureKey(tcell.NewEventKey(tcell.KeyRune, 'p', 0)); got != nil {
		t.Fatal("handled page binding was forwarded")
	}
	if got := app.captureKey(tcell.NewEventKey(tcell.KeyRune, 'q', 0)); got != nil {
		t.Fatal("handled global binding was forwarded")
	}
	if pageHandled != 1 || globalHandled != 1 {
		t.Fatalf("handled counts = page %d, global %d, want 1 and 1", pageHandled, globalHandled)
	}

	input := app.controls.InputField("Search", "")
	input.SetBindings(keybinding.OnKey(
		tcell.KeyEnter,
		"done",
		func() {},
	))
	app.SetFocus(input)
	pageEvent := tcell.NewEventKey(tcell.KeyRune, 'p', 0)
	if got := app.captureKey(pageEvent); got != pageEvent {
		t.Fatal("text input did not receive page binding rune")
	}
	globalEvent := tcell.NewEventKey(tcell.KeyRune, 'q', 0)
	if got := app.captureKey(globalEvent); got != globalEvent {
		t.Fatal("text input did not receive global binding rune")
	}
	stopped := 0
	capture := app.runtimeInputCapture(func() { stopped++ })
	if got := capture(tcell.NewEventKey(tcell.KeyCtrlC, 0, 0)); got != nil {
		t.Fatal("application Ctrl-C was forwarded")
	}
	if stopped != 1 {
		t.Fatalf("Ctrl-C stop callbacks = %d, want 1", stopped)
	}
	if pageHandled != 1 || globalHandled != 1 {
		t.Fatalf("input handled counts = page %d, global %d, want 1 and 1", pageHandled, globalHandled)
	}

	modal := &bindingPrimitive{Box: tview.NewBox()}
	handle := app.overlays.Push(modal)
	overlayEvent := tcell.NewEventKey(tcell.KeyRune, 'p', 0)
	if got := app.captureKey(overlayEvent); got != overlayEvent {
		t.Fatal("overlay event reached page binding dispatcher")
	}
	handle.Close()
}

func TestApplicationMovesFocusWithTabAndBacktab(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	first := tview.NewBox()
	second := tview.NewBox()
	third := tview.NewBox()
	page := applicationTestPage{
		id:         "logbook",
		title:      "Logbook",
		content:    first,
		focusables: []tview.Primitive{first, second, third},
	}
	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	app.showPage(page)

	if got := app.captureKey(tcell.NewEventKey(tcell.KeyTab, 0, 0)); got != nil {
		t.Fatal("Tab was forwarded")
	}
	if got := app.engine.GetFocus(); got != second {
		t.Fatalf("focus after Tab = %T, want second control", got)
	}

	if got := app.captureKey(tcell.NewEventKey(tcell.KeyBacktab, 0, 0)); got != nil {
		t.Fatal("Backtab was forwarded")
	}
	if got := app.engine.GetFocus(); got != first {
		t.Fatalf("focus after Backtab = %T, want first control", got)
	}

	app.captureKey(tcell.NewEventKey(tcell.KeyBacktab, 0, 0))
	if got := app.engine.GetFocus(); got != third {
		t.Fatalf("wrapped focus after Backtab = %T, want third control", got)
	}

	if footer := drawApplicationFooter(t, app); strings.Contains(
		footer,
		"Tab/Shift+Tab next/previous",
	) {
		t.Fatalf("footer = %q, contains hidden focus navigation hint", footer)
	}
}

func TestApplicationMovesFocusPastOpenSelectPopup(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	selectField := app.controls.SelectField(
		"Mode",
		[]string{"CW", "SSB"},
		0,
		6,
		24,
	)
	next := tview.NewBox()
	page := applicationTestPage{
		id:         "logbook",
		title:      "Logbook",
		content:    selectField,
		focusables: []tview.Primitive{selectField, next},
	}
	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	app.showPage(page)

	selectField.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)
	if !app.overlays.Active() {
		t.Fatal("select popup did not open")
	}
	if got := app.captureKey(tcell.NewEventKey(tcell.KeyTab, 0, 0)); got != nil {
		t.Fatal("Tab was forwarded from select popup")
	}

	if app.overlays.Active() {
		t.Fatal("select popup remained open after focus moved")
	}
	if got := app.engine.GetFocus(); got != next {
		t.Fatalf("focus after popup Tab = %T, want next control", got)
	}
}

func TestApplicationIsolatesModalInputAndRestoresFocus(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	pageContent := tview.NewBox()
	pageHandled := 0
	page := applicationTestPage{
		id:      "logbook",
		title:   "Logbook",
		content: pageContent,
		bindings: []keybinding.Binding{
			bindingForRune("page", 'p', &pageHandled),
		},
	}
	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	cancel, runDone := startTestApplication(t, app, page.ID())
	defer finishTestApplication(t, cancel, runDone)

	first := tview.NewBox()
	second := tview.NewBox()
	modalHandled := 0
	dialog := applicationTestModal{
		content:    tview.NewFlex().AddItem(first, 0, 1, true),
		focusables: []tview.Primitive{first, second},
		bindings: []keybinding.Binding{
			bindingForRune("modal", 'm', &modalHandled),
		},
	}
	app.OpenModalForCurrentLayer(dialog)
	waitForApplicationCondition(t, app, "modal open", app.overlays.Active)

	if got := app.engine.GetFocus(); got != first {
		t.Fatalf("modal focus = %T, want first control", got)
	}
	onApplication(app, func() *tcell.EventKey {
		return app.captureKey(tcell.NewEventKey(tcell.KeyBacktab, 0, 0))
	})
	if got := onApplication(app, app.engine.GetFocus); got != second {
		t.Fatalf("focus after modal Backtab = %T, want second control", got)
	}
	pageEvent := tcell.NewEventKey(tcell.KeyRune, 'p', 0)
	if got := onApplication(app, func() *tcell.EventKey {
		return app.captureKey(pageEvent)
	}); got != pageEvent {
		t.Fatal("unhandled modal event was not forwarded to modal content")
	}
	if pageHandled != 0 {
		t.Fatalf("page binding handled %d modal events, want 0", pageHandled)
	}
	if got := onApplication(app, func() *tcell.EventKey {
		return app.captureKey(tcell.NewEventKey(tcell.KeyRune, 'm', 0))
	}); got != nil {
		t.Fatal("handled modal binding was forwarded")
	}
	if modalHandled != 1 {
		t.Fatalf("modal binding handled %d events, want 1", modalHandled)
	}

	onApplication(app, func() *tcell.EventKey {
		return app.captureKey(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	})
	if got := onApplication(app, app.engine.GetFocus); got != first {
		t.Fatalf("focus after modal Tab = %T, want first control", got)
	}
	footer := onApplication(app, func() string {
		return drawApplicationFooter(t, app)
	})
	for _, expected := range []string{"m modal"} {
		if !strings.Contains(footer, expected) {
			t.Errorf("modal footer = %q, want %q", footer, expected)
		}
	}
	if strings.Contains(footer, "p page") {
		t.Errorf("modal footer = %q, unexpectedly contains page binding", footer)
	}
	for _, hidden := range []string{"Esc close", "Tab/Shift+Tab next/previous"} {
		if strings.Contains(footer, hidden) {
			t.Errorf("modal footer = %q, unexpectedly contains %q", footer, hidden)
		}
	}

	if got := onApplication(app, func() *tcell.EventKey {
		return app.captureKey(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	}); got != nil {
		t.Fatal("Escape was forwarded")
	}
	waitForApplicationCondition(t, app, "modal close", func() bool {
		return !app.overlays.Active()
	})
	if onApplication(app, app.overlays.Active) {
		t.Fatal("overlay remains active after closing modal")
	}
	if got := onApplication(app, app.engine.GetFocus); got != pageContent {
		t.Fatalf("restored focus = %T, want page content", got)
	}
}

func TestApplicationFocusesModalContentWithoutFocusableControls(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	page := applicationTestPage{
		id:      "logbook",
		title:   "Logbook",
		content: tview.NewBox(),
	}
	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	cancel, runDone := startTestApplication(t, app, page.ID())
	defer finishTestApplication(t, cancel, runDone)

	content := tview.NewBox()
	app.OpenModalForCurrentLayer(applicationTestModal{content: content})
	waitForApplicationCondition(t, app, "modal open", app.overlays.Active)
	if got := onApplication(app, app.engine.GetFocus); got != content {
		t.Fatalf("modal focus = %T, want modal content", got)
	}
}

func TestApplicationCancelsModalRunWhenModalCloses(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	cancel, runDone := startApplicationWithDefaultPage(t, app)
	defer finishTestApplication(t, cancel, runDone)
	started := make(chan struct{}, 1)
	stopped := make(chan struct{}, 1)
	handle := app.OpenModalForCurrentLayer(applicationTestModal{
		content: tview.NewBox(),
		run: func(ctx context.Context) {
			started <- struct{}{}
			<-ctx.Done()
			stopped <- struct{}{}
		},
	})
	waitForApplicationSignal(t, started, "modal Run start")
	select {
	case <-stopped:
		t.Fatal("modal Run stopped before the modal closed")
	default:
	}

	handle.Close()

	waitForApplicationSignal(t, stopped, "modal Run cancellation")
}

func TestModalVisibilityFollowsRunLifecycle(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	cancel, runDone := startApplicationWithDefaultPage(t, app)
	defer finishTestApplication(t, cancel, runDone)
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	app.OpenModalForCurrentLayer(applicationTestModal{
		content: tview.NewBox(),
		run: func(context.Context) {
			started <- struct{}{}
			<-release
		},
	})
	waitForApplicationSignal(t, started, "modal Run start")
	if !app.overlays.Active() {
		t.Fatal("modal was not visible while Run was active")
	}

	close(release)
	released = true
	waitForApplicationCondition(t, app, "modal hide after Run", func() bool {
		return !app.overlays.Active()
	})
}

func TestApplicationWaitsForModalRunBeforeHidingItAndKeepsPageRunning(
	t *testing.T,
) {
	app := newApplication(nordTheme).(*application)
	page := &lifecycleApplicationTestPage{
		applicationTestPage: applicationTestPage{
			id: "decoder", title: "Decoder", content: tview.NewBox(),
		},
		started: make(chan struct{}, 1),
		stopped: make(chan struct{}, 1),
	}
	if err := app.Register(page); err != nil {
		t.Fatal(err)
	}
	cancel, runDone := startTestApplication(t, app, page.ID())
	defer finishTestApplication(t, cancel, runDone)
	waitForApplicationSignal(t, page.started, "decoder page start")

	modalStopping := make(chan struct{}, 1)
	releaseModal := make(chan struct{})
	released := false
	defer func() {
		if !released {
			close(releaseModal)
		}
	}()
	handle := app.OpenModalForCurrentLayer(applicationTestModal{
		content: tview.NewBox(),
		run: func(ctx context.Context) {
			<-ctx.Done()
			modalStopping <- struct{}{}
			<-releaseModal
		},
	})
	waitForApplicationCondition(t, app, "modal open", app.overlays.Active)

	handle.Close()
	waitForApplicationSignal(t, modalStopping, "modal cancellation")
	if !app.overlays.Active() {
		t.Fatal("modal was hidden before Run returned")
	}
	if len(page.stopped) != 0 {
		t.Fatal("decoder page stopped while modal was open")
	}

	close(releaseModal)
	released = true
	waitForApplicationCondition(t, app, "modal hide after Run", func() bool {
		return !app.overlays.Active()
	})
	if len(page.stopped) != 0 {
		t.Fatal("decoder page stopped after modal closed")
	}
}

func TestApplicationCancelsDescendantModalRunsWhenParentCloses(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	cancel, runDone := startApplicationWithDefaultPage(t, app)
	defer finishTestApplication(t, cancel, runDone)
	parentStarted := make(chan struct{}, 1)
	parentStopped := make(chan struct{}, 1)
	childStarted := make(chan struct{}, 1)
	childStopped := make(chan struct{}, 1)
	parent := app.OpenModalForCurrentLayer(applicationTestModal{
		content: tview.NewBox(),
		run: func(ctx context.Context) {
			parentStarted <- struct{}{}
			<-ctx.Done()
			parentStopped <- struct{}{}
		},
	})
	app.OpenModalForCurrentLayer(applicationTestModal{
		content: tview.NewBox(),
		run: func(ctx context.Context) {
			childStarted <- struct{}{}
			<-ctx.Done()
			childStopped <- struct{}{}
		},
	})
	waitForApplicationSignal(t, parentStarted, "parent modal Run start")
	waitForApplicationSignal(t, childStarted, "child modal Run start")

	parent.Close()

	waitForApplicationSignal(t, parentStopped, "parent modal Run cancellation")
	waitForApplicationSignal(t, childStopped, "child modal Run cancellation")
	waitForApplicationCondition(t, app, "modal stack close", func() bool {
		return !app.overlays.Active()
	})
	if app.overlays.Active() {
		t.Fatal("descendant overlay remains after parent modal closed")
	}
}

func TestApplicationRejectsModalRequestedByNonTopLayer(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	logbook := applicationTestPage{
		id: "logbook", title: "Logbook", content: tview.NewBox(),
	}
	decoder := applicationTestPage{
		id: "decoder", title: "Decoder", content: tview.NewBox(),
	}
	if err := app.Register(logbook); err != nil {
		t.Fatal(err)
	}
	if err := app.Register(decoder); err != nil {
		t.Fatal(err)
	}
	cancel, runDone := startTestApplication(t, app, logbook.ID())
	defer finishTestApplication(t, cancel, runDone)
	if err := app.Show(decoder.ID()); err != nil {
		t.Fatal(err)
	}
	waitForApplicationCondition(t, app, "decoder page", func() bool {
		return app.activePage != nil && app.activePage.ID() == decoder.ID()
	})

	app.OpenModal(logbook.Content(), applicationTestModal{
		content: tview.NewBox(),
	})
	requestedCount := app.layers.requestedCount()

	if requestedCount != 1 {
		t.Fatalf("requested layer count = %d, want page only", requestedCount)
	}
	if app.overlays.Active() {
		t.Fatal("modal requested by hidden page became visible")
	}
}

func TestApplicationRunsBackgroundWorkForTopLayerAndAppliesUpdate(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	page := applicationTestPage{
		id: "logbook", title: "Logbook", content: tview.NewBox(),
	}
	if err := app.Register(page); err != nil {
		t.Fatal(err)
	}
	cancel, runDone := startTestApplication(t, app, page.ID())
	defer finishTestApplication(t, cancel, runDone)

	workStarted := make(chan struct{})
	releaseWork := make(chan struct{})
	updated := make(chan struct{})
	accepted := app.Background(page.Content(), func(ctx context.Context) func() {
		close(workStarted)
		select {
		case <-releaseWork:
		case <-ctx.Done():
			return nil
		}
		return func() { close(updated) }
	})
	if !accepted {
		t.Fatal("Background() rejected work for the top page")
	}
	waitForApplicationSignal(t, workStarted, "background work start")
	close(releaseWork)
	waitForApplicationSignal(t, updated, "background UI update")
}

func TestApplicationScopesBackgroundWorkToModalLifecycle(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	page := applicationTestPage{
		id: "logbook", title: "Logbook", content: tview.NewBox(),
	}
	if err := app.Register(page); err != nil {
		t.Fatal(err)
	}
	cancel, runDone := startTestApplication(t, app, page.ID())
	defer finishTestApplication(t, cancel, runDone)

	dialog := applicationTestModal{content: tview.NewBox()}
	handle := app.OpenModal(page.Content(), dialog)
	waitForApplicationCondition(t, app, "modal open", app.overlays.Active)

	if app.Background(page.Content(), func(context.Context) func() {
		t.Error("background work ran for a covered page")
		return nil
	}) {
		t.Fatal("Background() accepted work for a non-top page")
	}

	workStarted := make(chan struct{})
	workCancelled := make(chan struct{})
	releaseWork := make(chan struct{})
	updated := make(chan struct{}, 1)
	accepted := app.Background(dialog.Content(), func(ctx context.Context) func() {
		close(workStarted)
		<-ctx.Done()
		close(workCancelled)
		<-releaseWork
		return func() { updated <- struct{}{} }
	})
	if !accepted {
		t.Fatal("Background() rejected work for the top modal")
	}
	waitForApplicationSignal(t, workStarted, "modal background work start")

	handle.Close()
	waitForApplicationSignal(t, workCancelled, "modal background cancellation")
	if !app.overlays.Active() {
		t.Fatal("modal was hidden before its background work stopped")
	}
	close(releaseWork)
	waitForApplicationCondition(t, app, "modal close", func() bool {
		return !app.overlays.Active()
	})
	select {
	case <-updated:
		t.Fatal("cancelled modal background work updated the UI")
	default:
	}
}

func TestApplicationForwardsTypedRunesToFocusedModalInput(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	page := applicationTestPage{
		id:      "logbook",
		title:   "Logbook",
		content: tview.NewBox(),
	}
	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	cancel, runDone := startTestApplication(t, app, page.ID())
	defer finishTestApplication(t, cancel, runDone)

	input := app.controls.Modal().InputField("Callsign", "")
	app.OpenModalForCurrentLayer(applicationTestModal{
		content:    input,
		focusables: []tview.Primitive{input},
	})
	waitForApplicationCondition(t, app, "modal open", app.overlays.Active)
	event := tcell.NewEventKey(tcell.KeyRune, 'x', 0)
	forwarded := onApplication(app, func() *tcell.EventKey {
		return app.captureKey(event)
	})
	if forwarded == nil {
		t.Fatal("typed rune was consumed by application input capture")
	}
	onApplication(app, func() bool {
		app.overlays.Root().InputHandler()(forwarded, app.SetFocus)
		return true
	})

	if got := onApplication(app, input.Value); got != "x" {
		t.Fatalf("modal input value = %q, want x", got)
	}
	footer := onApplication(app, func() string {
		return drawApplicationFooter(t, app)
	})
	if strings.Contains(footer, "Esc cancel") {
		t.Fatalf("modal input footer = %q, contains hidden Esc hint", footer)
	}
	escape := tcell.NewEventKey(tcell.KeyEscape, 0, 0)
	forwarded = onApplication(app, func() *tcell.EventKey {
		return app.captureKey(escape)
	})
	if forwarded != nil {
		t.Fatal("modal input Escape was forwarded")
	}
	waitForApplicationCondition(t, app, "modal close", func() bool {
		return !app.overlays.Active()
	})
	if onApplication(app, app.overlays.Active) {
		t.Fatal("application modal Escape binding did not close modal")
	}
}

func TestApplicationModalEscapePrecedesDialogBindings(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	page := applicationTestPage{
		id:      "logbook",
		title:   "Logbook",
		content: tview.NewBox(),
	}
	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	cancel, runDone := startTestApplication(t, app, page.ID())
	defer finishTestApplication(t, cancel, runDone)

	handled := 0
	dialog := applicationTestModal{
		content: tview.NewBox(),
		bindings: []keybinding.Binding{keybinding.OnKey(
			tcell.KeyEscape,
			"custom",
			func() { handled++ },
		)},
	}
	app.OpenModalForCurrentLayer(dialog)
	waitForApplicationCondition(t, app, "modal open", app.overlays.Active)

	if got := onApplication(app, func() *tcell.EventKey {
		return app.captureKey(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	}); got != nil {
		t.Fatal("modal Escape binding was forwarded")
	}
	if handled != 0 {
		t.Fatalf("dialog Escape binding handled %d events, want 0", handled)
	}
	waitForApplicationCondition(t, app, "modal close", func() bool {
		return !app.overlays.Active()
	})
	if onApplication(app, app.overlays.Active) {
		t.Fatal("application Escape binding did not close modal")
	}
	footer := onApplication(app, func() string {
		return drawApplicationFooter(t, app)
	})
	if strings.Contains(footer, "Esc custom") {
		t.Fatalf("modal footer = %q, contains hidden Esc hint", footer)
	}
}

func TestApplicationLetsPopupAboveModalOwnInput(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	page := applicationTestPage{
		id:      "logbook",
		title:   "Logbook",
		content: tview.NewBox(),
	}
	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	cancel, runDone := startTestApplication(t, app, page.ID())
	defer finishTestApplication(t, cancel, runDone)

	modalHandled := 0
	dialog := applicationTestModal{
		content: tview.NewBox(),
		bindings: []keybinding.Binding{
			bindingForRune("modal", 'm', &modalHandled),
		},
	}
	app.OpenModalForCurrentLayer(dialog)
	waitForApplicationCondition(t, app, "modal open", app.overlays.Active)
	popup := &bindingPrimitive{Box: tview.NewBox()}
	popupHandle := onApplication(app, func() components.Overlay {
		return app.overlays.Push(popup)
	})

	event := tcell.NewEventKey(tcell.KeyRune, 'm', 0)
	if got := onApplication(app, func() *tcell.EventKey {
		return app.captureKey(event)
	}); got != event {
		t.Fatal("popup event reached modal dispatcher")
	}
	if modalHandled != 0 {
		t.Fatalf("modal handled %d popup events, want 0", modalHandled)
	}

	onApplication(app, func() bool {
		popupHandle.Close()
		return true
	})
	if got := onApplication(app, func() *tcell.EventKey {
		return app.captureKey(event)
	}); got != nil {
		t.Fatal("modal binding was forwarded after popup closed")
	}
	if modalHandled != 1 {
		t.Fatalf("modal handled %d events after popup closed, want 1", modalHandled)
	}
}

func TestApplicationComposesFocusPageAndGlobalHints(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	controlHandled := 0
	pageHandled := 0
	globalHandled := 0
	content := &bindingPrimitive{
		Box: tview.NewBox(),
		bindings: []keybinding.Binding{
			bindingForRune("control", 'x', &controlHandled),
		},
	}
	page := applicationTestPage{
		id:      "logbook",
		title:   "Logbook",
		status:  "24 QSOs",
		content: content,
		bindings: []keybinding.Binding{
			bindingForRune("page", 'p', &pageHandled),
		},
	}
	app.globalBindings = []keybinding.Binding{
		bindingForRune("global", 'q', &globalHandled),
	}
	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	app.showPage(page)

	line := drawApplicationFooter(t, app)
	for _, expected := range []string{"x control", "p page", "q global"} {
		if !strings.Contains(line, expected) {
			t.Errorf("footer = %q, want %q", line, expected)
		}
	}

	input := app.controls.InputField("Search", "")
	input.SetBindings(keybinding.OnKey(
		tcell.KeyEnter,
		"done",
		func() {},
	))
	app.SetFocus(input)
	line = drawApplicationFooter(t, app)
	if strings.Contains(line, "Enter done") {
		t.Errorf("input footer = %q, contains hidden Enter hint", line)
	}
	for _, hidden := range []string{"p page", "q global"} {
		if strings.Contains(line, hidden) {
			t.Errorf("input footer = %q, unexpectedly contains %q", line, hidden)
		}
	}

	modalHandled := 0
	modal := &bindingPrimitive{
		Box: tview.NewBox(),
		bindings: []keybinding.Binding{
			keybinding.OnKey(
				tcell.KeyEscape,
				"close modal",
				func() { modalHandled++ },
			),
		},
	}
	handle := app.overlays.Push(modal)
	line = drawApplicationFooter(t, app)
	if strings.Contains(line, "Esc close modal") {
		t.Errorf("modal footer = %q, contains hidden Esc hint", line)
	}
	for _, hidden := range []string{"Enter done", "p page", "q global"} {
		if strings.Contains(line, hidden) {
			t.Errorf("modal footer = %q, unexpectedly contains %q", line, hidden)
		}
	}
	handle.Close()
}

func TestApplicationDoesNotAdvertiseNativeControlKeys(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	table := app.controls.Table("QSOs")
	table.SetRect(0, 0, 40, 5)
	page := applicationTestPage{
		id:      "logbook",
		title:   "Logbook",
		content: table,
	}
	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	app.showPage(page)

	table.MouseHandler()(
		tview.MouseLeftDown,
		tcell.NewEventMouse(1, 1, tcell.Button1, 0),
		app.SetFocus,
	)

	if got := app.engine.GetFocus(); got != table {
		t.Fatalf("mouse focus = %T, want table wrapper %T", got, table)
	}
	if footer := drawApplicationFooter(t, app); strings.Contains(footer, "↑/k ↓/j move") {
		t.Fatalf("footer after mouse focus = %q, contains unowned table hint", footer)
	}
}

func TestPageLifecycleFollowsVisibility(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	logbook := &lifecycleApplicationTestPage{
		applicationTestPage: applicationTestPage{
			id:      "logbook",
			title:   "Logbook",
			content: tview.NewBox(),
		},
		started: make(chan struct{}, 1),
		stopped: make(chan struct{}, 1),
	}
	decoder := &lifecycleApplicationTestPage{
		applicationTestPage: applicationTestPage{
			id:      "decoder",
			title:   "Decoder",
			content: tview.NewBox(),
		},
		started: make(chan struct{}, 1),
		stopped: make(chan struct{}, 1),
	}
	if err := app.Register(logbook); err != nil {
		t.Fatal(err)
	}
	if err := app.Register(decoder); err != nil {
		t.Fatal(err)
	}
	cancel, runDone := startTestApplication(t, app, logbook.ID())
	defer finishTestApplication(t, cancel, runDone)
	waitForApplicationSignal(t, logbook.started, "logbook start")

	if len(decoder.started) != 0 {
		t.Fatal("hidden decoder page was activated")
	}
	if err := app.Show(decoder.ID()); err != nil {
		t.Fatal(err)
	}
	waitForApplicationSignal(t, decoder.started, "decoder start")
	waitForApplicationSignal(t, logbook.stopped, "logbook stop")
	if err := app.Show(logbook.ID()); err != nil {
		t.Fatal(err)
	}
	waitForApplicationSignal(t, decoder.stopped, "decoder stop")
	waitForApplicationSignal(t, logbook.started, "restarted logbook")
}

func TestApplicationWaitsForPageRunBeforeShowingNextPage(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	current := &blockingLifecyclePage{
		applicationTestPage: applicationTestPage{
			id:      "decoder",
			title:   "Decoder",
			content: tview.NewBox(),
		},
		waitStarted: make(chan struct{}, 1),
		release:     make(chan struct{}),
	}
	next := &lifecycleApplicationTestPage{
		applicationTestPage: applicationTestPage{
			id:      "logbook",
			title:   "Logbook",
			content: tview.NewBox(),
		},
		started: make(chan struct{}, 1),
		stopped: make(chan struct{}, 1),
	}
	if err := app.Register(current); err != nil {
		t.Fatal(err)
	}
	if err := app.Register(next); err != nil {
		t.Fatal(err)
	}
	cancel, runDone := startTestApplication(t, app, current.ID())
	defer finishTestApplication(t, cancel, runDone)

	if err := app.Show(next.ID()); err != nil {
		t.Fatal(err)
	}
	waitForApplicationSignal(t, current.waitStarted, "current page cancellation")
	if app.activePage.ID() != current.ID() {
		t.Fatalf("active page = %q before current Run() returned", app.activePage.ID())
	}

	close(current.release)
	waitForApplicationSignal(t, next.started, "next page start")
	if app.activePage.ID() != next.ID() {
		t.Fatalf("active page = %q, want %q", app.activePage.ID(), next.ID())
	}
}

func startApplicationWithDefaultPage(
	t *testing.T,
	app *application,
) (context.CancelFunc, <-chan error) {
	t.Helper()
	page := applicationTestPage{
		id:      "logbook",
		title:   "Logbook",
		content: tview.NewBox(),
	}
	if err := app.Register(page); err != nil {
		t.Fatal(err)
	}
	return startTestApplication(t, app, page.ID())
}

func startTestApplication(
	t *testing.T,
	app *application,
	initialPageID string,
) (context.CancelFunc, <-chan error) {
	t.Helper()
	drawn := make(chan struct{})
	app.engine.SetBeforeDrawFunc(func(tcell.Screen) bool {
		select {
		case <-drawn:
		default:
			close(drawn)
		}
		return false
	})
	app.engine.SetScreen(tcell.NewSimulationScreen("UTF-8"))
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx, initialPageID) }()
	waitForApplicationSignal(t, drawn, "application draw")
	waitForApplicationCondition(t, app, "initial page", func() bool {
		return app.activePage != nil && app.activePage.ID() == initialPageID
	})
	return cancel, done
}

func waitForApplicationCondition(
	t *testing.T,
	app *application,
	description string,
	condition func() bool,
) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		matched := false
		app.Update(func() { matched = condition() })
		if matched {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", description)
}

func onApplication[T any](app *application, operation func() T) T {
	var result T
	app.Update(func() { result = operation() })
	return result
}

func finishTestApplication(
	t *testing.T,
	cancel context.CancelFunc,
	done <-chan error,
) {
	t.Helper()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("application did not stop")
	}
}

func TestApplicationRequiresActivePageBeforeRun(t *testing.T) {
	app := newApplication(nordTheme)
	if err := app.Run(context.Background(), "missing"); err == nil {
		t.Fatal("application ran without an active page")
	}
}

func TestApplicationRunWaitsForContextShutdown(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	page := applicationTestPage{
		id:      "logbook",
		title:   "Logbook",
		content: tview.NewBox(),
	}
	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	started := make(chan struct{})
	app.engine.SetBeforeDrawFunc(func(tcell.Screen) bool {
		select {
		case <-started:
		default:
			close(started)
		}
		return false
	})
	app.engine.SetScreen(tcell.NewSimulationScreen("UTF-8"))

	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() {
		runDone <- app.Run(ctx, page.ID())
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("application did not start")
	}
	cancel()

	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("Run() = %v", err)
		}
	case <-time.After(time.Second):
		cancel()
		t.Fatal("Run() did not wait for context shutdown")
	}
}

func TestApplicationQuitWaitsForActivePageShutdown(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	page := &blockingLifecyclePage{
		applicationTestPage: applicationTestPage{
			id:      "decoder",
			title:   "Decoder",
			content: tview.NewBox(),
		},
		waitStarted: make(chan struct{}, 1),
		release:     make(chan struct{}),
	}
	if err := app.Register(page); err != nil {
		t.Fatal(err)
	}
	drawn := make(chan struct{})
	app.engine.SetBeforeDrawFunc(func(tcell.Screen) bool {
		select {
		case <-drawn:
		default:
			close(drawn)
		}
		return false
	})
	app.engine.SetScreen(tcell.NewSimulationScreen("UTF-8"))
	runDone := make(chan error, 1)
	go func() { runDone <- app.Run(t.Context(), page.ID()) }()
	waitForApplicationSignal(t, drawn, "application draw")

	if !app.exitBinding.Handle(tcell.NewEventKey(tcell.KeyRune, 'q', 0)) {
		t.Fatal("quit binding was not handled")
	}
	waitForApplicationSignal(t, page.waitStarted, "page shutdown wait")
	select {
	case <-runDone:
		t.Fatal("application returned before page shutdown completed")
	case <-time.After(20 * time.Millisecond):
	}

	close(page.release)
	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("Run() = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("application did not stop after page shutdown")
	}
}

type applicationTestPage struct {
	id         string
	title      string
	status     string
	content    tview.Primitive
	focusables []tview.Primitive
	bindings   []keybinding.Binding
	menuItems  []components.MenuItem
}

type lifecycleApplicationTestPage struct {
	applicationTestPage
	started chan struct{}
	stopped chan struct{}
}

type settingsAwareApplicationTestPage struct {
	applicationTestPage
	settingsChanges int
}

func (p *settingsAwareApplicationTestPage) SettingsChanged() {
	p.settingsChanges++
}

func (p *lifecycleApplicationTestPage) Run(ctx context.Context) {
	p.started <- struct{}{}
	<-ctx.Done()
	p.stopped <- struct{}{}
}

type blockingLifecyclePage struct {
	applicationTestPage
	waitStarted chan struct{}
	release     chan struct{}
}

func (p *blockingLifecyclePage) Run(ctx context.Context) {
	<-ctx.Done()
	select {
	case p.waitStarted <- struct{}{}:
	default:
	}
	<-p.release
}

func (p applicationTestPage) ID() string {
	return p.id
}

func (p applicationTestPage) Title() string {
	return p.title
}

func (p applicationTestPage) Status() string {
	return p.status
}

func (p applicationTestPage) Content() tview.Primitive {
	return p.content
}

func (p applicationTestPage) Focusables() []tview.Primitive {
	return p.focusables
}

func (p applicationTestPage) KeyBindings() []keybinding.Binding {
	return p.bindings
}

func (p applicationTestPage) MenuItems() []components.MenuItem {
	return p.menuItems
}

func (p applicationTestPage) SettingsChanged() {}

func (p applicationTestPage) Run(ctx context.Context) { <-ctx.Done() }

type bindingPrimitive struct {
	*tview.Box
	bindings []keybinding.Binding
}

type applicationTestModal struct {
	content    tview.Primitive
	focusables []tview.Primitive
	bindings   []keybinding.Binding
	run        func(context.Context)
}

func (m applicationTestModal) Content() tview.Primitive {
	return m.content
}

func (m applicationTestModal) Focusables() []tview.Primitive {
	return m.focusables
}

func (m applicationTestModal) KeyBindings() []keybinding.Binding {
	return m.bindings
}

func (m applicationTestModal) Size() modal.Size {
	return modal.Size{Width: 30, Height: 10}
}

func (m applicationTestModal) Run(ctx context.Context) {
	if m.run != nil {
		m.run(ctx)
		return
	}
	<-ctx.Done()
}

func (p *bindingPrimitive) KeyBindings() []keybinding.Binding {
	return p.bindings
}

func bindingForRune(
	description string,
	key rune,
	count *int,
) keybinding.Binding {
	return keybinding.OnRune(key, description, func() { *count++ })
}

func waitForApplicationSignal(
	t *testing.T,
	signal <-chan struct{},
	description string,
) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func drawApplicationFooter(t *testing.T, app *application) string {
	return drawApplicationLine(t, app, 5)
}

func drawApplicationLine(
	t *testing.T,
	app *application,
	y int,
) string {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("initialize screen: %v", err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(100, 6)
	app.layout.SetRect(0, 0, 100, 6)
	app.layout.Draw(screen)

	var line strings.Builder
	for x := 0; x < 100; x++ {
		character, _, _, _ := screen.GetContent(x, y)
		line.WriteRune(character)
	}
	return strings.TrimSpace(line.String())
}
