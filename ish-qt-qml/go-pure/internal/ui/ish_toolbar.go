package ui

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

const accessoryButtonWidth = unit.Dp(31)
const accessoryButtonHeight = unit.Dp(43)

func (s *Screen) accessoryBar(gtx C) D {
	bar := color.NRGBA{R: 214, G: 218, B: 224, A: 255}
	paint.FillShape(gtx.Ops, bar, clip.Rect{Max: gtx.Constraints.Max}.Op())
	return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, s.accessoryStack)
}

func (s *Screen) accessoryStack(gtx C) D {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(s.accessoryButton(&s.Tab, "⇥", nil, func() { s.writeString("\t") })),
		layout.Rigid(toolbarGap),
		layout.Rigid(s.accessoryButton(&s.Ctrl, "⌃", nil, func() { s.writeBytes([]byte{0x1d}) })),
		layout.Rigid(toolbarGap),
		layout.Rigid(s.accessoryButton(&s.Esc, "⎋", nil, func() { s.writeBytes([]byte{0x1b}) })),
		layout.Rigid(toolbarGap),
		layout.Rigid(s.accessoryButton(&s.Arrows, "", nil, s.sendArrowUp)),
		layout.Flexed(1, func(gtx C) D { return D{} }),
		layout.Rigid(s.accessoryButton(&s.Info, "ⓘ", nil, func() { s.SettingsOpen = true })),
		layout.Rigid(toolbarGap),
		layout.Rigid(s.accessoryButton(&s.Paste, "", ishPasteImage, func() { s.PasteRequested = true })),
		layout.Rigid(toolbarGap),
		layout.Rigid(s.accessoryButton(&s.Hide, "", ishHideKeyboardImage, func() { s.Focused = false })),
	)
}

func toolbarGap(gtx C) D {
	return D{Size: image.Pt(gtx.Dp(unit.Dp(6)), 0)}
}

func (s *Screen) accessoryButton(click *widget.Clickable, label string, icon image.Image, action func()) func(C) D {
	return func(gtx C) D {
		for click.Clicked(gtx) {
			action()
		}
		return click.Layout(gtx, func(gtx C) D {
			return s.drawButton(gtx, click, label, icon)
		})
	}
}

func (s *Screen) drawButton(gtx C, click *widget.Clickable, label string, icon image.Image) D {
	size := image.Pt(gtx.Dp(accessoryButtonWidth), gtx.Dp(accessoryButtonHeight))
	gtx.Constraints = layout.Exact(size)
	fill := color.NRGBA{R: 172, G: 180, B: 190, A: 255}
	if !click.Pressed() {
		fill = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	}
	paint.FillShape(gtx.Ops, fill, clip.UniformRRect(image.Rectangle{Max: size}, gtx.Dp(unit.Dp(5))).Op(gtx.Ops))
	return s.buttonContent(gtx, label, icon)
}

func (s *Screen) buttonContent(gtx C, label string, icon image.Image) D {
	if icon != nil {
		img := widget.Image{Src: paint.NewImageOp(icon), Fit: widget.Contain, Position: layout.Center, Scale: 0.45}
		return img.Layout(gtx)
	}
	text := material.Body1(s.Theme, label)
	text.Color = color.NRGBA{A: 255}
	text.TextSize = unit.Sp(20)
	return text.Layout(gtx)
}

func (s *Screen) sendArrowUp() {
	s.writeBytes([]byte{0x1b, '[', 'A'})
}
