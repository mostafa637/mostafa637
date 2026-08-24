package ui

import (
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mostafa637/mostafa637/go-pure/internal/terminal"
)

type C = layout.Context
type D = layout.Dimensions

type InputSink interface{ Write([]byte) error }
type ResizeSink interface{ Resize(cols, rows int) error }
type SessionActions interface {
	NewSession()
	CloseSession()
	RestartSession()
}

type Screen struct {
	Theme          *material.Theme
	Terminal       *terminal.Model
	Input          InputSink
	KeyTag         struct{}
	ClipboardTag   struct{}
	PasteRequested bool
	SettingsOpen   bool
	settingsState  *settingsState
	SettingsPath   string
	Focused        bool
	Sessions       SessionActions

	Tab, Ctrl, Esc, Arrows, Info, Paste, Hide widget.Clickable
	New, Close, Restart                       widget.Clickable
}

func NewScreen(model *terminal.Model, input InputSink) *Screen {
	return &Screen{Theme: material.NewTheme(), Terminal: model, Input: input}
}

func (s *Screen) Layout(gtx C) D {
	if s.SettingsOpen {
		return s.layoutSettings(gtx)
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(1, s.layoutTerminal), layout.Rigid(s.layoutAccessory))
}

func (s *Screen) layoutAccessory(gtx C) D {
	return s.accessoryBar(gtx)
}
