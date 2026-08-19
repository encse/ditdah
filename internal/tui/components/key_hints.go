package components

import (
	"strings"

	"morsemanual/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

const keyHintSeparator = "   "

type keyHints struct {
	*tview.Box
	bindings   []keybinding.Binding
	widths     []int
	align      int
	text       tcell.Color
	background tcell.Color
	changed    func()
}

func newKeyHints(
	theme Theme,
	align int,
	changed func(),
) *keyHints {
	return &keyHints{
		Box:        tview.NewBox().SetBackgroundColor(theme.Background),
		align:      align,
		text:       theme.MutedText,
		background: theme.Background,
		changed:    changed,
	}
}

func (h *keyHints) SetBindings(bindings []keybinding.Binding) {
	h.bindings = keybinding.Visible(bindings)
	h.widths = h.widths[:0]
	for _, binding := range h.bindings {
		hint := binding.Hint()
		h.widths = append(h.widths, runewidth.StringWidth(
			hint.Keys+" "+hint.Description,
		))
	}
}

func (h *keyHints) ContentWidth() int {
	width := 0
	for _, itemWidth := range h.widths {
		width += itemWidth
	}
	if len(h.widths) > 1 {
		width += (len(h.widths) - 1) * len(keyHintSeparator)
	}
	return width
}

func (h *keyHints) Text() string {
	parts := make([]string, 0, len(h.bindings))
	for _, binding := range h.bindings {
		hint := binding.Hint()
		parts = append(parts, hint.Keys+" "+hint.Description)
	}
	return strings.Join(parts, keyHintSeparator)
}

func (h *keyHints) Draw(screen tcell.Screen) {
	h.Box.DrawForSubclass(screen, h)
	left, top, width, height := h.GetInnerRect()
	if width < 1 || height < 1 {
		return
	}

	position := h.contentStart(left, width)
	normal := tcell.StyleDefault.Foreground(h.text).Background(h.background)
	keyStyle := normal.Bold(true)
	right := left + width
	for _, binding := range h.bindings {
		hint := binding.Hint()
		keyWidth := runewidth.StringWidth(hint.Keys)
		drawText(screen, hint.Keys, position, top, max(0, right-position), keyStyle)
		position += keyWidth

		position++
		drawText(
			screen,
			hint.Description,
			position,
			top,
			max(0, right-position),
			normal,
		)
		position += runewidth.StringWidth(hint.Description) +
			len(keyHintSeparator)
	}
}

func (h *keyHints) MouseHandler() mouseHandler {
	return h.WrapMouseHandler(func(
		action tview.MouseAction,
		event *tcell.EventMouse,
		_ func(tview.Primitive),
	) (bool, tview.Primitive) {
		if action != tview.MouseLeftClick &&
			action != tview.MouseLeftDoubleClick {
			return false, nil
		}
		x, y := event.Position()
		left, top, width, height := h.GetInnerRect()
		if x < left || x >= left+width || y < top || y >= top+height {
			return false, nil
		}
		position := h.contentStart(left, width)
		for index, itemWidth := range h.widths {
			if x >= position && x < position+itemWidth {
				h.bindings[index].Invoke()
				notify(h.changed)
				return true, nil
			}
			position += itemWidth + len(keyHintSeparator)
		}
		return false, nil
	})
}

func (h *keyHints) contentStart(left, width int) int {
	contentWidth := h.ContentWidth()
	if h.align == tview.AlignCenter {
		return left + max(0, (width-contentWidth)/2)
	}
	if h.align == tview.AlignRight {
		return left + max(0, width-contentWidth)
	}
	return left
}
