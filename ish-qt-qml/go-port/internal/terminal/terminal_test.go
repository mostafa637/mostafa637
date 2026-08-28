package terminal

import "testing"

func TestFeedUTF8AcrossChunks(t *testing.T) {
	tm := New(12, 2)
	tm.Feed([]byte("Alp"))
	tm.Feed([]byte("i\xD9"))
	tm.Feed([]byte("\x84\xD9\x8A\xD9\x86\xD9\x8A"))
	tm.Feed([]byte("ne"))
	got := tm.String()
	if got[:len("Alpi")] != "Alpi" {
		t.Fatalf("ASCII prefix = %q", got)
	}
	want := "Alpi\u0644\u064a\u0646\u064a"
	if len(got) < len(want) || got[:len(want)] != want {
		t.Fatalf("decoded text = %q, want prefix %q", got, want)
	}
	if got[len(want):len(want)+2] != "ne" {
		t.Fatalf("trailing ASCII text = %q", got)
	}
}

func TestANSIFormattingAndCursor(t *testing.T) {
	tm := New(12, 2)
	tm.Feed([]byte("\x1b[31mred\x1b[0m\r\n\x1b[2;3Hgo"))
	runes := tm.Runes()
	if len(runes) != 24 {
		t.Fatalf("rune grid length = %d, want 24", len(runes))
	}
	if runes[0].R != 'r' || runes[0].Brush.FG.R != 205 {
		t.Fatalf("red cell = %#v", runes[0])
	}
	if runes[12+2].R != 'g' || runes[12+3].R != 'o' {
		t.Fatalf("cursor placement = %q", tm.String())
	}
}

func TestEraseAndScrollControlSequences(t *testing.T) {
	tm := New(5, 2)
	tm.Feed([]byte("12345\nabcde\x1b[2J"))
	if got := tm.String(); got != "     \n     \n" {
		t.Fatalf("erase display = %q", got)
	}
}
