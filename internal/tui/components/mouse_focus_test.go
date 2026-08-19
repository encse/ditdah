package components

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestWrappedControlsKeepTheirIdentityWhenFocusedByMouse(t *testing.T) {
	controls := newTestFactory()
	tests := []struct {
		name    string
		control tview.Primitive
		action  tview.MouseAction
		x       int
		y       int
	}{
		{name: "input field", control: controls.InputField("", ""), x: 1, y: 1},
		{name: "text area", control: controls.TextArea("", ""), x: 1, y: 1},
		{name: "button", control: controls.Button("OK"), x: 1, y: 0},
		{
			name: "menu",
			control: controls.Menu(
				"File", []MenuItem{{Label: "Exit"}},
			),
			x:      1,
			y:      0,
			action: tview.MouseLeftClick,
		},
		{name: "table", control: controls.Table(""), x: 1, y: 1},
		{name: "text view", control: controls.TextView(), x: 1, y: 1},
		{
			name: "select field",
			control: controls.SelectField(
				"Mode", []string{"CW", "SSB"}, 0, 5, 15,
			),
			x: 6,
			y: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.control.SetRect(0, 0, 20, 3)
			var focused tview.Primitive
			handler := test.control.MouseHandler()
			if handler == nil {
				t.Fatal("MouseHandler() = nil")
			}
			action := test.action
			if action == 0 {
				action = tview.MouseLeftDown
			}
			handler(
				action,
				tcell.NewEventMouse(test.x, test.y, tcell.Button1, 0),
				func(primitive tview.Primitive) { focused = primitive },
			)
			if focused != test.control {
				t.Fatalf("focused control = %T, want wrapper %T", focused, test.control)
			}
		})
	}
}
