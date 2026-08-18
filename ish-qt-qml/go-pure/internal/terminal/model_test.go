package terminal

import "testing"

func TestModelFeedAndSnapshot(t *testing.T) {
	m := NewModel(10, 3)
	m.Feed([]byte("hello\r\n\x1b[31mred\x1b[0m"))
	s := m.Snapshot()
	if s.Cols != 10 || s.Rows != 3 {
		t.Fatalf("unexpected size: %dx%d", s.Cols, s.Rows)
	}
	if len(s.Cells) != 30 {
		t.Fatalf("unexpected cell count: %d", len(s.Cells))
	}
	if s.Cells[0].Rune != 'h' {
		t.Fatalf("first rune = %q, want h", s.Cells[0].Rune)
	}
	foundRed := false
	for _, cell := range s.Cells {
		if cell.Rune == 'r' && cell.Brush.FG.R > 100 && cell.Brush.FG.G < 100 {
			foundRed = true
			break
		}
	}
	if !foundRed {
		t.Fatal("red SGR cell was not found")
	}
}
