package ui

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func (s *Screen) accessoryRows(gtx C) D {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(s.accessoryLabels), layout.Flexed(1, s.accessoryButtons))
}

func (s *Screen) accessoryLabels(gtx C) D {
	labels := []string{"Tab", "Ctrl", "Esc", "Arrows", "Settings", "Paste", "Hide"}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, labelCells(s.Theme, labels)...)
}

func labelCells(theme *material.Theme, labels []string) []layout.FlexChild {
	cells := make([]layout.FlexChild, 0, len(labels))
	for _, label := range labels {
		text := label
		cells = append(cells, layout.Flexed(1, func(gtx C) D {
			caption := material.Caption(theme, text)
			caption.Color = color.NRGBA{R: 210, G: 210, B: 215, A: 255}
			caption.TextSize = unit.Sp(9)
			return caption.Layout(gtx)
		}))
	}
	return cells
}

func (s *Screen) accessoryButtons(gtx C) D {
	buttons := []toolbarButton{
		{&s.Tab, "↹", func() { s.writeString("\t") }},
		{&s.Ctrl, "⌃", func() { s.writeBytes([]byte{0x1d}) }},
		{&s.Esc, "⎋", func() { s.writeBytes([]byte{0x1b}) }},
		{&s.Arrows, "↕", func() { s.writeBytes([]byte{0x1b, '[', 'A'}) }},
		{&s.Settings, "⚙", func() { s.SettingsOpen = true }},
		{&s.Paste, "▣", func() { s.PasteRequested = true }},
		{&s.Hide, "⌨", func() { s.Focused = false }},
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, toolbarCells(s, buttons)...)
}

type toolbarButton struct {
	click  *widget.Clickable
	glyph  string
	action func()
}

func toolbarCells(s *Screen, buttons []toolbarButton) []layout.FlexChild {
	cells := make([]layout.FlexChild, 0, len(buttons))
	for _, button := range buttons {
		item := button
		cells = append(cells, layout.Flexed(1, func(gtx C) D {
			return s.accessoryButton(gtx, item)
		}))
	}
	return cells
}

func (s *Screen) accessoryButton(gtx C, item toolbarButton) D {
	button := material.Button(s.Theme, item.click, item.glyph)
	button.TextSize = unit.Sp(18)
	button.Background = color.NRGBA{R: 75, G: 75, B: 80, A: 255}
	button.Color = color.NRGBA{R: 240, G: 240, B: 242, A: 255}
	button.CornerRadius = unit.Dp(3)
	dims := button.Layout(gtx)
	for item.click.Clicked(gtx) {
		item.action()
	}
	return dims
}
