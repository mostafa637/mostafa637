package terminal

import (
	"github.com/viktomas/gritty/buffer"
	"github.com/viktomas/gritty/parser"
)

func (m *Model) applyCursorCSI(op parser.Operation) {
	p := func(i, def int) int { return op.Param(i, def) }
	c := m.buf.Cursor()
	switch op.R {
	case 'A':
		m.buf.MoveCursorRelative(0, -p(0, 1))
	case 'B', 'e':
		m.buf.MoveCursorRelative(0, p(0, 1))
	case 'C', 'a':
		m.buf.MoveCursorRelative(p(0, 1), 0)
	case 'D':
		m.buf.MoveCursorRelative(-p(0, 1), 0)
	case 'E':
		m.buf.MoveCursorRelative(0, p(0, 1))
		m.buf.CR()
	case 'F':
		m.buf.MoveCursorRelative(0, -p(0, 1))
		m.buf.CR()
	case 'G', '`':
		m.buf.SetCursor(p(0, 1)-1, c.Y)
	case 'd':
		m.buf.SetCursor(c.X, p(0, 1)-1)
	case 'H', 'f':
		m.buf.SetCursor(p(1, 1)-1, p(0, 1)-1)
	}
}

func (m *Model) applyEraseCSI(op parser.Operation) {
	c := m.buf.Cursor()
	if op.R == 'J' {
		m.buf.ClearLines(0, m.rows)
		return
	}
	m.buf.ClearCurrentLine(c.X, m.cols)
}

func (m *Model) applyModeCSI(op parser.Operation) {
	if len(op.Params) == 0 || op.Params[0] != 1049 {
		return
	}
	if op.R == 'h' {
		m.buf.SwitchToAlternateBuffer()
		return
	}
	m.buf.SwitchToPrimaryBuffer()
}

func (m *Model) applySGR(params []int) {
	if len(params) == 0 {
		params = []int{0}
	}
	for _, p := range params {
		m.applySGRParam(p)
	}
}

func (m *Model) applySGRParam(p int) {
	brush := m.buf.Brush()
	switch {
	case p == 0:
		m.buf.ResetBrush()
	case p == 1:
		brush.Bold = true
	case p == 7:
		brush.Invert = true
	case p == 22:
		brush.Bold = false
	case p == 27:
		brush.Invert = false
	case p >= 30 && p <= 37:
		brush.FG = ansiColor(p-30, false)
	case p == 39:
		brush.FG = buffer.DefaultFG
	case p >= 40 && p <= 47:
		brush.BG = ansiColor(p-40, false)
	case p == 49:
		brush.BG = buffer.DefaultBG
	case p >= 90 && p <= 97:
		brush.FG = ansiColor(p-90, true)
	case p >= 100 && p <= 107:
		brush.BG = ansiColor(p-100, true)
	default:
		return
	}
	m.buf.SetBrush(brush)
}
