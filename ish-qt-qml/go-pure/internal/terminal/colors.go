package terminal

import "github.com/viktomas/gritty/buffer"

func ansiColor(index int, bright bool) buffer.Color {
	dark := ansiDarkColors()
	light := ansiLightColors()
	if index < 0 || index >= len(dark) {
		return buffer.DefaultFG
	}
	if bright {
		return light[index]
	}
	return dark[index]
}

func ansiDarkColors() [8]buffer.Color {
	return [8]buffer.Color{
		buffer.NewColor(0, 0, 0), buffer.NewColor(205, 49, 49),
		buffer.NewColor(13, 188, 121), buffer.NewColor(229, 229, 16),
		buffer.NewColor(36, 114, 200), buffer.NewColor(188, 63, 188),
		buffer.NewColor(17, 168, 205), buffer.NewColor(229, 229, 229),
	}
}

func ansiLightColors() [8]buffer.Color {
	return [8]buffer.Color{
		buffer.NewColor(102, 102, 102), buffer.NewColor(241, 76, 76),
		buffer.NewColor(35, 209, 139), buffer.NewColor(245, 245, 67),
		buffer.NewColor(59, 142, 234), buffer.NewColor(214, 112, 214),
		buffer.NewColor(41, 184, 219), buffer.NewColor(255, 255, 255),
	}
}
