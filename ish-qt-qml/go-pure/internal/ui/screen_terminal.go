package ui

import (
	"image/color"

	"gioui.org/io/key"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

func (s *Screen) layoutTerminal(gtx C) D {
	paint.FillShape(gtx.Ops, color.NRGBA{R: 40, G: 40, B: 40, A: 255}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	key.InputHintOp{Tag: &s.KeyTag, Hint: key.HintText}.Add(gtx.Ops)
	s.focusKeyboard(gtx)
	s.handleEvents(gtx)
	cols, rows := terminalSize(gtx)
	if current := s.Terminal.Snapshot(); current.Cols != cols || current.Rows != rows {
		s.Terminal.Resize(cols, rows)
		if resize, ok := s.Input.(ResizeSink); ok {
			_ = resize.Resize(cols, rows)
		}
	}
	s.drawCells(gtx, cols, rows)
	return D{Size: gtx.Constraints.Max}
}

func (s *Screen) focusKeyboard(gtx C) {
	if s.Focused {
		return
	}
	gtx.Execute(key.FocusCmd{Tag: &s.KeyTag})
	gtx.Execute(key.SoftKeyboardCmd{Show: true})
	s.Focused = true
}

func (s *Screen) handleEvents(gtx C) {
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
}

func terminalSize(gtx C) (int, int) {
	w, h := gtx.Dp(unit.Dp(8)), gtx.Dp(unit.Dp(18))
	if w < 1 {
		w = 8
	}
	if h < 1 {
		h = 18
	}
	cols, rows := gtx.Constraints.Max.X/w, gtx.Constraints.Max.Y/h
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return cols, rows
}
