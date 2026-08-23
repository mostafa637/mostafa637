package ui

import (
	"image/color"

	"github.com/viktomas/gritty/buffer"
)

func brushColors(brush buffer.Brush) (color.NRGBA, color.NRGBA) {
	bg := color.NRGBA{R: brush.BG.R, G: brush.BG.G, B: brush.BG.B, A: 255}
	fg := color.NRGBA{R: brush.FG.R, G: brush.FG.G, B: brush.FG.B, A: 255}
	if brush.Invert {
		bg, fg = fg, bg
	}
	return bg, fg
}
