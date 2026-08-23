package ui

import (
	"image"

	"gioui.org/font"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/viktomas/gritty/buffer"
)

func (s *Screen) drawCells(gtx C, cols, rows int) {
	w, h := gtx.Dp(unit.Dp(8)), gtx.Dp(unit.Dp(18))
	for i, cell := range s.Terminal.Snapshot().Cells {
		x, y := (i%cols)*w, (i/cols)*h
		if x >= gtx.Constraints.Max.X || y >= gtx.Constraints.Max.Y {
			continue
		}
		s.drawCell(gtx, cell.Rune, cell.Brush, image.Rect(x, y, x+w, y+h))
	}
	_ = rows
}

func (s *Screen) drawCell(gtx C, r rune, brush buffer.Brush, rect image.Rectangle) {
	bg, fg := brushColors(brush)
	paint.FillShape(gtx.Ops, bg, clip.Rect{Min: rect.Min, Max: rect.Max}.Op())
	if r == 0 {
		return
	}
	stack := op.Offset(rect.Min).Push(gtx.Ops)
	label := material.Label(s.Theme, unit.Sp(14), string(r))
	label.Font, label.LineHeight, label.Color = font.Font{Typeface: "monospace"}, unit.Sp(18), fg
	label.Layout(gtx)
	stack.Pop()
}
