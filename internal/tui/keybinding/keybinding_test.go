package keybinding

import (
	"reflect"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestBindingHandlesEvent(t *testing.T) {
	binding := Binding{
		Hint: Hint{Keys: "q", Description: "quit"},
		Handler: func(event *tcell.EventKey) bool {
			return event.Key() == tcell.KeyRune && event.Rune() == 'q'
		},
	}

	if !binding.Handle(tcell.NewEventKey(tcell.KeyRune, 'q', 0)) {
		t.Fatal("q was not handled")
	}
	if binding.Handle(tcell.NewEventKey(tcell.KeyRune, 'x', 0)) {
		t.Fatal("x was handled")
	}
}

func TestBindingWithoutHandlerDoesNotHandleEvent(t *testing.T) {
	if (Binding{}).Handle(tcell.NewEventKey(tcell.KeyEnter, 0, 0)) {
		t.Fatal("binding without a handler handled an event")
	}
}

func TestHintsPreservesBindingOrder(t *testing.T) {
	bindings := []Binding{
		{Hint: Hint{Keys: "/", Description: "search"}},
		{Hint: Hint{Keys: "q", Description: "quit"}},
	}
	want := []Hint{
		{Keys: "/", Description: "search"},
		{Keys: "q", Description: "quit"},
	}

	if got := Hints(bindings); !reflect.DeepEqual(got, want) {
		t.Fatalf("Hints() = %#v, want %#v", got, want)
	}
}
