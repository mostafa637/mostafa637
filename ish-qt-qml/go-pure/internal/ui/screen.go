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
	"github.com/mostafa637/mostafa637/go-pure/internal/terminal"
)

type C = layout.Context
type D = layout.Dimensions

type InputSink interface{ Write([]byte) error }
type ResizeSink interface{ Resize(cols, rows int) error }

type Screen struct {
	Theme          *material.Theme
	Terminal       *terminal.Model
	Input          InputSink
	KeyTag         struct{}
	ClipboardTag   struct{}
	PasteRequested bool
	SettingsOpen   bool
	settingsState  *settingsState
	Focused        bool

	Tab, Ctrl, Esc, Paste, Settings widget.Clickable
}

func NewScreen(model *terminal.Model, input InputSink) *Screen {
	return &Screen{Theme: material.NewTheme(), Terminal: model, Input: input}
}

func (s *Screen) Layout(gtx C) D {
	content := s.layoutTerminal
	if s.SettingsOpen {
		content = s.layoutSettings
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(s.layoutToolbar), layout.Flexed(1, content))
}

func (s *Screen) layoutToolbar(gtx C) D {
	const height = 48
	gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(height))
	gtx.Constraints.Max.Y = gtx.Constraints.Min.Y
	paint.FillShape(gtx.Ops, color.NRGBA{R: 242, G: 242, B: 247, A: 255}, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}.Op())
	return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, s.toolbarItems)
}

func (s *Screen) toolbarItems(gtx C) D {
	return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
		layout.Rigid(s.actionButton(gtx, &s.Tab, "Tab", func() { s.writeString("\t") })),
		layout.Rigid(s.actionButton(gtx, &s.Ctrl, "Ctrl", func() { s.writeBytes([]byte{0x1d}) })),
		layout.Rigid(s.actionButton(gtx, &s.Esc, "Esc", func() { s.writeBytes([]byte{0x1b}) })),
		layout.Flexed(1, func(gtx C) D { return D{Size: gtx.Constraints.Min} }),
		layout.Rigid(s.actionButton(gtx, &s.Paste, "Paste", func() { s.PasteRequested = true })),
		layout.Rigid(s.actionButton(gtx, &s.Settings, "Settings", func() { s.SettingsOpen = true })),
	)
}

func (s *Screen) actionButton(gtx C, click *widget.Clickable, label string, action func()) func(C) D {
	return func(gtx C) D {
		button := material.Button(s.Theme, click, label)
		button.TextSize, button.Background = unit.Sp(13), color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		button.Color, button.CornerRadius = color.NRGBA{R: 45, G: 45, B: 50, A: 255}, unit.Dp(5)
		dims := button.Layout(gtx)
		for click.Clicked(gtx) {
			action()
		}
		return dims
	}
}
