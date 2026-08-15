package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestApplicationRegistersAndShowsPage(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	content := &hintPrimitive{
		Box: tview.NewBox(),
		hints: []keybinding.Hint{
			{Keys: "x", Description: "control"},
		},
	}
	page := applicationTestPage{
		id:      "logbook",
		title:   "Logbook",
		status:  "24 QSOs",
		content: content,
	}

	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	if err := app.Show(page.ID()); err != nil {
		t.Fatalf("show page: %v", err)
	}

	if app.activePage.ID() != page.ID() {
		t.Fatalf("active page = %q, want %q", app.activePage.ID(), page.ID())
	}
	if got := app.engine.GetFocus(); got != content {
		t.Fatalf("focus = %T, want page content", got)
	}
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
	if len(app.globalBindings) != 1 {
		t.Fatalf("global binding count = %d, want 1", len(app.globalBindings))
	}
	binding := app.globalBindings[0]
	if binding.Hint != (keybinding.Hint{Keys: "q", Description: "quit"}) {
		t.Fatalf("quit hint = %#v", binding.Hint)
	}
	if binding.Handle(tcell.NewEventKey(tcell.KeyRune, 'x', 0)) {
		t.Fatal("quit binding handled unrelated key")
	}
	if !binding.Handle(tcell.NewEventKey(tcell.KeyRune, 'q', 0)) {
		t.Fatal("quit binding did not handle q")
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
			bindingForRune("p", "page", 'p', &pageHandled),
		},
	}
	app.globalBindings = []keybinding.Binding{
		{
			Hint: keybinding.Hint{Keys: "q", Description: "global"},
			Handler: func(event *tcell.EventKey) bool {
				if event.Key() != tcell.KeyRune || event.Rune() != 'q' {
					return false
				}
				globalHandled++
				return true
			},
		},
	}
	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	if err := app.Show(page.ID()); err != nil {
		t.Fatalf("show page: %v", err)
	}

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
		keybinding.Hint{Keys: "Enter", Description: "done"},
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
	if got := app.captureKey(tcell.NewEventKey(tcell.KeyCtrlC, 0, 0)); got != nil {
		t.Fatal("application Ctrl-C was forwarded")
	}
	if pageHandled != 1 || globalHandled != 1 {
		t.Fatalf("input handled counts = page %d, global %d, want 1 and 1", pageHandled, globalHandled)
	}

	modal := &hintPrimitive{Box: tview.NewBox()}
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
	if err := app.Show(page.ID()); err != nil {
		t.Fatalf("show page: %v", err)
	}

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

	if footer := drawApplicationFooter(t, app); !strings.Contains(
		footer,
		"Tab/Shift+Tab next/previous",
	) {
		t.Fatalf("footer = %q, want focus navigation hint", footer)
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
			bindingForRune("p", "page", 'p', &pageHandled),
		},
	}
	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	if err := app.Show(page.ID()); err != nil {
		t.Fatalf("show page: %v", err)
	}

	first := tview.NewBox()
	second := tview.NewBox()
	modalHandled := 0
	dialog := applicationTestModal{
		content:    tview.NewFlex().AddItem(first, 0, 1, true),
		focusables: []tview.Primitive{first, second},
		bindings: []keybinding.Binding{
			bindingForRune("m", "modal", 'm', &modalHandled),
		},
	}
	handle := app.OpenModal(dialog)

	if got := app.engine.GetFocus(); got != first {
		t.Fatalf("modal focus = %T, want first control", got)
	}
	handle.FocusNext()
	if got := app.engine.GetFocus(); got != second {
		t.Fatalf("focus after handle FocusNext = %T, want second control", got)
	}
	app.captureKey(tcell.NewEventKey(tcell.KeyBacktab, 0, 0))
	if got := app.engine.GetFocus(); got != first {
		t.Fatalf("focus after modal Backtab = %T, want first control", got)
	}
	pageEvent := tcell.NewEventKey(tcell.KeyRune, 'p', 0)
	if got := app.captureKey(pageEvent); got != pageEvent {
		t.Fatal("unhandled modal event was not forwarded to modal content")
	}
	if pageHandled != 0 {
		t.Fatalf("page binding handled %d modal events, want 0", pageHandled)
	}
	if got := app.captureKey(tcell.NewEventKey(tcell.KeyRune, 'm', 0)); got != nil {
		t.Fatal("handled modal binding was forwarded")
	}
	if modalHandled != 1 {
		t.Fatalf("modal binding handled %d events, want 1", modalHandled)
	}

	app.captureKey(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	if got := app.engine.GetFocus(); got != second {
		t.Fatalf("focus after modal Tab = %T, want second control", got)
	}
	footer := drawApplicationFooter(t, app)
	for _, expected := range []string{"m modal", "Esc close", "Tab/Shift+Tab next/previous"} {
		if !strings.Contains(footer, expected) {
			t.Errorf("modal footer = %q, want %q", footer, expected)
		}
	}
	if strings.Contains(footer, "p page") {
		t.Errorf("modal footer = %q, unexpectedly contains page binding", footer)
	}

	if got := app.captureKey(tcell.NewEventKey(tcell.KeyEscape, 0, 0)); got != nil {
		t.Fatal("Escape was forwarded")
	}
	if app.overlays.Active() {
		t.Fatal("overlay remains active after closing modal")
	}
	if got := app.engine.GetFocus(); got != pageContent {
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
	if err := app.Show(page.ID()); err != nil {
		t.Fatalf("show page: %v", err)
	}

	content := tview.NewBox()
	app.OpenModal(applicationTestModal{content: content})
	if got := app.engine.GetFocus(); got != content {
		t.Fatalf("modal focus = %T, want modal content", got)
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
	if err := app.Show(page.ID()); err != nil {
		t.Fatalf("show page: %v", err)
	}

	input := app.controls.Modal().InputField("Callsign", "")
	var handle modal.Handle
	input.SetBindings(keybinding.OnKey(
		tcell.KeyEscape,
		keybinding.Hint{Keys: "Esc", Description: "cancel"},
		func() { handle.Close() },
	))
	handle = app.OpenModal(applicationTestModal{
		content:    input,
		focusables: []tview.Primitive{input},
	})
	event := tcell.NewEventKey(tcell.KeyRune, 'x', 0)
	forwarded := app.captureKey(event)
	if forwarded == nil {
		t.Fatal("typed rune was consumed by application input capture")
	}
	app.overlays.Root().InputHandler()(forwarded, app.SetFocus)

	if got := input.Value(); got != "x" {
		t.Fatalf("modal input value = %q, want x", got)
	}
	footer := drawApplicationFooter(t, app)
	if strings.Count(footer, "Esc") != 1 || !strings.Contains(footer, "Esc cancel") {
		t.Fatalf("modal input footer = %q, want one Esc cancel hint", footer)
	}
	escape := tcell.NewEventKey(tcell.KeyEscape, 0, 0)
	forwarded = app.captureKey(escape)
	if forwarded != escape {
		t.Fatal("modal input Escape was consumed by application")
	}
	app.overlays.Root().InputHandler()(forwarded, app.SetFocus)
	if app.overlays.Active() {
		t.Fatal("modal input Escape callback did not close modal")
	}
}

func TestApplicationLetsModalBindingOverrideDefaultEscape(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	page := applicationTestPage{
		id:      "logbook",
		title:   "Logbook",
		content: tview.NewBox(),
	}
	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	if err := app.Show(page.ID()); err != nil {
		t.Fatalf("show page: %v", err)
	}

	handled := 0
	dialog := applicationTestModal{
		content: tview.NewBox(),
		bindings: []keybinding.Binding{{
			Hint: keybinding.Hint{Keys: "Esc", Description: "custom"},
			Handler: func(event *tcell.EventKey) bool {
				if event.Key() != tcell.KeyEscape {
					return false
				}
				handled++
				return true
			},
		}},
	}
	app.OpenModal(dialog)

	if got := app.captureKey(tcell.NewEventKey(tcell.KeyEscape, 0, 0)); got != nil {
		t.Fatal("modal Escape binding was forwarded")
	}
	if handled != 1 {
		t.Fatalf("modal Escape binding handled %d events, want 1", handled)
	}
	if !app.overlays.Active() {
		t.Fatal("default Escape closed modal before its own binding")
	}
	footer := drawApplicationFooter(t, app)
	if strings.Count(footer, "Esc") != 1 || !strings.Contains(footer, "Esc custom") {
		t.Fatalf("modal footer = %q, want one Esc custom hint", footer)
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
	if err := app.Show(page.ID()); err != nil {
		t.Fatalf("show page: %v", err)
	}

	modalHandled := 0
	dialog := applicationTestModal{
		content: tview.NewBox(),
		bindings: []keybinding.Binding{
			bindingForRune("m", "modal", 'm', &modalHandled),
		},
	}
	app.OpenModal(dialog)
	popup := &hintPrimitive{Box: tview.NewBox()}
	popupHandle := app.overlays.Push(popup)

	event := tcell.NewEventKey(tcell.KeyRune, 'm', 0)
	if got := app.captureKey(event); got != event {
		t.Fatal("popup event reached modal dispatcher")
	}
	if modalHandled != 0 {
		t.Fatalf("modal handled %d popup events, want 0", modalHandled)
	}

	popupHandle.Close()
	if got := app.captureKey(event); got != nil {
		t.Fatal("modal binding was forwarded after popup closed")
	}
	if modalHandled != 1 {
		t.Fatalf("modal handled %d events after popup closed, want 1", modalHandled)
	}
}

func TestApplicationComposesFocusPageAndGlobalHints(t *testing.T) {
	app := newApplication(nordTheme).(*application)
	content := &hintPrimitive{
		Box: tview.NewBox(),
		hints: []keybinding.Hint{
			{Keys: "x", Description: "control"},
		},
	}
	page := applicationTestPage{
		id:      "logbook",
		title:   "Logbook",
		status:  "24 QSOs",
		content: content,
		bindings: []keybinding.Binding{
			{Hint: keybinding.Hint{Keys: "p", Description: "page"}},
		},
	}
	app.globalBindings = []keybinding.Binding{
		{Hint: keybinding.Hint{Keys: "q", Description: "global"}},
	}
	if err := app.Register(page); err != nil {
		t.Fatalf("register page: %v", err)
	}
	if err := app.Show(page.ID()); err != nil {
		t.Fatalf("show page: %v", err)
	}

	line := drawApplicationFooter(t, app)
	for _, expected := range []string{"x control", "p page", "q global"} {
		if !strings.Contains(line, expected) {
			t.Errorf("footer = %q, want %q", line, expected)
		}
	}

	input := app.controls.InputField("Search", "")
	input.SetBindings(keybinding.OnKey(
		tcell.KeyEnter,
		keybinding.Hint{Keys: "Enter", Description: "done"},
		func() {},
	))
	app.SetFocus(input)
	line = drawApplicationFooter(t, app)
	if !strings.Contains(line, "Enter done") {
		t.Errorf("input footer = %q, want input hint", line)
	}
	for _, hidden := range []string{"p page", "q global"} {
		if strings.Contains(line, hidden) {
			t.Errorf("input footer = %q, unexpectedly contains %q", line, hidden)
		}
	}

	modal := &hintPrimitive{
		Box: tview.NewBox(),
		hints: []keybinding.Hint{
			{Keys: "Esc", Description: "close modal"},
		},
	}
	handle := app.overlays.Push(modal)
	line = drawApplicationFooter(t, app)
	if !strings.Contains(line, "Esc close modal") {
		t.Errorf("modal footer = %q, want modal hint", line)
	}
	for _, hidden := range []string{"Enter done", "p page", "q global"} {
		if strings.Contains(line, hidden) {
			t.Errorf("modal footer = %q, unexpectedly contains %q", line, hidden)
		}
	}
	handle.Close()
}

func TestApplicationKeepsControlHintsAfterMouseFocus(t *testing.T) {
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
	if err := app.Show(page.ID()); err != nil {
		t.Fatalf("show page: %v", err)
	}

	table.MouseHandler()(
		tview.MouseLeftDown,
		tcell.NewEventMouse(1, 1, tcell.Button1, 0),
		app.SetFocus,
	)

	if got := app.engine.GetFocus(); got != table {
		t.Fatalf("mouse focus = %T, want table wrapper %T", got, table)
	}
	if footer := drawApplicationFooter(t, app); !strings.Contains(footer, "↑/k ↓/j move") {
		t.Fatalf("footer after mouse focus = %q, want table hints", footer)
	}
}

func TestApplicationRequiresActivePageBeforeRun(t *testing.T) {
	app := newApplication(nordTheme)
	if err := app.Run(context.Background()); err == nil {
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
	if err := app.Show(page.ID()); err != nil {
		t.Fatalf("show page: %v", err)
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
		runDone <- app.Run(ctx)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		app.Stop()
		t.Fatal("application did not start")
	}
	cancel()

	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("Run() = %v", err)
		}
	case <-time.After(time.Second):
		app.Stop()
		t.Fatal("Run() did not wait for context shutdown")
	}
}

type applicationTestPage struct {
	id         string
	title      string
	status     string
	content    tview.Primitive
	focusables []tview.Primitive
	bindings   []keybinding.Binding
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

type hintPrimitive struct {
	*tview.Box
	hints []keybinding.Hint
}

type applicationTestModal struct {
	content    tview.Primitive
	focusables []tview.Primitive
	bindings   []keybinding.Binding
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

func (p *hintPrimitive) KeyHints() []keybinding.Hint {
	return p.hints
}

func bindingForRune(
	keys string,
	description string,
	key rune,
	count *int,
) keybinding.Binding {
	return keybinding.Binding{
		Hint: keybinding.Hint{Keys: keys, Description: description},
		Handler: func(event *tcell.EventKey) bool {
			if event.Key() != tcell.KeyRune || event.Rune() != key {
				return false
			}
			*count++
			return true
		},
	}
}

func drawApplicationFooter(t *testing.T, app *application) string {
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
		character, _, _, _ := screen.GetContent(x, 5)
		line.WriteRune(character)
	}
	return strings.TrimSpace(line.String())
}
