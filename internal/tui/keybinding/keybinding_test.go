package keybinding

import (
	"reflect"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestOnRuneHandlesOnlyConfiguredRune(t *testing.T) {
	handled := 0
	binding := OnRune('q', "quit", func() { handled++ })

	if !binding.Handle(tcell.NewEventKey(tcell.KeyRune, 'q', 0)) {
		t.Fatal("q was not handled")
	}
	if binding.Handle(tcell.NewEventKey(tcell.KeyRune, 'x', 0)) {
		t.Fatal("x was handled")
	}
	if handled != 1 {
		t.Fatalf("handler calls = %d, want 1", handled)
	}
	if got := binding.Hint(); got != (Hint{Keys: "q", Description: "quit"}) {
		t.Fatalf("Hint() = %#v", got)
	}
}

func TestOnKeyHandlesOnlyConfiguredKey(t *testing.T) {
	handled := 0
	binding := OnKey(tcell.KeyEscape, "close", func() { handled++ })

	if binding.Handle(tcell.NewEventKey(tcell.KeyEnter, 0, 0)) {
		t.Fatal("Enter was handled")
	}
	if !binding.Handle(tcell.NewEventKey(tcell.KeyEscape, 0, 0)) {
		t.Fatal("Escape was not handled")
	}
	if handled != 1 {
		t.Fatalf("handler calls = %d, want 1", handled)
	}
	if got := binding.Hint(); got != (Hint{Keys: "Esc", Description: "close"}) {
		t.Fatalf("Hint() = %#v", got)
	}
}

func TestOnDerivesHintFromEveryAcceptedStroke(t *testing.T) {
	binding := On(
		"open",
		func() {},
		Key(tcell.KeyEnter),
		Rune(' '),
	)

	if got := binding.Hint(); got != (Hint{Keys: "Enter/Space", Description: "open"}) {
		t.Fatalf("Hint() = %#v", got)
	}
}

func TestBindingCanBeInvokedWithoutKeyboardEvent(t *testing.T) {
	handled := 0
	binding := OnRune('q', "quit", func() { handled++ })

	if !binding.Invoke() {
		t.Fatal("Invoke() = false, want true")
	}
	if handled != 1 {
		t.Fatalf("handler calls = %d, want 1", handled)
	}
}

func TestActionHasNoKeyboardShortcut(t *testing.T) {
	handled := 0
	action := Action("about", func() { handled++ })

	if action.Handle(tcell.NewEventKey(tcell.KeyRune, 'a', 0)) {
		t.Fatal("shortcut-free action handled a key")
	}
	if !action.Invoke() || handled != 1 {
		t.Fatalf("action invocation returned false or called handler %d times", handled)
	}
	if got := Hints([]Binding{action}); len(got) != 0 {
		t.Fatalf("shortcut-free action hints = %#v, want none", got)
	}
}

func TestHintsKeepsApplicationKeysAndHidesConventionalKeys(t *testing.T) {
	bindings := []Binding{
		OnRune('/', "search", func() {}),
		OnKey(tcell.KeyEscape, "close", func() {}),
		OnKey(tcell.KeyEnter, "open", func() {}),
		OnKey(tcell.KeyTab, "next", func() {}),
		OnRune('q', "quit", func() {}),
		OnRune('x', "missing handler", nil),
	}
	want := []Hint{
		{Keys: "/", Description: "search"},
		{Keys: "q", Description: "quit"},
	}

	if got := Hints(bindings); !reflect.DeepEqual(got, want) {
		t.Fatalf("Hints() = %#v, want %#v", got, want)
	}
}
