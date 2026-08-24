package ui

import (
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func (s *Screen) sessionButtons(gtx C) D {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(s.sessionButton(gtx, &s.New, "New", s.newSession)),
		layout.Rigid(s.sessionButton(gtx, &s.Close, "Close", s.closeSession)),
		layout.Rigid(s.sessionButton(gtx, &s.Restart, "Restart", s.restartSession)),
	)
}

func (s *Screen) sessionButton(gtx C, click *widget.Clickable, text string, action func()) func(C) D {
	return func(gtx C) D {
		button := material.Button(s.Theme, click, text)
		button.TextSize = 11
		dims := button.Layout(gtx)
		for click.Clicked(gtx) {
			action()
		}
		return dims
	}
}

func (s *Screen) newSession() {
	if s.Sessions != nil {
		s.Sessions.NewSession()
	}
}
func (s *Screen) closeSession() {
	if s.Sessions != nil {
		s.Sessions.CloseSession()
	}
}
func (s *Screen) restartSession() {
	if s.Sessions != nil {
		s.Sessions.RestartSession()
	}
}
