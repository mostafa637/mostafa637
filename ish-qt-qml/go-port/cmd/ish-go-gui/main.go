package main

import (
	"context"
	"image/color"
	"log"
	"os"
	"strings"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/mostafa637/ish-qt-qml/go-port/internal/session"
)

type C = layout.Context
type D = layout.Dimensions

type appState struct {
	theme  *material.Theme
	input  widget.Editor
	output string

	buttons    [7]widget.Clickable
	buttonText [7]string
	ops        op.Ops
	session    *session.Session
}

func main() {
	go func() {
		window := new(app.Window)
		window.Option(app.Title("iSH Go"), app.Size(unit.Dp(900), unit.Dp(600)))
		if err := run(window); err != nil {
			log.Print(err)
			os.Exit(1)
		}
	}()
	app.Main()
}

func run(w *app.Window) error {
	state := &appState{
		theme:      material.NewTheme(),
		buttonText: [7]string{"ESC", "CTRL", "ALT", "TAB", "↑↓←→", "粘贴", "⌫"},
	}
	state.input.SingleLine = true
	state.input.Submit = true
	state.input.InputHint = key.HintText

	shell := os.Getenv("ISH_SHELL")
	if shell == "" {
		shell = "/system/bin/sh"
		if _, statErr := os.Stat(shell); statErr != nil {
			shell = "/bin/sh"
		}
	}
	started, err := session.Start(context.Background(), shell)
	if err != nil {
		return err
	}
	state.session = started
	defer state.session.Close()

	for {
		e := w.Event()
		switch e := e.(type) {
		case app.FrameEvent:
			gtx := app.NewContext(&state.ops, e)

			state.drainOutput()
			state.layout(gtx)
			e.Frame(&state.ops)
			w.Invalidate()
		case app.DestroyEvent:
			return e.Err
		case key.Event:
			state.handleKey(e)
		}
	}
}

func (s *appState) drainOutput() {
	for {
		select {
		case chunk, ok := <-s.session.Output():
			if !ok {
				return
			}
			s.output += string(chunk)
			if len(s.output) > 256*1024 {
				s.output = s.output[len(s.output)-256*1024:]
			}
		default:
			return
		}
	}
}

func (s *appState) handleKey(e key.Event) {
	if e.State != key.Press || e.Name != key.NameEscape {
		return
	}
	_ = s.session.Write([]byte("\x1b"))
}

func (s *appState) layout(gtx C) {
	paint.Fill(gtx.Ops, color.NRGBA{R: 20, G: 20, B: 22, A: 255})
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(1, func(gtx C) D {
			return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx C) D {
				label := material.Label(s.theme, unit.Sp(14), s.output)
				label.Color = color.NRGBA{R: 236, G: 236, B: 240, A: 255}
				label.LineHeightScale = 1.15
				return label.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx C) D {
			return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx C) D {
				style := material.Editor(s.theme, &s.input, "command")
				style.TextSize = unit.Sp(15)
				style.Color = color.NRGBA{R: 236, G: 236, B: 240, A: 255}
				style.HintColor = color.NRGBA{R: 150, G: 150, B: 156, A: 255}
				style.SelectionColor = color.NRGBA{R: 72, G: 86, B: 160, A: 255}

				if event, changed := s.input.Update(gtx); changed {
					if submit, ok := event.(widget.SubmitEvent); ok {
						_ = s.session.Write([]byte(submit.Text + "\n"))
						s.input.SetText("")
					}
				}
				return style.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx C) D { return s.accessory(gtx) }),
	)
}

func (s *appState) accessory(gtx C) D {
	return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEvenly}.Layout(gtx,
			s.button(gtx, 0),
			s.button(gtx, 1),
			s.button(gtx, 2),
			s.button(gtx, 3),
			s.button(gtx, 4),
			s.button(gtx, 5),
			s.button(gtx, 6),
		)
	})
}

func (s *appState) button(gtx C, index int) layout.FlexChild {
	return layout.Rigid(func(gtx C) D {
		style := material.Button(s.theme, &s.buttons[index], s.buttonText[index])
		style.TextSize = unit.Sp(12)
		if s.buttons[index].Clicked(gtx) {
			s.sendAccessory(index)
		}
		return style.Layout(gtx)
	})
}

func (s *appState) sendAccessory(index int) {
	seq := []byte{}
	switch index {
	case 0:
		seq = []byte("\x1b")
	case 1:
		// The next typed rune is handled by the terminal editor/session in a
		// future modifier model; keep a visible state marker for now.
		seq = []byte("\x1b[27;5u")
	case 2:
		seq = []byte("\x1b")
	case 3:
		seq = []byte("\t")
	case 4:
		seq = []byte("\x1b[A")
	case 5:
		seq = []byte{}
	case 6:
		seq = []byte("\x7f")
	}
	if len(seq) > 0 {
		_ = s.session.Write(seq)
	}
	if index == 5 {
		// Gio clipboard integration is platform-specific; leave the button
		// operational without inventing a clipboard payload.
		s.output = strings.TrimSuffix(s.output, "\n") + "\n[paste requested]\n"
	}
}

var _ = event.Event(nil)
