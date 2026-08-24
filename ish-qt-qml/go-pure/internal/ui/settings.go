package ui

import (
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type settingsState struct {
	FontSize   int
	Cursor     bool
	Back       widget.Clickable
	FontDown   widget.Clickable
	FontUp     widget.Clickable
	CursorFlip widget.Clickable
}

var defaultSettings = settingsState{FontSize: 14, Cursor: true}

func (s *Screen) settings() *settingsState {
	if s.settingsState == nil {
		v := defaultSettings
		s.settingsState = &v
	}
	return s.settingsState
}

func (s *Screen) layoutSettings(gtx C) D {
	state := s.settings()
	if state.Back.Clicked(gtx) {
		s.SettingsOpen = false
	}
	if state.FontDown.Clicked(gtx) && state.FontSize > 8 {
		state.FontSize--
	}
	if state.FontUp.Clicked(gtx) && state.FontSize < 32 {
		state.FontSize++
	}
	if state.CursorFlip.Clicked(gtx) {
		state.Cursor = !state.Cursor
	}
	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(s.settingsTitle),
		layout.Rigid(func(gtx C) D { return s.settingsFont(gtx, state) }),
		layout.Rigid(func(gtx C) D { return s.settingsCursor(gtx, state) }),
		layout.Rigid(func(gtx C) D { return s.settingsBack(gtx, state) }))
}

func (s *Screen) settingsTitle(gtx C) D {
	return material.H6(s.Theme, "Settings").Layout(gtx)
}

func (s *Screen) settingsFont(gtx C, state *settingsState) D {
	label := material.Body1(s.Theme, "Terminal font size: "+itoa(state.FontSize))
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(material.Button(s.Theme, &state.FontDown, "−").Layout),
		layout.Rigid(label.Layout), layout.Rigid(material.Button(s.Theme, &state.FontUp, "+").Layout))
}

func (s *Screen) settingsCursor(gtx C, state *settingsState) D {
	value := "hidden"
	if state.Cursor {
		value = "visible"
	}
	return material.Button(s.Theme, &state.CursorFlip, "Cursor: "+value).Layout(gtx)
}

func (s *Screen) settingsBack(gtx C, state *settingsState) D {
	return material.Button(s.Theme, &state.Back, "Back to terminal").Layout(gtx)
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	out := make([]byte, 0, 3)
	for value > 0 {
		out = append([]byte{byte('0' + value%10)}, out...)
		value /= 10
	}
	return string(out)
}
