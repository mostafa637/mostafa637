package ui

import (
	"image/color"

	"github.com/viktomas/gritty/buffer"
)

func brushColors(brush buffer.Brush) (color.NRGBA, color.NRGBA) {
	bg, fg := colorFrom(brush.BG), colorFrom(brush.FG)
	if isDefaultBrush(brush) {
		bg, fg = color.NRGBA{R: 250, G: 250, B: 250, A: 255}, color.NRGBA{A: 255}
	}
	if brush.Invert {
		bg, fg = fg, bg
	}
	return bg, fg
}

func colorFrom(value buffer.Color) color.NRGBA {
	return color.NRGBA{R: value.R, G: value.G, B: value.B, A: 255}
}

func isDefaultBrush(brush buffer.Brush) bool {
	return brush.BG == buffer.DefaultBG && brush.FG == buffer.DefaultFG
}
