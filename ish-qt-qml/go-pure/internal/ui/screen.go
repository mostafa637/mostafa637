package ui

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/viktomas/gritty/buffer"

	"github.com/mostafa637/mostafa637/go-pure/internal/terminal"
)

type C = layout.Context
type D = layout.Dimensions

// InputSink is the only input dependency exposed to the UI.
type InputSink interface {
	Write([]byte) error
}

// Screen is the top-level iSH Gio screen. It is deliberately unaware of
// process creation and can be driven by a fake session in unit tests.
type Screen struct {
	Theme    *material.Theme
	Terminal *terminal.Model
	Input    InputSink
	KeyTag   struct{}
	Focused  bool
	Tab      widget.Clickable
	Ctrl     widget.Clickable
	Esc      widget.Clickable
	Paste    widget.Clickable
	Settings widget.Clickable
}

func NewScreen(model *terminal.Model, input InputSink) *Screen {
	return &Screen{
		Theme:    material.NewTheme(),
		Terminal: model,
		Input:    input,
	}
}

func (s *Screen) Layout(gtx C) D {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(s.layoutToolbar),
		layout.Flexed(1, s.layoutTerminal),
	)
}

func (s *Screen) layoutToolbar(gtx C) D {
	const height = 48
	gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(height))
	gtx.Constraints.Max.Y = gtx.Constraints.Min.Y
	paint.FillShape(gtx.Ops, color.NRGBA{R: 242, G: 242, B: 247, A: 255}, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}.Op())

	return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
			layout.Rigid(s.actionButton(gtx, &s.Tab, "Tab", func() { s.writeString("\t") })),
			layout.Rigid(s.actionButton(gtx, &s.Ctrl, "Ctrl", func() { s.writeBytes([]byte{0x1d}) })),
			layout.Rigid(s.actionButton(gtx, &s.Esc, "Esc", func() { s.writeBytes([]byte{0x1b}) })),
			layout.Flexed(1, func(gtx C) D { return D{Size: gtx.Constraints.Min} }),
			layout.Rigid(s.actionButton(gtx, &s.Paste, "Paste", func() {})),
			layout.Rigid(s.actionButton(gtx, &s.Settings, "Settings", func() {})),
		)
	})
}

func (s *Screen) actionButton(gtx C, click *widget.Clickable, label string, action func()) func(C) D {
	return func(gtx C) D {
		button := material.Button(s.Theme, click, label)
		button.TextSize = unit.Sp(13)
		button.Background = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		button.Color = color.NRGBA{R: 45, G: 45, B: 50, A: 255}
		button.CornerRadius = unit.Dp(5)
		dims := button.Layout(gtx)
		for click.Clicked(gtx) {
			action()
		}
		return dims
	}
}

func (s *Screen) layoutTerminal(gtx C) D {
	paint.FillShape(gtx.Ops, color.NRGBA{R: 40, G: 40, B: 40, A: 255}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	key.InputHintOp{Tag: &s.KeyTag, Hint: key.HintText}.Add(gtx.Ops)
	if !s.Focused {
		gtx.Execute(key.FocusCmd{Tag: &s.KeyTag})
		gtx.Execute(key.SoftKeyboardCmd{Show: true})
		s.Focused = true
	}

	for {
		event, ok := gtx.Event(key.Filter{Focus: &s.KeyTag})
		if !ok {
			break
		}
		if e, ok := event.(key.Event); ok && e.State == key.Press {
			s.handleKey(e)
		}
	}
	for {
		event, ok := gtx.Event(key.FocusFilter{Target: &s.KeyTag})
		if !ok {
			break
		}
		if e, ok := event.(key.EditEvent); ok {
			s.writeString(e.Text)
		}
	}

	snapshot := s.Terminal.Snapshot()
	cellWidth := gtx.Dp(unit.Dp(8))
	cellHeight := gtx.Dp(unit.Dp(18))
	if cellWidth < 1 {
		cellWidth = 8
	}
	if cellHeight < 1 {
		cellHeight = 18
	}
	for i, cell := range snapshot.Cells {
		x := (i % snapshot.Cols) * cellWidth
		y := (i / snapshot.Cols) * cellHeight
		if x >= gtx.Constraints.Max.X || y >= gtx.Constraints.Max.Y {
			continue
		}
		cellRect := image.Rect(x, y, x+cellWidth, y+cellHeight)
		bg, fg := brushColors(cell.Brush)
		paint.FillShape(gtx.Ops, bg, clip.Rect{Min: cellRect.Min, Max: cellRect.Max}.Op())
		if cell.Rune == 0 {
			continue
		}
		stack := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
		label := material.Label(s.Theme, unit.Sp(14), string(cell.Rune))
		label.Font = font.Font{Typeface: "monospace"}
		label.LineHeight = unit.Sp(18)
		label.Color = fg
		label.Layout(gtx)
		stack.Pop()
	}
	return D{Size: gtx.Constraints.Max}
}

func (s *Screen) handleKey(e key.Event) {
	if s.Input == nil {
		return
	}
	if e.Modifiers&key.ModCtrl != 0 {
		name := string(e.Name)
		if len(name) == 1 {
			r := name[0]
			if r >= 'a' && r <= 'z' {
				r -= 'a' - 1
			} else if r >= 'A' && r <= 'Z' {
				r -= 'A' - 1
			}
			s.writeBytes([]byte{r})
			return
		}
	}
	switch e.Name {
	case key.NameReturn, key.NameEnter:
		s.writeString("\r")
	case key.NameDeleteBackward:
		s.writeBytes([]byte{0x7f})
	case key.NameTab:
		s.writeString("\t")
	case key.NameEscape:
		s.writeBytes([]byte{0x1b})
	case key.NameUpArrow:
		s.writeString("\x1b[A")
	case key.NameDownArrow:
		s.writeString("\x1b[B")
	case key.NameLeftArrow:
		s.writeString("\x1b[D")
	case key.NameRightArrow:
		s.writeString("\x1b[C")
	default:
		if len(string(e.Name)) == 1 {
			s.writeString(string(e.Name))
		}
	}
}

func (s *Screen) writeString(value string) {
	s.writeBytes([]byte(value))
}

func (s *Screen) writeBytes(value []byte) {
	if len(value) == 0 || s.Input == nil {
		return
	}
	_ = s.Input.Write(value)
}

func brushColors(brush buffer.Brush) (color.NRGBA, color.NRGBA) {
	bg := color.NRGBA{R: brush.BG.R, G: brush.BG.G, B: brush.BG.B, A: 255}
	fg := color.NRGBA{R: brush.FG.R, G: brush.FG.G, B: brush.FG.B, A: 255}
	if brush.Invert {
		bg, fg = fg, bg
	}
	return bg, fg
}
