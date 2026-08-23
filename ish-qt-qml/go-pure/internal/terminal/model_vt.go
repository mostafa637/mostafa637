package terminal

import "github.com/viktomas/gritty/parser"

func (m *Model) apply(op parser.Operation) {
	switch op.T {
	case parser.OpPrint:
		m.buf.WriteRune(op.R)
	case parser.OpExecute:
		m.applyExecute(op.R)
	case parser.OpESC:
		m.applyEscape(op.R)
	case parser.OpCSI:
		m.applyCSI(op)
	}
}

func (m *Model) applyExecute(r rune) {
	switch r {
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
}

func (m *Model) applyEscape(r rune) {
	switch r {
	case '7':
		m.buf.SaveCursor()
	case '8':
		m.buf.RestoreCursor()
	case 'M':
		m.buf.ReverseIndex()
	}
}

func (m *Model) applyCSI(op parser.Operation) {
	switch op.R {
	case 'A', 'B', 'C', 'D', 'E', 'F', 'G', '`', 'd', 'H', 'f':
		m.applyCursorCSI(op)
	case 'J', 'K':
		m.applyEraseCSI(op)
	case 'm':
		m.applySGR(op.Params)
	case 'h', 'l':
		m.applyModeCSI(op)
	}
}
