package app

import (
	"context"
	"fmt"

	gioapp "gioui.org/app"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/mostafa637/mostafa637/go-pure/internal/session"
	"github.com/mostafa637/mostafa637/go-pure/internal/terminal"
	"github.com/mostafa637/mostafa637/go-pure/internal/ui"
)

func Run(ctx context.Context, sess session.Session) error {
	window := new(gioapp.Window)
	window.Option(gioapp.Title("iSH Pure Go"), gioapp.Size(unit.Dp(390), unit.Dp(844)))

	model := terminal.NewModel(80, 24)
	screen := ui.NewScreen(model, sess)
	if err := sess.Start(ctx, 80, 24); err != nil {
		return fmt.Errorf("start session: %w", err)
	}
	defer sess.Close()

	go func() {
		for chunk := range sess.Output() {
			model.Feed(chunk)
			window.Invalidate()
		}
		window.Invalidate()
	}()

	var ops op.Ops
	for {
		e := window.Event()
		switch e := e.(type) {
		case gioapp.DestroyEvent:
			return e.Err
		case gioapp.FrameEvent:
			gtx := gioapp.NewContext(&ops, e)
			screen.Layout(gtx)
			e.Frame(gtx.Ops)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
}
