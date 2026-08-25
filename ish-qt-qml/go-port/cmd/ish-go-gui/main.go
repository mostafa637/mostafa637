package main

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/mostafa637/ish-qt-qml/go-port/internal/rootfs"
	"github.com/mostafa637/ish-qt-qml/go-port/internal/session"
	"github.com/mostafa637/ish-qt-qml/go-port/internal/terminal"
)

//go:embed assets/root.tar.gz
var embeddedRootfs []byte

type C = layout.Context
type D = layout.Dimensions

type appState struct {
	theme *material.Theme
	input widget.Editor
	term  *terminal.Terminal

	buttons    [7]widget.Clickable
	buttonText [7]string
	ops        op.Ops
	session    *session.Session
	startTried bool
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
		term:       terminal.New(100, 30),
	}
	state.input.SingleLine = true
	state.input.Submit = true
	state.input.InputHint = key.HintText

	for {
		e := w.Event()
		switch e := e.(type) {
		case app.FrameEvent:
			gtx := app.NewContext(&state.ops, e)
			if !state.startTried {
				state.startTried = true
				started, err := startApplicationSession()
				if err != nil {
					state.term.Feed([]byte("\r\n[Alpine startup failed] " + err.Error() + "\r\n"))
				} else {
					state.session = started
					_ = state.session.Resize(100, 30)
				}
			}
			state.drainOutput()
			state.layout(gtx)
			e.Frame(&state.ops)
			w.Invalidate()
		case app.DestroyEvent:
			if state.session != nil {
				_ = state.session.Close()
			}
			return e.Err
		case key.Event:
			state.handleKey(e)
		}
	}
}

func startApplicationSession() (*session.Session, error) {
	if session.NativeCoreAvailable() {
		base, err := prepareRootfs(context.Background())
		if err != nil {
			return nil, err
		}
		return session.StartAlpine(context.Background(), base)
	}
	if runtime.GOOS == "android" {
		return nil, errors.New("APK was built without the native iSH/Asbestos core")
	}
	shell := os.Getenv("ISH_SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return session.Start(context.Background(), shell, "-i")
}

func prepareRootfs(ctx context.Context) (string, error) {
	if override := os.Getenv("ISH_ROOTFS_BASE"); override != "" {
		if err := rootfs.Validate(ctx, override); err != nil {
			return "", err
		}
		return override, nil
	}
	dataDir, err := app.DataDir()
	if err != nil {
		return "", errors.New("cannot locate application data directory: " + err.Error())
	}
	base := filepath.Join(dataDir, "ish-rootfs")
	marker := filepath.Join(base, ".installed-v1")
	if _, err := os.Stat(marker); err == nil {
		if err := rootfs.Validate(ctx, base); err == nil {
			return base, nil
		}
	}
	if err := rootfs.Install(ctx, bytes.NewReader(embeddedRootfs), base); err != nil {
		return "", err
	}
	if err := os.WriteFile(marker, []byte("go-gio-alpine-v1\n"), 0o600); err != nil {
		return "", errors.New("write rootfs marker: " + err.Error())
	}
	return base, nil
}

func (s *appState) drainOutput() {
	if s.session == nil {
		return
	}
	for {
		select {
		case chunk, ok := <-s.session.Output():
			if !ok {
				return
			}
			s.term.Feed(chunk)
		default:
			return
		}
	}
}

func (s *appState) handleKey(e key.Event) {
	if s.session == nil || e.State != key.Press || e.Name != key.NameEscape {
		return
	}
	_ = s.session.Write([]byte("\x1b"))
}

func (s *appState) layout(gtx C) {
	paint.Fill(gtx.Ops, color.NRGBA{R: 20, G: 20, B: 22, A: 255})
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(1, func(gtx C) D {
			return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx C) D {
				label := material.Label(s.theme, unit.Sp(14), s.term.String())
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

				if editorEvent, changed := s.input.Update(gtx); changed {
					if submit, ok := editorEvent.(widget.SubmitEvent); ok && s.session != nil {
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
	if s.session == nil {
		return
	}
	seq := []byte{}
	switch index {
	case 0:
		seq = []byte("\x1b")
	case 1:
		seq = []byte("\x1b[27;5u")
	case 2:
		seq = []byte("\x1b")
	case 3:
		seq = []byte("\t")
	case 4:
		seq = []byte("\x1b[A")
	case 6:
		seq = []byte("\x7f")
	}
	if len(seq) > 0 {
		_ = s.session.Write(seq)
	}
	if index == 5 {
		s.term.Feed([]byte("\n[paste requested]\n"))
	}
}

var _ = event.Event(nil)
