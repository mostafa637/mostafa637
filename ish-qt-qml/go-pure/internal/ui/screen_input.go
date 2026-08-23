package ui

import (
	"io"

	"gioui.org/io/key"
	"gioui.org/io/transfer"
)

func (s *Screen) handleKey(e key.Event) {
	if s.Input == nil {
		return
	}
	if s.handleControlKey(e) {
		return
	}
	switch e.Name {
	case key.NameReturn, key.NameEnter:
		s.writeString("\r")
	case key.NameDeleteBackward:
		s.writeBytes([]byte{0x7f})
	case key.NameTab:
		s.writeString("\t")
	case key.NameEscape:
		s.writeBytes([]byte{0x1b})
	case key.NameUpArrow:
		s.writeString("\x1b[A")
	case key.NameDownArrow:
		s.writeString("\x1b[B")
	case key.NameLeftArrow:
		s.writeString("\x1b[D")
	case key.NameRightArrow:
		s.writeString("\x1b[C")
	default:
		s.writeRuneName(e)
	}
}

func (s *Screen) handleControlKey(e key.Event) bool {
	if e.Modifiers&key.ModCtrl == 0 {
		return false
	}
	name := string(e.Name)
	if len(name) != 1 {
		return false
	}
	r := name[0]
	if r >= 'a' && r <= 'z' {
		r -= 'a' - 1
	}
	if r >= 'A' && r <= 'Z' {
		r -= 'A' - 1
	}
	s.writeBytes([]byte{r})
	return true
}

func (s *Screen) writeRuneName(e key.Event) {
	if len(string(e.Name)) == 1 {
		s.writeString(string(e.Name))
	}
}

func (s *Screen) consumeClipboard(e transfer.DataEvent) {
	if e.Open == nil {
		return
	}
	data := e.Open()
	defer data.Close()
	buf, err := io.ReadAll(data)
	if err == nil {
		s.writeBytes(buf)
	}
}

func (s *Screen) writeString(value string) { s.writeBytes([]byte(value)) }

func (s *Screen) writeBytes(value []byte) {
	if len(value) == 0 || s.Input == nil {
		return
	}
	_ = s.Input.Write(value)
}
