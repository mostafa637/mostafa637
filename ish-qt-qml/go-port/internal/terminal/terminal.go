package terminal

import (
	"sync"
	"unicode/utf8"

	"github.com/viktomas/gritty/buffer"
	"github.com/viktomas/gritty/parser"
)

// Terminal is the gritty-backed terminal state used by the Gio front end.
// It parses the byte stream from an iSH/PTY backend and keeps a cell grid,
// cursor, scroll region, alternate screen, and SGR attributes.
type Terminal struct {
	mu          sync.Mutex
	parser      *parser.Parser
	buf         *buffer.Buffer
	cols        int
	rows        int
	pendingUTF8 []byte
}

func New(cols, rows int) *Terminal {
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}
	return &Terminal{
		parser: parser.New(),
		buf:    buffer.New(cols, rows),
		cols:   cols,
		rows:   rows,
	}
}

// Feed consumes arbitrary chunks. gritty's parser retains escape-sequence
// state between calls, which is required because PTY reads split sequences.
func (t *Terminal) Feed(data []byte) {
	if len(data) == 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	data = append(t.pendingUTF8, data...)
	t.pendingUTF8 = nil
	for i := 0; i < len(data); {
		if data[i] < 0x80 {
			for _, op := range t.parser.Parse(data[i : i+1]) {
				t.apply(op)
			}
			i++
			continue
		}
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size == 1 {
			if utf8.FullRune(data[i:]) {
				t.buf.WriteRune(utf8.RuneError)
				i++
				continue
			}
			t.pendingUTF8 = append(t.pendingUTF8, data[i:]...)
			return
		}
		t.buf.WriteRune(r)
		i += size
	}
}

func (t *Terminal) Resize(cols, rows int) {
	if cols < 1 || rows < 1 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if cols == t.cols && rows == t.rows {
		return
	}
	t.cols, t.rows = cols, rows
	t.buf.Resize(buffer.BufferSize{Cols: cols, Rows: rows})
}

func (t *Terminal) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.buf.String()
}

func (t *Terminal) Runes() []buffer.BrushedRune {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]buffer.BrushedRune(nil), t.buf.Runes()...)
}

func (t *Terminal) Cursor() buffer.Cursor {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.buf.Cursor()
}

func (t *Terminal) apply(op parser.Operation) {
	switch op.T {
	case parser.OpPrint:
		t.writeRune(op.R)
	case parser.OpExecute:
		switch byte(op.R) {
		case '\a':
		case '\b':
			t.buf.Backspace()
		case '\t':
			t.buf.Tab()
		case '\n', '\v', '\f':
			if t.buf.Cursor().X >= t.cols {
				t.buf.CR()
			}
			t.buf.LF()
		case '\r':
			t.buf.CR()
		case '\x0e', '\x0f':
			// Shift-in/shift-out are not relevant to UTF-8 terminal text.
		}
	case parser.OpESC:
		t.applyESC(op)
	case parser.OpCSI:
		t.applyCSI(op)
	case parser.OpOSC:
		// Window-title and hyperlink OSC sequences do not change the grid.
	}
}

func (t *Terminal) writeRune(r rune) {
	if t.buf.Cursor().X >= t.cols {
		t.buf.CR()
	}
	t.buf.WriteRune(r)
}

func (t *Terminal) applyESC(op parser.Operation) {
	switch op.R {
	case '7':
		t.buf.SaveCursor()
	case '8':
		t.buf.RestoreCursor()
	case 'D':
		t.buf.LF()
	case 'E':
		t.buf.CR()
		t.buf.LF()
	case 'M':
		t.buf.ReverseIndex()
	case 'c':
		t.buf = buffer.New(t.cols, t.rows)
	}
}

func (t *Terminal) applyCSI(op parser.Operation) {
	p := func(i, def int) int { return op.Param(i, def) }
	cur := t.buf.Cursor()
	switch op.R {
	case 'A':
		t.buf.MoveCursorRelative(0, -p(0, 1))
	case 'B', 'e':
		t.buf.MoveCursorRelative(0, p(0, 1))
	case 'C', 'a':
		t.buf.MoveCursorRelative(p(0, 1), 0)
	case 'D':
		t.buf.MoveCursorRelative(-p(0, 1), 0)
	case 'E':
		t.buf.MoveCursorRelative(0, p(0, 1))
		t.buf.CR()
	case 'F':
		t.buf.MoveCursorRelative(0, -p(0, 1))
		t.buf.CR()
	case 'G', '`':
		t.buf.SetCursor(p(0, 1)-1, cur.Y)
	case 'd':
		t.buf.SetCursor(cur.X, p(0, 1)-1)
	case 'H', 'f':
		t.buf.SetCursor(p(1, 1)-1, p(0, 1)-1)
	case 'J':
		t.eraseDisplay(p(0, 0))
	case 'K':
		t.eraseLine(p(0, 0))
	case 'L':
		t.buf.InsertLine(p(0, 1))
	case 'M':
		t.buf.DeleteLine(p(0, 1))
	case 'P':
		t.buf.DeleteCharacter(p(0, 1))
	case 'S':
		t.buf.ScrollUp(p(0, 1))
	case 'T':
		// gritty's buffer exposes reverse scrolling through ReverseIndex.
		for i := 0; i < p(0, 1); i++ {
			t.buf.ReverseIndex()
		}
	case 'X':
		t.buf.ClearCurrentLine(cur.X, cur.X+p(0, 1))
	case 'm':
		t.applySGR(op)
	case 'r':
		t.buf.SetScrollArea(p(0, 1)-1, p(1, t.rows))
	case 's':
		t.buf.SaveCursor()
	case 'u':
		t.buf.RestoreCursor()
	case 'h':
		if op.Intermediate == "?" && hasParam(op, 1049) {
			t.buf.SwitchToAlternateBuffer()
		}
	case 'l':
		if op.Intermediate == "?" && hasParam(op, 1049) {
			t.buf.SwitchToPrimaryBuffer()
		}
	}
}

func (t *Terminal) eraseDisplay(mode int) {
	cur := t.buf.Cursor()
	switch mode {
	case 0:
		t.buf.ClearCurrentLine(cur.X, t.cols)
		t.buf.ClearLines(cur.Y+1, t.rows)
	case 1:
		t.buf.ClearLines(0, cur.Y)
		t.buf.ClearCurrentLine(0, cur.X+1)
	case 2, 3:
		t.buf.ClearLines(0, t.rows)
	}
}

func (t *Terminal) eraseLine(mode int) {
	cur := t.buf.Cursor()
	switch mode {
	case 0:
		t.buf.ClearCurrentLine(cur.X, t.cols)
	case 1:
		t.buf.ClearCurrentLine(0, cur.X+1)
	case 2:
		t.buf.ClearCurrentLine(0, t.cols)
	}
}

func hasParam(op parser.Operation, want int) bool {
	for _, p := range op.Params {
		if p == want {
			return true
		}
	}
	return false
}

func (t *Terminal) applySGR(op parser.Operation) {
	params := op.Params
	if len(params) == 0 {
		params = []int{0}
	}
	for i := 0; i < len(params); i++ {
		code := params[i]
		br := t.buf.Brush()
		switch {
		case code == 0:
			t.buf.ResetBrush()
		case code == 1:
			br.Bold = true
			t.buf.SetBrush(br)
		case code == 22:
			br.Bold = false
			t.buf.SetBrush(br)
		case code == 7:
			br.Invert = true
			t.buf.SetBrush(br)
		case code == 27:
			br.Invert = false
			t.buf.SetBrush(br)
		case code >= 30 && code <= 37:
			br.FG = ansiNormal(code - 30)
			t.buf.SetBrush(br)
		case code >= 40 && code <= 47:
			br.BG = ansiNormal(code - 40)
			t.buf.SetBrush(br)
		case code >= 90 && code <= 97:
			br.FG = ansiBright(code - 90)
			t.buf.SetBrush(br)
		case code >= 100 && code <= 107:
			br.BG = ansiBright(code - 100)
			t.buf.SetBrush(br)
		case code == 39:
			br.FG = buffer.DefaultFG
			t.buf.SetBrush(br)
		case code == 49:
			br.BG = buffer.DefaultBG
			t.buf.SetBrush(br)
		case code == 38 || code == 48:
			if i+1 < len(params) && params[i+1] == 5 && i+2 < len(params) {
				if code == 38 {
					br.FG = ansi256(params[i+2])
				} else {
					br.BG = ansi256(params[i+2])
				}
				t.buf.SetBrush(br)
				i += 2
			} else if i+4 < len(params) && params[i+1] == 2 {
				c := buffer.NewColor(uint8(clamp(params[i+2])), uint8(clamp(params[i+3])), uint8(clamp(params[i+4])))
				if code == 38 {
					br.FG = c
				} else {
					br.BG = c
				}
				t.buf.SetBrush(br)
				i += 4
			}
		}
	}
}

func clamp(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

func ansiNormal(n int) buffer.Color {
	colors := [...]buffer.Color{
		{R: 0, G: 0, B: 0}, {R: 205, G: 49, B: 49}, {R: 13, G: 188, B: 121}, {R: 229, G: 229, B: 16},
		{R: 36, G: 114, B: 200}, {R: 188, G: 63, B: 188}, {R: 17, G: 168, B: 205}, {R: 229, G: 229, B: 229},
	}
	if n < 0 || n >= len(colors) {
		return colors[0]
	}
	return colors[n]
}

func ansiBright(n int) buffer.Color {
	colors := [...]buffer.Color{
		{R: 102, G: 102, B: 102}, {R: 241, G: 76, B: 76}, {R: 35, G: 209, B: 139}, {R: 245, G: 245, B: 67},
		{R: 59, G: 142, B: 234}, {R: 214, G: 112, B: 214}, {R: 41, G: 184, B: 219}, {R: 229, G: 229, B: 229},
	}
	if n < 0 || n >= len(colors) {
		return colors[0]
	}
	return colors[n]
}

func ansi256(n int) buffer.Color {
	n = clamp(n)
	if n < 8 {
		return ansiNormal(n)
	}
	if n < 16 {
		return ansiBright(n - 8)
	}
	if n < 232 {
		n -= 16
		levels := [...]uint8{0, 95, 135, 175, 215, 255}
		return buffer.NewColor(levels[(n/36)%6], levels[(n/6)%6], levels[n%6])
	}
	level := uint8(8 + (n-232)*10)
	return buffer.NewColor(level, level, level)
}
