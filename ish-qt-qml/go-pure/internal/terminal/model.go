package terminal

import (
	"sync"

	"github.com/viktomas/gritty/buffer"
	"github.com/viktomas/gritty/parser"
)

// Cell is an immutable value used by the Gio renderer.
type Cell struct {
	Rune  rune
	Brush buffer.Brush
}

// Snapshot is a render-ready copy of the terminal state.
type Snapshot struct {
	Cols   int
	Rows   int
	Cursor buffer.Cursor
	Cells  []Cell
}

// Model owns gritty's VT parser and screen buffer. It contains no Gio code.
type Model struct {
	mu     sync.RWMutex
	buf    *buffer.Buffer
	parser *parser.Parser
	cols   int
	rows   int
}

func NewModel(cols, rows int) *Model {
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}
	return &Model{
		buf:    buffer.New(cols, rows),
		parser: parser.New(),
		cols:   cols,
		rows:   rows,
	}
}

func (m *Model) Resize(cols, rows int) {
	if cols < 1 || rows < 1 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buf.Resize(buffer.BufferSize{Cols: cols, Rows: rows})
	m.cols, m.rows = cols, rows
}

func (m *Model) Feed(p []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, op := range m.parser.Parse(p) {
		m.apply(op)
	}
}

func (m *Model) Snapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cells := m.buf.Runes()
	out := Snapshot{
		Cols:   m.cols,
		Rows:   m.rows,
		Cursor: m.buf.Cursor(),
		Cells:  make([]Cell, len(cells)),
	}
	for i, cell := range cells {
		out.Cells[i] = Cell{Rune: cell.R, Brush: cell.Brush}
	}
	return out
}

func (m *Model) apply(op parser.Operation) {
	switch op.T {
	case parser.OpPrint:
		m.buf.WriteRune(op.R)
	case parser.OpExecute:
		switch op.R {
		case '\b':
			m.buf.Backspace()
		case '\t':
			m.buf.Tab()
		case '\r':
			m.buf.CR()
		case '\n':
			m.buf.LF()
		case 0x8d:
			m.buf.ReverseIndex()
		}
	case parser.OpESC:
		switch op.R {
		case '7':
			m.buf.SaveCursor()
		case '8':
			m.buf.RestoreCursor()
		case 'M':
			m.buf.ReverseIndex()
		}
	case parser.OpCSI:
		m.applyCSI(op)
	case parser.OpOSC:
		// OSC is intentionally ignored by the first renderer. It can be added
		// later for titles, hyperlinks, and clipboard integration.
	}
}

func (m *Model) applyCSI(op parser.Operation) {
	p := func(i, def int) int { return op.Param(i, def) }
	cursor := m.buf.Cursor()
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
		m.buf.SetCursor(p(0, 1)-1, cursor.Y)
	case 'd':
		m.buf.SetCursor(cursor.X, p(0, 1)-1)
	case 'H', 'f':
		m.buf.SetCursor(p(1, 1)-1, p(0, 1)-1)
	case 'J':
		m.buf.ClearLines(0, m.rows)
	case 'K':
		m.buf.ClearCurrentLine(cursor.X, m.cols)
	case 'm':
		m.applySGR(op.Params)
	case 'h':
		if len(op.Params) > 0 && op.Params[0] == 1049 {
			m.buf.SwitchToAlternateBuffer()
		}
	case 'l':
		if len(op.Params) > 0 && op.Params[0] == 1049 {
			m.buf.SwitchToPrimaryBuffer()
		}
	}
}

func (m *Model) applySGR(params []int) {
	if len(params) == 0 {
		params = []int{0}
	}
	brush := m.buf.Brush()
	for _, p := range params {
		switch p {
		case 0:
			m.buf.ResetBrush()
			brush = m.buf.Brush()
		case 1:
			brush.Bold = true
		case 7:
			brush.Invert = true
		case 22:
			brush.Bold = false
		case 27:
			brush.Invert = false
		case 30, 31, 32, 33, 34, 35, 36, 37:
			brush.FG = ansiColor(p-30, false)
		case 39:
			brush.FG = buffer.DefaultFG
		case 40, 41, 42, 43, 44, 45, 46, 47:
			brush.BG = ansiColor(p-40, false)
		case 49:
			brush.BG = buffer.DefaultBG
		case 90, 91, 92, 93, 94, 95, 96, 97:
			brush.FG = ansiColor(p-90, true)
		case 100, 101, 102, 103, 104, 105, 106, 107:
			brush.BG = ansiColor(p-100, true)
		}
	}
	m.buf.SetBrush(brush)
}

func ansiColor(index int, bright bool) buffer.Color {
	dark := [8]buffer.Color{
		buffer.NewColor(0, 0, 0), buffer.NewColor(205, 49, 49), buffer.NewColor(13, 188, 121),
		buffer.NewColor(229, 229, 16), buffer.NewColor(36, 114, 200), buffer.NewColor(188, 63, 188),
		buffer.NewColor(17, 168, 205), buffer.NewColor(229, 229, 229),
	}
	light := [8]buffer.Color{
		buffer.NewColor(102, 102, 102), buffer.NewColor(241, 76, 76), buffer.NewColor(35, 209, 139),
		buffer.NewColor(245, 245, 67), buffer.NewColor(59, 142, 234), buffer.NewColor(214, 112, 214),
		buffer.NewColor(41, 184, 219), buffer.NewColor(255, 255, 255),
	}
	if index < 0 || index >= len(dark) {
		return buffer.DefaultFG
	}
	if bright {
		return light[index]
	}
	return dark[index]
}
