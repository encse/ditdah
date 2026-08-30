package components

import (
	"fmt"
	"strings"
	"unicode"

	"ditdah/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

// SelectField is a selection control with a full-width bordered popup.
type SelectField interface {
	tview.FormItem
	keybinding.BindingProvider
	CurrentOption() (index int, value string)
	SetOptions(options []string, selected int)
	SetChangedFunc(handler func(index int, value string))
}

type selectField struct {
	*tview.Box

	label               string
	options             []string
	selected            int
	highlighted         int
	open                bool
	disabled            bool
	labelWidth          int
	labelColor          tcell.Color
	fieldText           tcell.Color
	fieldBackground     tcell.Color
	selectionText       tcell.Color
	selectionBackground tcell.Color
	borderColor         tcell.Color
	focusBackground     tcell.Color
	formLabelWidth      int
	formFieldWidth      int
	overlays            OverlayHost
	overlay             Overlay
	changed             func(index int, value string)
}

func newSelectField(
	label string,
	options []string,
	selected int,
	labelWidth int,
	fieldWidth int,
	theme Theme,
	overlays OverlayHost,
	focusChanged func(),
) SelectField {
	field := &selectField{
		Box:                 tview.NewBox().SetBackgroundColor(theme.Background),
		label:               label,
		options:             options,
		selected:            selected,
		highlighted:         selected,
		labelColor:          theme.LabelColor,
		fieldText:           theme.FieldTextColor,
		fieldBackground:     theme.FieldBackground,
		selectionText:       theme.SelectionText,
		selectionBackground: theme.SelectionBackground,
		borderColor:         theme.PopupBorder,
		focusBackground:     theme.ActiveFieldBackground,
		labelWidth:          labelWidth,
		formLabelWidth:      labelWidth,
		formFieldWidth:      fieldWidth,
		overlays:            overlays,
	}
	field.SetFocusFunc(func() {
		notify(focusChanged)
	})
	return field
}

func (s *selectField) GetLabel() string {
	return fmt.Sprintf("%-*s", s.formLabelWidth, s.label)
}

func (s *selectField) GetFieldWidth() int {
	return s.formFieldWidth
}

func (s *selectField) GetFieldHeight() int { return 1 }

func (s *selectField) SetFormAttributes(
	labelWidth int,
	labelColor tcell.Color,
	backgroundColor tcell.Color,
	fieldTextColor tcell.Color,
	fieldBackgroundColor tcell.Color,
) tview.FormItem {
	s.labelWidth = labelWidth
	s.labelColor = labelColor
	s.fieldText = fieldTextColor
	s.fieldBackground = fieldBackgroundColor
	s.SetBackgroundColor(backgroundColor)
	return s
}

func (s *selectField) SetFinishedFunc(func(tcell.Key)) tview.FormItem {
	// The application owns form focus navigation. This method exists only to
	// satisfy tview.FormItem; the select never translates keys into navigation.
	return s
}

func (s *selectField) SetDisabled(disabled bool) tview.FormItem {
	s.disabled = disabled
	if disabled {
		s.closePopup()
	}
	return s
}

func (s *selectField) CurrentOption() (int, string) {
	if s.selected < 0 || s.selected >= len(s.options) {
		return -1, ""
	}
	return s.selected, s.options[s.selected]
}

func (s *selectField) SetOptions(options []string, selected int) {
	s.closePopup()
	s.options = append(s.options[:0], options...)
	s.selected = selected
	s.highlighted = selected
}

func (s *selectField) SetChangedFunc(handler func(index int, value string)) {
	s.changed = handler
}

func (s *selectField) KeyBindings() []keybinding.Binding {
	return []keybinding.Binding{keybinding.On(
		"open",
		func() {
			s.openPopup()
		},
		keybinding.Key(tcell.KeyEnter),
		keybinding.Rune(' '),
	)}
}

func (s *selectField) Draw(screen tcell.Screen) {
	s.Box.DrawForSubclass(screen, s)
	x, y, width, height := s.GetInnerRect()
	if width <= s.labelWidth || height < 1 {
		return
	}

	tview.Print(screen, s.label, x, y, s.labelWidth, tview.AlignLeft, s.labelColor)
	fieldX := x + s.labelWidth
	fieldWidth := width - s.labelWidth
	fieldBackground := s.fieldBackground
	if s.HasFocus() || s.open {
		fieldBackground = s.focusBackground
	}
	fieldStyle := tcell.StyleDefault.
		Foreground(s.fieldText).
		Background(fieldBackground)
	if s.HasFocus() && !s.open {
		fieldStyle = fieldStyle.Bold(true)
	}
	fill(screen, fieldX, y, fieldWidth, fieldStyle)

	_, value := s.CurrentOption()
	drawText(screen, value, fieldX+1, y, fieldWidth-3, fieldStyle)
	arrow := '▼'
	if s.open {
		arrow = '▲'
	}
	screen.SetContent(fieldX+fieldWidth-2, y, arrow, nil, fieldStyle)

}

func (s *selectField) InputHandler() func(
	*tcell.EventKey,
	func(tview.Primitive),
) {
	return s.WrapInputHandler(func(event *tcell.EventKey, _ func(tview.Primitive)) {
		if s.disabled {
			return
		}
		for _, binding := range s.KeyBindings() {
			if binding.Handle(event) {
				return
			}
		}

		switch event.Key() {
		case tcell.KeyDown:
			if !s.open {
				s.openPopup()
			} else if s.highlighted < len(s.options)-1 {
				s.highlighted++
			}
		case tcell.KeyUp:
			if !s.open {
				s.openPopup()
			} else if s.highlighted > 0 {
				s.highlighted--
			}
		case tcell.KeyHome:
			if s.open && len(s.options) > 0 {
				s.highlighted = 0
			}
		case tcell.KeyEnd:
			if s.open && len(s.options) > 0 {
				s.highlighted = len(s.options) - 1
			}
		}
	})
}

func (s *selectField) MouseHandler() func(
	tview.MouseAction,
	*tcell.EventMouse,
	func(tview.Primitive),
) (bool, tview.Primitive) {
	return s.WrapMouseHandler(func(
		action tview.MouseAction,
		event *tcell.EventMouse,
		setFocus func(tview.Primitive),
	) (bool, tview.Primitive) {
		if s.disabled {
			return false, nil
		}

		x, y := event.Position()
		fieldX, fieldY, width, _ := s.GetInnerRect()
		fieldX += s.labelWidth
		width -= s.labelWidth
		if action == tview.MouseLeftDown &&
			y == fieldY && x >= fieldX && x < fieldX+width {
			setFocus(s)
			if s.open {
				s.closePopup()
			} else {
				s.openPopup()
			}
			return true, s
		}
		return false, nil
	})
}

func (s *selectField) openPopup() {
	if len(s.options) == 0 || s.overlays == nil || s.open {
		return
	}
	s.open = true
	s.highlighted = s.selected
	if s.highlighted < 0 {
		s.highlighted = 0
	}
	s.overlay = s.overlays.Push(newSelectPopup(s))
}

func (s *selectField) closePopup() {
	s.open = false
	if s.overlay == nil {
		return
	}
	overlay := s.overlay
	s.overlay = nil
	overlay.Close()
}

func (s *selectField) removePopup() {
	s.open = false
	if s.overlay == nil {
		return
	}
	overlay := s.overlay
	s.overlay = nil
	overlay.Remove()
}

func (s *selectField) selectHighlighted() {
	previous := s.selected
	if s.highlighted >= 0 && s.highlighted < len(s.options) {
		s.selected = s.highlighted
	}
	s.closePopup()
	if s.selected != previous && s.changed != nil {
		_, value := s.CurrentOption()
		s.changed(s.selected, value)
	}
}

type selectPopup struct {
	*tview.Box
	selectField *selectField
	list        *tview.List
	x           int
	y           int
	width       int
	height      int
}

func (p *selectPopup) KeyBindings() []keybinding.Binding {
	return []keybinding.Binding{keybinding.OnKey(
		tcell.KeyEscape,
		"close",
		p.selectField.closePopup,
	)}
}

func newSelectPopup(selectField *selectField) tview.Primitive {
	list := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetWrapAround(false).
		SetMainTextStyle(
			tcell.StyleDefault.
				Foreground(selectField.fieldText).
				Background(selectField.fieldBackground),
		).
		SetSelectedStyle(
			tcell.StyleDefault.
				Foreground(selectField.selectionText).
				Background(selectField.selectionBackground).
				Bold(true),
		)
	list.SetBackgroundColor(selectField.fieldBackground)
	list.SetBorder(true).
		SetBorderColor(selectField.borderColor)
	for _, option := range selectField.options {
		list.AddItem(option, "", 0, nil)
	}
	list.SetCurrentItem(selectField.highlighted)

	popup := &selectPopup{
		Box:         tview.NewBox(),
		selectField: selectField,
		list:        list,
	}
	list.SetChangedFunc(func(index int, _ string, _ string, _ rune) {
		selectField.highlighted = index
	})
	list.SetSelectedFunc(func(index int, _ string, _ string, _ rune) {
		selectField.highlighted = index
		selectField.selectHighlighted()
	})
	list.SetDoneFunc(selectField.closePopup)
	return popup
}

func (p *selectPopup) Draw(screen tcell.Screen) {
	p.position(screen)
	if p.height < 3 || p.width < 3 {
		return
	}
	p.list.SetRect(p.x, p.y, p.width, p.height)
	p.list.Draw(screen)
}

func (p *selectPopup) position(screen tcell.Screen) {
	_, screenHeight := screen.Size()
	fieldX, fieldY, fieldWidth, _ := p.selectField.GetInnerRect()
	fieldX += p.selectField.labelWidth
	fieldWidth -= p.selectField.labelWidth

	wantedHeight := len(p.selectField.options) + 2
	spaceBelow := screenHeight - fieldY - 1
	spaceAbove := fieldY
	p.height = wantedHeight
	switch {
	case wantedHeight <= spaceBelow:
		p.y = fieldY + 1
	case wantedHeight <= spaceAbove:
		p.y = fieldY - wantedHeight
	case spaceBelow >= spaceAbove:
		p.y = fieldY + 1
		p.height = spaceBelow
	default:
		p.height = spaceAbove
		p.y = fieldY - p.height
	}
	p.x = fieldX
	p.width = fieldWidth
}

func (p *selectPopup) Focus(_ func(tview.Primitive)) {
	p.Box.Focus(nil)
}

func (p *selectPopup) Blur() {
	p.Box.Blur()
	p.list.Blur()
	p.selectField.removePopup()
}

func (p *selectPopup) HasFocus() bool {
	return p.Box.HasFocus() || p.list.HasFocus()
}

func (p *selectPopup) InputHandler() func(
	*tcell.EventKey,
	func(tview.Primitive),
) {
	return p.WrapInputHandler(func(
		event *tcell.EventKey,
		setFocus func(tview.Primitive),
	) {
		for _, binding := range p.KeyBindings() {
			if binding.Handle(event) {
				return
			}
		}
		if event.Key() == tcell.KeyRune && unicode.IsLetter(event.Rune()) {
			if index := firstOptionStartingWith(
				p.selectField.options,
				event.Rune(),
			); index >= 0 {
				p.list.SetCurrentItem(index)
				return
			}
		}
		if handler := p.list.InputHandler(); handler != nil {
			handler(event, setFocus)
		}
	})
}

func firstOptionStartingWith(options []string, character rune) int {
	wanted := unicode.ToLower(character)
	for index, option := range options {
		for _, first := range strings.TrimSpace(option) {
			if unicode.ToLower(first) == wanted {
				return index
			}
			break
		}
	}
	return -1
}

func (p *selectPopup) MouseHandler() func(
	tview.MouseAction,
	*tcell.EventMouse,
	func(tview.Primitive),
) (bool, tview.Primitive) {
	// The tview.List handles interaction inside the popup. This full-screen
	// overlay handler only routes those events to the list, closes the popup on
	// an outside click, and consumes the click so it cannot reach the UI below.
	// Mouse capture is kept only until button release; retaining the removed
	// popup as the capture target would steal all subsequent mouse events.
	return p.WrapMouseHandler(func(
		action tview.MouseAction,
		event *tcell.EventMouse,
		_ func(tview.Primitive),
	) (bool, tview.Primitive) {
		x, y := event.Position()
		inside := x >= p.x && x < p.x+p.width &&
			y >= p.y && y < p.y+p.height
		var capture tview.Primitive
		if action == tview.MouseLeftDown {
			capture = p
		}
		if inside {
			if action == tview.MouseScrollUp || action == tview.MouseScrollDown {
				// List's native wheel handling moves only its viewport offset. Its
				// next Draw may move that offset back to keep the current item in
				// view, so a dropdown wheel moves the current item instead.
				key := tcell.KeyUp
				if action == tview.MouseScrollDown {
					key = tcell.KeyDown
				}
				p.list.InputHandler()(
					tcell.NewEventKey(key, 0, 0),
					func(tview.Primitive) {},
				)
				return true, nil
			}
			if handler := p.list.MouseHandler(); handler != nil {
				consumed, _ := handler(
					action,
					event,
					func(tview.Primitive) {},
				)
				if consumed {
					return true, capture
				}
			}
			return true, capture
		}

		if action == tview.MouseLeftDown || action == tview.MouseLeftClick {
			p.selectField.closePopup()
		}
		return true, capture
	})
}

func fill(
	screen tcell.Screen,
	x int,
	y int,
	width int,
	style tcell.Style,
) {
	for column := 0; column < width; column++ {
		screen.SetContent(x+column, y, ' ', nil, style)
	}
}

func drawText(
	screen tcell.Screen,
	text string,
	x int,
	y int,
	width int,
	style tcell.Style,
) {
	position := 0
	for _, character := range text {
		characterWidth := runewidth.RuneWidth(character)
		if characterWidth <= 0 {
			continue
		}
		if position+characterWidth > width {
			break
		}
		screen.SetContent(x+position, y, character, nil, style)
		position += characterWidth
	}
}
