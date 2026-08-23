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
	cols, rows = validSize(cols, rows)
	return &Model{buf: buffer.New(cols, rows), parser: parser.New(), cols: cols, rows: rows}
}

func validSize(cols, rows int) (int, int) {
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}
	return cols, rows
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
	out := Snapshot{Cols: m.cols, Rows: m.rows, Cursor: m.buf.Cursor(), Cells: make([]Cell, len(cells))}
	for i, cell := range cells {
		out.Cells[i] = Cell{Rune: cell.R, Brush: cell.Brush}
	}
	return out
}
