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

type managedUI struct {
	ctx context.Context
	win *gioapp.Window
	mgr *session.Manager
	scr *ui.Screen
	id  int
}

func RunManaged(ctx context.Context, factory session.Factory) error {
	win := new(gioapp.Window)
	win.Option(gioapp.Title("iSH Pure Go"), gioapp.Size(unit.Dp(390), unit.Dp(844)))
	mgr := session.NewManager(factory)
	id, sess, err := mgr.Create(ctx, 80, 24)
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}
	model := terminal.NewModel(80, 24)
	ctrl := &managedUI{ctx: ctx, win: win, mgr: mgr, id: id}
	ctrl.scr = ui.NewScreen(model, sess)
	ctrl.scr.Sessions = ctrl
	ctrl.attach(sess, model)
	return ctrl.loop()
}

func (m *managedUI) attach(sess session.Session, model *terminal.Model) {
	go func() {
		for chunk := range sess.Output() {
			model.Feed(chunk)
			m.win.Invalidate()
		}
		m.win.Invalidate()
	}()
}

func (m *managedUI) loop() error {
	var ops op.Ops
	for {
		e := m.win.Event()
		switch e := e.(type) {
		case gioapp.DestroyEvent:
			m.mgr.CloseAll()
			return e.Err
		case gioapp.FrameEvent:
			gtx := gioapp.NewContext(&ops, e)
			m.scr.Layout(gtx)
			e.Frame(gtx.Ops)
		}
		select {
		case <-m.ctx.Done():
			m.mgr.CloseAll()
			return m.ctx.Err()
		default:
		}
	}
}

func (m *managedUI) NewSession() {
	id, sess, err := m.mgr.Create(m.ctx, 80, 24)
	if err != nil {
		return
	}
	m.setActive(id, sess)
}

func (m *managedUI) CloseSession() { _ = m.mgr.Close(m.id) }

func (m *managedUI) RestartSession() {
	sess, err := m.mgr.Restart(m.ctx, m.id, 80, 24)
	if err == nil {
		m.setActive(m.id, sess)
	}
}

func (m *managedUI) setActive(id int, sess session.Session) {
	m.id = id
	m.scr.Input = sess
	m.scr.Terminal = terminal.NewModel(80, 24)
	m.attach(sess, m.scr.Terminal)
	m.win.Invalidate()
}
