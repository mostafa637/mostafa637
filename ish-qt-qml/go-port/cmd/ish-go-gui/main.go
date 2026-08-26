package main

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gioui.org/app"
	giofont "gioui.org/font"
	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/mostafa637/ish-qt-qml/go-port/internal/rootfs"
	"github.com/mostafa637/ish-qt-qml/go-port/internal/session"
	"github.com/mostafa637/ish-qt-qml/go-port/internal/terminal"
	"github.com/viktomas/gritty/buffer"
)

//go:embed assets/root.tar.gz
var embeddedRootfs []byte

// This database is generated from the same rootfs archive on Linux. Android
// installs it as-is to avoid the modernc SQLite VFS lstat syscall rejected by
// Android x86_64 seccomp.
//
//go:embed assets/meta.db
var embeddedMetadata []byte

type C = layout.Context
type D = layout.Dimensions

type appState struct {
	theme *material.Theme
	input widget.Editor
	term  *terminal.Terminal

	buttons [7]widget.Clickable
	arrows  [4]widget.Clickable
	ops     op.Ops
	session *session.Session

	startTried    bool
	startDone     bool
	startCh       chan sessionStartResult
	focused       bool
	controlActive bool
	termCols      int
	termRows      int
}

type sessionStartResult struct {
	session *session.Session
	err     error
}

var (
	terminalBlack = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	// iSH follows the keyboard appearance. The reference UI is the dark
	// variant: charcoal accessory strip, translucent light key faces, and
	// white SF-Symbol-like glyphs.
	barBackground = color.NRGBA{R: 37, G: 37, B: 42, A: 255}
	keyBackground = color.NRGBA{R: 255, G: 255, B: 255, A: 55}
	keySecondary  = color.NRGBA{R: 147, G: 147, B: 147, A: 66}
	keyForeground = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	keyShadow     = color.NRGBA{R: 0, G: 0, B: 0, A: 105}
)

func main() {
	go func() {
		window := new(app.Window)
		window.Option(app.Title("iSH"), app.Size(unit.Dp(900), unit.Dp(600)))
		if err := run(window); err != nil {
			log.Print(err)
			os.Exit(1)
		}
	}()
	app.Main()
}

func run(w *app.Window) error {
	state := &appState{
		theme:   material.NewTheme(),
		term:    terminal.New(100, 30),
		startCh: make(chan sessionStartResult, 1),
	}
	state.input.SingleLine = false
	state.input.Submit = false
	state.input.InputHint = key.HintText
	state.input.LineHeight = unit.Sp(17)
	state.input.LineHeightScale = 1

	for {
		e := w.Event()
		switch e := e.(type) {
		case app.FrameEvent:
			gtx := app.NewContext(&state.ops, e)
			if !state.startTried {
				state.startTried = true
				// Rootfs extraction and CoreSession startup can take several seconds
				// on a fresh Android install. Never block Gio's frame/UI goroutine.
				go func() {
					started, err := startApplicationSession()
					state.startCh <- sessionStartResult{session: started, err: err}
				}()
			}
			if !state.startDone {
				select {
				case result := <-state.startCh:
					state.startDone = true
					if result.err != nil {
						state.term.Feed([]byte("\r\n[iSH] Alpine startup failed: " + result.err.Error() + "\r\n"))
					} else {
						state.session = result.session
						if state.termCols > 0 && state.termRows > 0 {
							_ = state.session.Resize(uint16(state.termCols), uint16(state.termRows))
						}
					}
				default:
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
	// Android x86_64 rejects the legacy lstat syscall used by os.Stat.
	if f, err := os.Open(marker); err == nil {
		_ = f.Close()
		if runtime.GOOS == "android" {
			if err := rootfs.ValidateBundled(base); err == nil {
				return base, nil
			}
		} else if err := rootfs.Validate(ctx, base); err == nil {
			return base, nil
		}
	}
	var installErr error
	if runtime.GOOS == "android" {
		installErr = rootfs.InstallBundled(ctx, bytes.NewReader(embeddedRootfs), bytes.NewReader(embeddedMetadata), base)
	} else {
		installErr = rootfs.Install(ctx, bytes.NewReader(embeddedRootfs), base)
	}
	if installErr != nil {
		return "", installErr
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
	if s.session == nil || e.State != key.Press {
		return
	}
	if e.Name == key.NameEscape {
		_ = s.session.Write([]byte("\x1b"))
	}
}

func (s *appState) layout(gtx C) {
	paint.Fill(gtx.Ops, terminalBlack)
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(1, func(gtx C) D {
			return s.terminalView(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			return s.accessory(gtx)
		}),
	)
	if !s.focused {
		gtx.Execute(key.FocusCmd{Tag: &s.input})
		s.focused = true
	}
}

func (s *appState) terminalView(gtx C) D {
	paint.Fill(gtx.Ops, terminalBlack)
	// iSH's terminal is the first responder; it does not show a separate
	// command-entry bar. Gio still needs an Editor to connect to Android IME,
	// so keep a full-surface transparent editor over the terminal surface.
	if editorEvent, changed := s.input.Update(gtx); changed {
		s.forwardEditorEvent(editorEvent)
	}
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			// term.css pins #terminal to all four edges with no body margin.
			return s.renderTerminal(gtx)
		}),
		layout.Stacked(func(gtx C) D {
			// Keep the editor full-size but visually transparent. Android IME
			// needs a real focus target; a 1x1 target is easily lost during
			// resize and does not behave like iSH's first-responder terminal.
			style := material.Editor(s.theme, &s.input, "")
			style.TextSize = unit.Sp(1)
			style.LineHeight = unit.Sp(1)
			style.Color = color.NRGBA{A: 0}
			style.HintColor = color.NRGBA{A: 0}
			style.SelectionColor = color.NRGBA{A: 0}
			return style.Layout(gtx)
		}),
	)
}

func (s *appState) forwardEditorEvent(ev widget.EditorEvent) {
	if s.session == nil {
		return
	}
	switch e := ev.(type) {
	case widget.ChangeEvent:
		// Editor.Text is the complete edit buffer. Clear it after every change so
		// the widget behaves like iSH's first responder, not a command box.
		value := s.input.Text()
		if value != "" {
			s.forwardTerminalText(value)
			s.input.SetText("")
		}
	case widget.SubmitEvent:
		if e.Text != "" {
			s.forwardTerminalText(e.Text)
		}
	}
}

func (s *appState) forwardTerminalText(value string) {
	if value == "" || s.session == nil {
		return
	}
	payload, nextControl := terminalInputBytes(value, s.controlActive)
	s.controlActive = nextControl
	if len(payload) > 0 || value == " " {
		_ = s.session.Write(payload)
	}
}

func terminalInputBytes(value string, controlActive bool) ([]byte, bool) {
	if value == "" {
		return nil, controlActive
	}
	if controlActive {
		runes := []rune(value)
		if len(runes) == 1 {
			if b, ok := controlByte(runes[0]); ok {
				return []byte{b}, false
			}
			// iSH consumes an unsupported one-rune Control insertion without
			// sending it to the terminal.
			return nil, false
		}
		// Like iSH, a multi-character IME insertion is sent as ordinary text;
		// the one-shot Control state is consumed by that insertion.
		controlActive = false
	}
	return []byte(strings.ReplaceAll(value, "\n", "\r")), controlActive
}

func controlByte(ch rune) (byte, bool) {
	validation := ch
	if validation >= 'A' && validation <= 'Z' {
		validation += 'a' - 'A'
	}
	if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyz@^26-=[]\\ ", validation) {
		return 0, false
	}
	if ch == ' ' {
		return 0, true
	}
	if ch == '2' {
		ch = '@'
	}
	if ch == '6' {
		ch = '^'
	}
	if ch >= 'a' && ch <= 'z' {
		ch -= 'a' - 'A'
	}
	return byte(ch) ^ 0x40, true
}

func (s *appState) renderTerminal(gtx C) D {
	const (
		cellWidthDp  = float32(9)
		cellHeightDp = float32(17)
	)
	widthDp := float32(gtx.Constraints.Max.X) / gtx.Metric.PxPerDp
	heightDp := float32(gtx.Constraints.Max.Y) / gtx.Metric.PxPerDp
	cols := maxInt(1, int(widthDp/cellWidthDp))
	rows := maxInt(1, int(heightDp/cellHeightDp))
	if cols != s.termCols || rows != s.termRows {
		s.termCols, s.termRows = cols, rows
		s.term.Resize(cols, rows)
		if s.session != nil {
			_ = s.session.Resize(uint16(cols), uint16(rows))
		}
	}

	runes := s.term.Runes()
	if len(runes) < cols*rows {
		runes = append(runes, make([]buffer.BrushedRune, cols*rows-len(runes))...)
	}
	if len(runes) > cols*rows {
		runes = runes[:cols*rows]
	}
	fontStyle := material.Label(s.theme, unit.Sp(15), "")
	fontStyle.Font.Typeface = "monospace"
	fontStyle.LineHeight = unit.Sp(17)
	fontStyle.LineHeightScale = 1

	cellWidth := maxInt(1, int(cellWidthDp*gtx.Metric.PxPerDp))
	cellHeight := maxInt(1, int(cellHeightDp*gtx.Metric.PxPerDp))
	for row := 0; row < rows; row++ {
		line := runes[row*cols : (row+1)*cols]
		for col := 0; col < cols; {
			start := col
			brush := line[col].Brush
			for col < cols && line[col].Brush == brush {
				col++
			}
			textValue := cellsText(line[start:col])
			if textValue == "" {
				continue
			}
			x := start * cellWidth
			y := row * cellHeight
			w := (col - start) * cellWidth
			push := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
			paint.FillShape(gtx.Ops, cellBackground(line[start]), clip.Rect{Max: image.Pt(w, cellHeight)}.Op())
			fontStyle.Text = textValue
			fontStyle.Color = cellForeground(line[start])
			if brush.Bold {
				fontStyle.Font.Weight = giofont.Bold
			} else {
				fontStyle.Font.Weight = giofont.Normal
			}
			local := gtx
			local.Constraints.Min = image.Point{}
			local.Constraints.Max = image.Pt(w, cellHeight)
			fontStyle.Layout(local)
			push.Pop()
		}
	}

	cursor := s.term.Cursor()
	if cursor.X >= 0 && cursor.X < cols && cursor.Y >= 0 && cursor.Y < rows {
		push := op.Offset(image.Pt(cursor.X*cellWidth, cursor.Y*cellHeight)).Push(gtx.Ops)
		paint.FillShape(gtx.Ops, color.NRGBA{R: 235, G: 235, B: 235, A: 145}, clip.Rect{Max: image.Pt(cellWidth, cellHeight)}.Op())
		push.Pop()
	}
	return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func cellsText(cells []buffer.BrushedRune) string {
	var b strings.Builder
	for _, cell := range cells {
		if cell.R == 0 {
			b.WriteByte(' ')
		} else {
			b.WriteRune(cell.R)
		}
	}
	return b.String()
}

func cellForeground(cell buffer.BrushedRune) color.NRGBA {
	if cell.Brush.Invert {
		return cellBackground(cell)
	}
	if cell.Brush.FG == buffer.DefaultFG {
		return color.NRGBA{R: 235, G: 235, B: 235, A: 255}
	}
	return ansiColor(cell.Brush.FG)
}

func cellBackground(cell buffer.BrushedRune) color.NRGBA {
	if cell.Brush.Invert {
		if cell.Brush.FG == buffer.DefaultFG {
			return color.NRGBA{R: 235, G: 235, B: 235, A: 255}
		}
		return ansiColor(cell.Brush.FG)
	}
	if cell.Brush.BG == buffer.DefaultBG {
		return terminalBlack
	}
	return ansiColor(cell.Brush.BG)
}

func ansiColor(c buffer.Color) color.NRGBA {
	return color.NRGBA{R: c.R, G: c.G, B: c.B, A: 255}
}

type barMetrics struct {
	button           float32
	horizontal       float32
	vertical         float32
	barHeight        float32
	gap              float32
	showHideKeyboard bool
}

func metricsForWidth(widthDp float32) barMetrics {
	// These values come from TerminalViewController.resizeBar. The Gio
	// implementation uses width as a practical proxy for phone/narrow-iPad/
	// wide-iPad traits, while keeping the original dimensions and 6pt stack gap.
	switch {
	case widthDp >= 600:
		return barMetrics{button: 43, horizontal: 15, vertical: 8, barHeight: 43, gap: 6, showHideKeyboard: false}
	case widthDp >= 430:
		return barMetrics{button: 36, horizontal: 10, vertical: 8, barHeight: 36, gap: 6, showHideKeyboard: false}
	default:
		return barMetrics{button: 32, horizontal: 6, vertical: 6, barHeight: 36, gap: 6, showHideKeyboard: true}
	}
}

func (s *appState) accessory(gtx C) D {
	paint.FillShape(gtx.Ops, barBackground, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}.Op())
	metrics := metricsForWidth(float32(gtx.Constraints.Max.X) / gtx.Metric.PxPerDp)
	button := metrics.button
	height := metrics.barHeight
	return layout.Inset{
		Top: unit.Dp(metrics.vertical), Bottom: unit.Dp(metrics.vertical),
		Left: unit.Dp(metrics.horizontal), Right: unit.Dp(metrics.horizontal),
	}.Layout(gtx, func(gtx C) D {
		gap := gtx.Dp(unit.Dp(metrics.gap))
		return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Flex{Axis: layout.Horizontal, Gap: gap}.Layout(gtx,
					layout.Rigid(func(gtx C) D { return s.accessoryButton(gtx, 0, button, height) }),
					layout.Rigid(func(gtx C) D { return s.accessoryButton(gtx, 1, button, height) }),
					layout.Rigid(func(gtx C) D { return s.accessoryButton(gtx, 2, button, height) }),
					layout.Rigid(func(gtx C) D { return s.arrowButton(gtx, button, height) }),
				)
			}),
			layout.Rigid(func(gtx C) D {
				children := []layout.FlexChild{
					layout.Rigid(func(gtx C) D { return s.accessoryButton(gtx, 5, button, height) }),
					layout.Rigid(func(gtx C) D { return s.accessoryButton(gtx, 4, button, height) }),
				}
				if metrics.showHideKeyboard {
					children = append(children, layout.Rigid(func(gtx C) D { return s.accessoryButton(gtx, 6, button, height) }))
				}
				return layout.Flex{Axis: layout.Horizontal, Gap: gap}.Layout(gtx, children...)
			}),
		)
	})
}

func (s *appState) accessoryButton(gtx C, index int, width, height float32) D {
	gtx.Constraints.Min.X = gtx.Dp(unit.Dp(width))
	gtx.Constraints.Max.X = gtx.Dp(unit.Dp(width))
	gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(height))
	gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(height))
	return s.buttons[index].Layout(gtx, func(gtx C) D {
		pressed := s.buttons[index].Pressed() || s.buttons[index].Hovered()
		if s.buttons[index].Clicked(gtx) {
			if index == 4 {
				gtx.Execute(clipboard.ReadCmd{Tag: &s.input})
			} else if index == 6 {
				gtx.Execute(key.SoftKeyboardCmd{Show: false})
			}
			s.sendAccessory(index)
		}
		face := clip.RRect{Rect: image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y), SE: 5, SW: 5, NE: 5, NW: 5}
		paint.FillShape(gtx.Ops, keyShadow, clip.RRect{Rect: image.Rect(0, 1, gtx.Constraints.Max.X, gtx.Constraints.Max.Y+1), SE: 5, SW: 5, NE: 5, NW: 5}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, keyBackground, face.Op(gtx.Ops))
		if pressed || (index == 1 && s.controlActive) {
			paint.FillShape(gtx.Ops, keySecondary, clip.RRect{Rect: image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y), SE: 5, SW: 5, NE: 5, NW: 5}.Op(gtx.Ops))
		}
		s.drawBarIcon(gtx, index)
		return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}
	})
}

func (s *appState) arrowButton(gtx C, width, height float32) D {
	gtx.Constraints.Min.X = gtx.Dp(unit.Dp(width))
	gtx.Constraints.Max.X = gtx.Dp(unit.Dp(width))
	gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(height))
	gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(height))
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			face := clip.RRect{Rect: image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y), SE: 5, SW: 5, NE: 5, NW: 5}
			paint.FillShape(gtx.Ops, keyShadow, clip.RRect{Rect: image.Rect(0, 1, gtx.Constraints.Max.X, gtx.Constraints.Max.Y+1), SE: 5, SW: 5, NE: 5, NW: 5}.Op(gtx.Ops))
			paint.FillShape(gtx.Ops, keyBackground, face.Op(gtx.Ops))
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}
		}),
		layout.Stacked(func(gtx C) D {
			width := gtx.Constraints.Max.X
			height := gtx.Constraints.Max.Y
			col := width / 3
			row := height / 3
			colWidth := func(index int) int {
				if index == 2 {
					return width - 2*col
				}
				return maxInt(1, col)
			}
			rowHeight := func(index int) int {
				if index == 2 {
					return height - 2*row
				}
				return maxInt(1, row)
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx C) D { return arrowBlank(gtx, colWidth(0), rowHeight(0)) }),
						layout.Rigid(func(gtx C) D { return s.arrowPart(gtx, 0, colWidth(1), rowHeight(0)) }),
						layout.Rigid(func(gtx C) D { return arrowBlank(gtx, colWidth(2), rowHeight(0)) }),
					)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx C) D { return s.arrowPart(gtx, 2, colWidth(0), rowHeight(1)) }),
						layout.Rigid(func(gtx C) D { return arrowBlank(gtx, colWidth(1), rowHeight(1)) }),
						layout.Rigid(func(gtx C) D { return s.arrowPart(gtx, 3, colWidth(2), rowHeight(1)) }),
					)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx C) D { return arrowBlank(gtx, colWidth(0), rowHeight(2)) }),
						layout.Rigid(func(gtx C) D { return s.arrowPart(gtx, 1, colWidth(1), rowHeight(2)) }),
						layout.Rigid(func(gtx C) D { return arrowBlank(gtx, colWidth(2), rowHeight(2)) }),
					)
				}),
			)
		}),
	)
}

func arrowBlank(gtx C, width, height int) D {
	gtx.Constraints.Min = image.Pt(width, height)
	gtx.Constraints.Max = image.Pt(width, height)
	return layout.Dimensions{Size: image.Pt(width, height)}
}

func (s *appState) arrowPart(gtx C, index int, width, height int) D {
	gtx.Constraints.Min = image.Pt(width, height)
	gtx.Constraints.Max = image.Pt(width, height)
	return s.arrows[index].Layout(gtx, func(gtx C) D {
		if s.arrows[index].Clicked(gtx) {
			s.sendArrow(index)
		}
		if s.arrows[index].Pressed() || s.arrows[index].Hovered() {
			paint.FillShape(gtx.Ops, keySecondary, clip.Rect{Max: image.Pt(width, height)}.Op())
		}
		glyphs := []string{"↑", "↓", "←", "→"}
		label := material.Label(s.theme, unit.Sp(10), glyphs[index])
		label.Color = keyForeground
		label.Alignment = text.Middle
		return layout.Center.Layout(gtx, label.Layout)
	})
}

func (s *appState) drawBarIcon(gtx C, index int) {
	w, h := gtx.Constraints.Max.X, gtx.Constraints.Max.Y
	icon := keyForeground
	centerX, centerY := w/2, h/2
	switch index {
	case 0: // arrow.right.to.line.compact / Tab
		paint.FillShape(gtx.Ops, icon, clip.RRect{Rect: image.Rect(centerX-10, centerY-1, centerX+7, centerY+2), SE: 1, SW: 1, NE: 1, NW: 1}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, icon, clip.RRect{Rect: image.Rect(centerX+5, centerY-8, centerX+8, centerY+9), SE: 1, SW: 1, NE: 1, NW: 1}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, icon, clip.RRect{Rect: image.Rect(centerX-3, centerY-6, centerX+1, centerY-3), SE: 1, SW: 1, NE: 1, NW: 1}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, icon, clip.RRect{Rect: image.Rect(centerX-1, centerY-4, centerX+4, centerY-1), SE: 1, SW: 1, NE: 1, NW: 1}.Op(gtx.Ops))
	case 1: // control, rendered as a deterministic chevron
		for _, tooth := range []image.Rectangle{
			image.Rect(centerX-9, centerY, centerX-6, centerY+3), image.Rect(centerX-7, centerY-3, centerX-4, centerY), image.Rect(centerX-5, centerY-6, centerX-2, centerY-3),
			image.Rect(centerX+2, centerY-6, centerX+5, centerY-3), image.Rect(centerX+4, centerY-3, centerX+7, centerY), image.Rect(centerX+6, centerY, centerX+9, centerY+3),
		} {
			paint.FillShape(gtx.Ops, icon, clip.RRect{Rect: tooth, SE: 1, SW: 1, NE: 1, NW: 1}.Op(gtx.Ops))
		}
	case 2: // escape
		label := material.Label(s.theme, unit.Sp(11), "ESC")
		label.Color = icon
		label.Alignment = text.Middle
		layout.Center.Layout(gtx, label.Layout)
	case 4: // doc.on.clipboard
		paint.FillShape(gtx.Ops, icon, clip.RRect{Rect: image.Rect(centerX-7, centerY-6, centerX+7, centerY+8), SE: 2, SW: 2, NE: 2, NW: 2}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, keyBackground, clip.RRect{Rect: image.Rect(centerX-4, centerY-3, centerX+5, centerY+5), SE: 1, SW: 1, NE: 1, NW: 1}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, icon, clip.RRect{Rect: image.Rect(centerX-3, centerY-9, centerX+4, centerY-5), SE: 1, SW: 1, NE: 1, NW: 1}.Op(gtx.Ops))
	case 5: // gear
		for _, tooth := range []image.Rectangle{
			image.Rect(centerX-2, centerY-10, centerX+2, centerY-4), image.Rect(centerX-2, centerY+4, centerX+2, centerY+10),
			image.Rect(centerX-10, centerY-2, centerX-4, centerY+2), image.Rect(centerX+4, centerY-2, centerX+10, centerY+2),
		} {
			paint.FillShape(gtx.Ops, icon, clip.RRect{Rect: tooth, SE: 1, SW: 1, NE: 1, NW: 1}.Op(gtx.Ops))
		}
		paint.FillShape(gtx.Ops, icon, clip.Ellipse(image.Rect(centerX-7, centerY-7, centerX+7, centerY+7)).Op(gtx.Ops))
		paint.FillShape(gtx.Ops, keyBackground, clip.Ellipse(image.Rect(centerX-3, centerY-3, centerX+3, centerY+3)).Op(gtx.Ops))
	case 6: // keyboard.chevron.compact.down
		paint.FillShape(gtx.Ops, icon, clip.RRect{Rect: image.Rect(centerX-11, centerY-7, centerX+11, centerY+4), SE: 2, SW: 2, NE: 2, NW: 2}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, keyBackground, clip.RRect{Rect: image.Rect(centerX-7, centerY-4, centerX+7, centerY-2), SE: 1, SW: 1, NE: 1, NW: 1}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, icon, clip.RRect{Rect: image.Rect(centerX-5, centerY+5, centerX-2, centerY+9), SE: 1, SW: 1, NE: 1, NW: 1}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, icon, clip.RRect{Rect: image.Rect(centerX+2, centerY+5, centerX+5, centerY+9), SE: 1, SW: 1, NE: 1, NW: 1}.Op(gtx.Ops))
	}
}

func (s *appState) sendArrow(index int) {
	if s.session == nil {
		return
	}
	seq := [][]byte{[]byte("\x1b[A"), []byte("\x1b[B"), []byte("\x1b[D"), []byte("\x1b[C")}
	if index >= 0 && index < len(seq) {
		_ = s.session.Write(seq[index])
	}
}

func (s *appState) sendAccessory(index int) {
	if s.session == nil {
		return
	}
	var seq []byte
	switch index {
	case 0:
		seq = []byte("\t")
	case 1:
		s.controlActive = !s.controlActive
		return
	case 2:
		seq = []byte("\x1b")
	case 4:
		// Paste is requested from accessoryButton through Gio's clipboard router.
		return
	case 5:
		// The gear surface is intentionally not faked; the terminal remains
		// usable while the full iSH settings/about controller is ported.
		return
	case 6:
		// SoftKeyboardCmd above performs the platform action.
		return
	}
	if len(seq) > 0 {
		_ = s.session.Write(seq)
	}
}
