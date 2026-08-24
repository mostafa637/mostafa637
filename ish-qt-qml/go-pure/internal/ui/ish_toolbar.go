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
		name := label
		cells = append(cells, layout.Flexed(1, func(gtx C) D {
			caption := material.Caption(theme, name)
			caption.Color = color.NRGBA{R: 210, G: 210, B: 215, A: 255}
			caption.TextSize = unit.Sp(9)
			return caption.Layout(gtx)
		}))
	}
	return cells
}

func (s *Screen) accessoryButtons(gtx C) D {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(1, s.singleButton(&s.Tab, "↹", func() { s.writeString("\t") })),
		layout.Flexed(1, s.singleButton(&s.Ctrl, "⌃", func() { s.writeBytes([]byte{0x1d}) })),
		layout.Flexed(1, s.singleButton(&s.Esc, "⎋", func() { s.writeBytes([]byte{0x1b}) })),
		layout.Flexed(1, s.arrowGroup),
		layout.Flexed(1, s.singleButton(&s.Settings, "⚙", func() { s.SettingsOpen = true })),
		layout.Flexed(1, s.singleButton(&s.Paste, "▣", func() { s.PasteRequested = true })),
		layout.Flexed(1, s.singleButton(&s.Hide, "⌨", func() { s.Focused = false })))
}

func (s *Screen) arrowGroup(gtx C) D {
	arrows := []toolbarButton{
		{&s.Left, "←", func() { s.writeBytes([]byte{0x1b, '[', 'D'}) }},
		{&s.Up, "↑", func() { s.writeBytes([]byte{0x1b, '[', 'A'}) }},
		{&s.Down, "↓", func() { s.writeBytes([]byte{0x1b, '[', 'B'}) }},
		{&s.Right, "→", func() { s.writeBytes([]byte{0x1b, '[', 'C'}) }},
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, toolbarCells(s, arrows)...)
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

func (s *Screen) singleButton(click *widget.Clickable, glyph string, action func()) func(C) D {
	return func(gtx C) D { return s.accessoryButton(gtx, toolbarButton{click, glyph, action}) }
}

func (s *Screen) accessoryButton(gtx C, item toolbarButton) D {
	button := material.Button(s.Theme, item.click, item.glyph)
	button.TextSize = unit.Sp(16)
	button.Background = color.NRGBA{R: 75, G: 75, B: 80, A: 255}
	button.Color = color.NRGBA{R: 240, G: 240, B: 242, A: 255}
	button.CornerRadius = unit.Dp(3)
	button.Inset = layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(2), Right: unit.Dp(2)}
	dims := button.Layout(gtx)
	for item.click.Clicked(gtx) {
		item.action()
	}
	return dims
}
