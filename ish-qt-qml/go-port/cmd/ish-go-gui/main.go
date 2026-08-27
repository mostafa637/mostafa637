package main

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	giofont "gioui.org/font"
	"gioui.org/font/opentype"
	"gioui.org/gesture"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
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

// These are the actual template assets from iSH's Assets.xcassets. The PDFs
// remain beside the PNGs as provenance; the PNGs are rasterized from those
// PDFs at high resolution and keep their original alpha contours.
//
//go:embed assets/icons/paste-original.png
var pasteOriginalPNG []byte

//go:embed assets/icons/hide-keyboard-original.png
var hideKeyboardOriginalPNG []byte

// DejaVu Sans contains the exact Unicode characters used as titles by the
// original storyboard (⇥, ⌃, ⎋, and the arrows). Bundling it prevents Android
// font fallback from silently dropping those glyphs.
//
//go:embed assets/fonts/DejaVuSans.ttf
var embeddedSymbolFont []byte

var (
	pasteOriginalIcon        = decodeEmbeddedIcon(pasteOriginalPNG)
	hideKeyboardOriginalIcon = decodeEmbeddedIcon(hideKeyboardOriginalPNG)
)

func decodeEmbeddedIcon(data []byte) image.Image {
	icon, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		panic("decode embedded iSH icon: " + err.Error())
	}
	return icon
}

func newThemeWithOriginalGlyphFont() *material.Theme {
	theme := material.NewTheme()
	faces, err := opentype.ParseCollection(embeddedSymbolFont)
	if err != nil {
		log.Printf("iSH original glyph font unavailable: %v", err)
		return theme
	}
	theme.Shaper = text.NewShaper(text.WithCollection(faces))
	return theme
}

type C = layout.Context
type D = layout.Dimensions

const arrowNone = -1

type pageID uint8

const (
	pageTerminal pageID = iota
	pageSettings
	pageAppearance
	pageExternalKeyboard
	pageFilesystems
	pageFilesBrowser
	pageAbout
)

type appState struct {
	theme *material.Theme
	input widget.Editor
	term  *terminal.Terminal

	buttons [7]widget.Clickable
	ops     op.Ops
	session *session.Session

	arrowDrag       gesture.Drag
	arrowStart      f32.Point
	arrowDirection  int
	arrowPressed    bool
	arrowNextRepeat time.Time

	page           pageID
	prefs          userPreferences
	prefsPath      string
	prefsLoaded    bool
	backButton     widget.Clickable
	settingsRows   [5]widget.Clickable
	appearanceRows [7]widget.Clickable
	externalRows   [4]widget.Clickable
	filesRows      [2]widget.Clickable
	aboutRows      [5]widget.Clickable
	settingsList   widget.List
	appearanceList widget.List
	externalList   widget.List
	filesList      widget.List
	browserList    widget.List
	aboutList      widget.List
	keepScreenOn   widget.Bool
	blinkCursor    widget.Bool
	hideStatusBar  widget.Bool
	rootfsBase     string
	fileEntries    []string

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
		theme:          newThemeWithOriginalGlyphFont(),
		term:           terminal.New(100, 30),
		startCh:        make(chan sessionStartResult, 1),
		arrowDirection: arrowNone,
		prefs:          defaultUserPreferences(),
	}
	state.input.SingleLine = false
	state.input.Submit = false
	state.input.InputHint = key.HintText
	state.input.LineHeight = unit.Sp(17)
	state.input.LineHeightScale = 1
	state.settingsList.List.Axis = layout.Vertical
	state.appearanceList.List.Axis = layout.Vertical
	state.externalList.List.Axis = layout.Vertical
	state.filesList.List.Axis = layout.Vertical
	state.browserList.List.Axis = layout.Vertical
	state.aboutList.List.Axis = layout.Vertical
	state.keepScreenOn.Value = false
	state.blinkCursor.Value = state.prefs.BlinkCursor
	state.hideStatusBar.Value = state.prefs.HideStatusBar

	for {
		e := w.Event()
		switch e := e.(type) {
		case app.FrameEvent:
			gtx := app.NewContext(&state.ops, e)
			if !state.prefsLoaded {
				state.prefsPath = preferenceFilePath()
				prefs, err := loadUserPreferences(state.prefsPath)
				if err != nil {
					log.Printf("iSH preferences load failed: %v", err)
					prefs = defaultUserPreferences()
				}
				state.prefs = prefs
				state.blinkCursor.Value = state.prefs.BlinkCursor
				state.hideStatusBar.Value = state.prefs.HideStatusBar
				state.prefsLoaded = true
				if err := saveUserPreferences(state.prefsPath, state.prefs); err != nil {
					log.Printf("iSH preferences initial save failed: %v", err)
				}
			}
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
						state.rootfsBase = rootfsDisplayPath()
						if state.termCols > 0 && state.termRows > 0 {
							_ = state.session.Resize(uint16(state.termCols), uint16(state.termRows))
						}
					}
				default:
				}
			}
			state.drainOutput()
			state.layout(gtx)
			state.syncPreferences()
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

func rootfsDisplayPath() string {
	if override := os.Getenv("ISH_ROOTFS_BASE"); override != "" {
		return override
	}
	dataDir, err := app.DataDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dataDir, "ish-rootfs")
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
	if e.State != key.Press {
		return
	}
	if e.Name == key.NameEscape && s.page != pageTerminal {
		s.goBack()
		return
	}
	if s.session == nil {
		return
	}
	if payload, handled := terminalKeyEventBytes(e, s.prefs); handled {
		if err := s.session.Write(payload); err != nil {
			log.Printf("iSH GUI hardware key failed: %v", err)
		}
	}
}

func (s *appState) layout(gtx C) {
	if s.page != pageTerminal {
		s.pageView(gtx)
		return
	}
	paint.Fill(gtx.Ops, s.terminalPalette().background)
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

func (s *appState) pageView(gtx C) {
	pageBackground := color.NRGBA{R: 242, G: 242, B: 247, A: 255}
	paint.Fill(gtx.Ops, pageBackground)
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D { return s.pageToolbar(gtx) }),
		layout.Flexed(1, func(gtx C) D {
			switch s.page {
			case pageSettings:
				return s.settingsPage(gtx)
			case pageAppearance:
				return s.appearancePage(gtx)
			case pageExternalKeyboard:
				return s.externalKeyboardPage(gtx)
			case pageFilesystems:
				return s.filesystemsPage(gtx)
			case pageFilesBrowser:
				return s.filesBrowserPage(gtx)
			case pageAbout:
				return s.aboutPage(gtx)
			default:
				return layout.Dimensions{}
			}
		}),
	)
}

func (s *appState) pageToolbar(gtx C) D {
	height := gtx.Dp(unit.Dp(56))
	gtx.Constraints.Min = image.Pt(gtx.Constraints.Max.X, height)
	gtx.Constraints.Max.Y = height
	paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 245}, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, height)}.Op())
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min = image.Pt(gtx.Dp(unit.Dp(88)), height)
			gtx.Constraints.Max = gtx.Constraints.Min
			clicked := s.backButton.Clicked(gtx)
			return s.backButton.Layout(gtx, func(gtx C) D {
				if clicked {
					s.goBack()
				}
				label := material.Label(s.theme, unit.Sp(17), "‹  Back")
				label.Color = color.NRGBA{R: 0, G: 122, B: 255, A: 255}
				return layout.Center.Layout(gtx, label.Layout)
			})
		}),
		layout.Flexed(1, func(gtx C) D {
			label := material.Label(s.theme, unit.Sp(17), s.pageTitle())
			label.Alignment = text.Middle
			label.Font.Weight = giofont.Bold
			label.Color = color.NRGBA{R: 20, G: 20, B: 24, A: 255}
			return label.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D { return layout.Dimensions{Size: image.Pt(gtx.Dp(unit.Dp(88)), height)} }),
	)
}

func (s *appState) pageTitle() string {
	switch s.page {
	case pageSettings:
		return "Settings"
	case pageAppearance:
		return "Appearance"
	case pageExternalKeyboard:
		return "External Keyboard"
	case pageFilesystems:
		return "Filesystems"
	case pageFilesBrowser:
		return "Browse Files"
	case pageAbout:
		return "About iSH"
	default:
		return "iSH"
	}
}

func (s *appState) goBack() {
	from := s.pageTitle()
	switch s.page {
	case pageAppearance, pageExternalKeyboard, pageFilesystems, pageAbout:
		s.page = pageSettings
	case pageFilesBrowser:
		s.page = pageFilesystems
	default:
		s.page = pageTerminal
	}
	log.Printf("iSH GUI page navigation: %s -> %s", from, s.pageTitle())
}

func pageLabel(theme *material.Theme, size unit.Sp, value string, col color.NRGBA) layout.Widget {
	return func(gtx C) D {
		label := material.Label(theme, size, value)
		label.Color = col
		return label.Layout(gtx)
	}
}

func (s *appState) pageRow(gtx C, button *widget.Clickable, title, detail string, action func()) D {
	height := gtx.Dp(unit.Dp(52))
	gtx.Constraints.Min = image.Pt(gtx.Constraints.Max.X, height)
	gtx.Constraints.Max.Y = height
	clicked := button.Clicked(gtx)
	return button.Layout(gtx, func(gtx C) D {
		if clicked && action != nil {
			action()
		}
		bg := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		if button.Pressed() || button.Hovered() {
			bg = color.NRGBA{R: 232, G: 232, B: 237, A: 255}
		}
		paint.FillShape(gtx.Ops, bg, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, height)}.Op())
		return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical, Alignment: layout.Start}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return pageLabel(s.theme, unit.Sp(17), title, color.NRGBA{R: 28, G: 28, B: 32, A: 255})(gtx)
						}),
						layout.Rigid(func(gtx C) D {
							if detail == "" {
								return layout.Dimensions{}
							}
							return pageLabel(s.theme, unit.Sp(13), detail, color.NRGBA{R: 110, G: 110, B: 116, A: 255})(gtx)
						}),
					)
				}),
				layout.Rigid(func(gtx C) D {
					chevron := material.Label(s.theme, unit.Sp(25), "›")
					chevron.Color = color.NRGBA{R: 142, G: 142, B: 147, A: 255}
					return chevron.Layout(gtx)
				}),
			)
		})
	})
}

func (s *appState) sectionGap(gtx C) D {
	return layout.Spacer{Height: unit.Dp(18)}.Layout(gtx)
}

func (s *appState) settingsPage(gtx C) D {
	return s.settingsList.Layout(gtx, 5, func(gtx C, index int) D {
		if index == 0 || index == 4 {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D { return s.sectionGap(gtx) }),
				layout.Rigid(func(gtx C) D { return s.settingsRow(gtx, index) }),
			)
		}
		return s.settingsRow(gtx, index)
	})
}

func (s *appState) settingsRow(gtx C, index int) D {
	titles := []string{"Appearance", "External Keyboard", "Filesystems", "Upgrade Repositories", "About"}
	details := []string{"Theme, font, cursor and color scheme", "Caps Lock and modifier mappings", "Manage the Alpine filesystems", "Refresh available package repositories", "Version, links and diagnostics"}
	return s.pageRow(gtx, &s.settingsRows[index], titles[index], details[index], func() {
		switch index {
		case 0:
			s.page = pageAppearance
		case 1:
			s.page = pageExternalKeyboard
		case 2:
			s.page = pageFilesystems
		case 4:
			s.page = pageAbout
		}
	})
}

func (s *appState) appearancePage(gtx C) D {
	return s.appearanceList.Layout(gtx, 7, func(gtx C, index int) D {
		if index == 0 || index == 3 {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D { return s.sectionGap(gtx) }),
				layout.Rigid(func(gtx C) D { return s.appearanceRow(gtx, index) }),
			)
		}
		return s.appearanceRow(gtx, index)
	})
}

func (s *appState) appearanceRow(gtx C, index int) D {
	titles := []string{"Theme", "Font", "Font Size", "Color Scheme", "Cursor Style", "Blink Cursor", "Hide Status Bar"}
	details := []string{
		s.prefs.Theme,
		s.prefs.FontFamily,
		fmt.Sprintf("%.0f pt", s.prefs.FontSize),
		colorSchemeName(s.prefs.ColorScheme),
		cursorStyleName(s.prefs.CursorStyle),
		"",
		"",
	}
	if index == 5 {
		return s.switchRow(gtx, &s.blinkCursor, titles[index])
	}
	if index == 6 {
		return s.switchRow(gtx, &s.hideStatusBar, titles[index])
	}
	var action func()
	switch index {
	case 0:
		action = func() {
			s.updatePreferences(func(p *userPreferences) {
				switch p.Theme {
				case "Default":
					p.Theme = "1337"
				case "1337":
					p.Theme = "Solarized"
				case "Solarized":
					p.Theme = "Hot Dog Stand"
				default:
					p.Theme = "Default"
				}
			})
		}
	case 1:
		action = func() {
			s.updatePreferences(func(p *userPreferences) {
				if p.FontFamily == "monospace" {
					p.FontFamily = "DejaVu Sans Mono"
				} else {
					p.FontFamily = "monospace"
				}
			})
		}
	case 2:
		action = func() {
			s.updatePreferences(func(p *userPreferences) { p.FontSize = nextFontSize(p.FontSize) })
		}
	case 3:
		action = func() {
			s.updatePreferences(func(p *userPreferences) {
				p.ColorScheme = (p.ColorScheme + 1) % 3
			})
		}
	case 4:
		action = func() {
			s.updatePreferences(func(p *userPreferences) {
				p.CursorStyle = (p.CursorStyle + 1) % 3
			})
		}
	}
	return s.pageRow(gtx, &s.appearanceRows[index], titles[index], details[index], action)
}

func (s *appState) switchRow(gtx C, value *widget.Bool, title string) D {
	height := gtx.Dp(unit.Dp(52))
	gtx.Constraints.Min = image.Pt(gtx.Constraints.Max.X, height)
	gtx.Constraints.Max.Y = height
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, height)}.Op())
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, height)}
		}),
		layout.Stacked(func(gtx C) D {
			return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, pageLabel(s.theme, unit.Sp(17), title, color.NRGBA{R: 28, G: 28, B: 32, A: 255})),
					layout.Rigid(func(gtx C) D { return material.Switch(s.theme, value, "").Layout(gtx) }),
				)
			})
		}),
	)
}

func (s *appState) externalKeyboardPage(gtx C) D {
	return s.externalList.Layout(gtx, 4, func(gtx C, index int) D {
		if index == 0 {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D { return s.sectionGap(gtx) }),
				layout.Rigid(func(gtx C) D { return s.externalRow(gtx, index) }),
			)
		}
		return s.externalRow(gtx, index)
	})
}

func (s *appState) externalRow(gtx C, index int) D {
	titles := []string{"Caps Lock Mapping", "Option Mapping", "Backtick Maps Escape", "Override Control-Space"}
	details := []string{
		capsLockMappingName(s.prefs.CapsLockMapping),
		optionMappingName(s.prefs.OptionMapping),
		boolPreferenceName(s.prefs.BacktickMapEscape),
		boolPreferenceName(s.prefs.OverrideControlSpace),
	}
	return s.pageRow(gtx, &s.externalRows[index], titles[index], details[index], func() {
		s.updatePreferences(func(p *userPreferences) {
			switch index {
			case 0:
				p.CapsLockMapping = (p.CapsLockMapping + 1) % 3
			case 1:
				p.OptionMapping = (p.OptionMapping + 1) % 2
			case 2:
				p.BacktickMapEscape = !p.BacktickMapEscape
			case 3:
				p.OverrideControlSpace = !p.OverrideControlSpace
			}
		})
	})
}

func boolPreferenceName(value bool) string {
	if value {
		return "On"
	}
	return "Off"
}

func terminalKeyEventBytes(e key.Event, prefs userPreferences) ([]byte, bool) {
	if e.Modifiers == 0 {
		switch e.Name {
		case key.NameEscape:
			return []byte("\x1b"), true
		case key.NameUpArrow:
			return []byte("\x1b[A"), true
		case key.NameDownArrow:
			return []byte("\x1b[B"), true
		case key.NameLeftArrow:
			return []byte("\x1b[D"), true
		case key.NameRightArrow:
			return []byte("\x1b[C"), true
		}
		if string(e.Name) == "`" && prefs.BacktickMapEscape {
			return []byte("\x1b"), true
		}
	}
	if e.Modifiers.Contain(key.ModAlt) && prefs.OptionMapping == optionEscape && isMetaKey(string(e.Name)) {
		return append([]byte{'\x1b'}, []byte(string(e.Name))...), true
	}
	if e.Modifiers.Contain(key.ModCtrl) && prefs.OverrideControlSpace && e.Name == key.NameSpace {
		return []byte{0}, true
	}
	return nil, false
}

func (s *appState) filesystemsPage(gtx C) D {
	return s.filesList.Layout(gtx, 2, func(gtx C, index int) D {
		if index == 0 {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D { return s.sectionGap(gtx) }),
				layout.Rigid(func(gtx C) D { return s.filesRow(gtx, index) }),
			)
		}
		return s.filesRow(gtx, index)
	})
}

func (s *appState) filesRow(gtx C, index int) D {
	base := s.rootfsBase
	if base == "" {
		base = "Alpine root"
	}
	titles := []string{"Default", "Browse Files"}
	details := []string{base, "Open the app-private rootfs"}
	return s.pageRow(gtx, &s.filesRows[index], titles[index], details[index], func() {
		if index == 1 {
			s.openFilesBrowser()
		}
	})
}

func (s *appState) openFilesBrowser() {
	s.fileEntries = nil
	if s.rootfsBase != "" {
		entries, err := os.ReadDir(s.rootfsBase)
		if err != nil {
			s.fileEntries = []string{"Unable to read rootfs: " + err.Error()}
		} else {
			for _, entry := range entries {
				name := entry.Name()
				if entry.IsDir() {
					name += "/"
				}
				s.fileEntries = append(s.fileEntries, name)
			}
		}
	}
	if len(s.fileEntries) == 0 {
		s.fileEntries = []string{"Alpine rootfs is not ready"}
	}
	s.page = pageFilesBrowser
}

func (s *appState) filesBrowserPage(gtx C) D {
	return s.browserList.Layout(gtx, len(s.fileEntries), func(gtx C, index int) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D { return s.sectionGap(gtx) }),
			layout.Rigid(func(gtx C) D {
				return s.pageRow(gtx, &s.filesRows[0], s.fileEntries[index], "", nil)
			}),
		)
	})
}

func (s *appState) aboutPage(gtx C) D {
	return s.aboutList.Layout(gtx, 5, func(gtx C, index int) D {
		if index == 0 {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D { return s.sectionGap(gtx) }),
				layout.Rigid(func(gtx C) D { return s.aboutRow(gtx, index) }),
			)
		}
		return s.aboutRow(gtx, index)
	})
}

func (s *appState) aboutRow(gtx C, index int) D {
	titles := []string{"iSH", "Send Feedback", "iSH on GitHub", "iSH Discord Server", "iSH on the Fediverse"}
	details := []string{"Go/Gio port · Alpine i386", "", "ish-app/ish", "Community", "Social updates"}
	return s.pageRow(gtx, &s.aboutRows[index%len(s.aboutRows)], titles[index], details[index], nil)
}

func (s *appState) terminalView(gtx C) D {
	paint.Fill(gtx.Ops, s.terminalPalette().background)
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
			dims := style.Layout(gtx)
			// widget.Editor clips its event.Op to the rendered text dimensions.
			// With an empty, transparent buffer that can be smaller than the
			// terminal, add a full terminal-surface target like iSH's first
			// responder. Keep the target clipped to terminalView: the accessory
			// bar is a separate sibling and must receive its own pointer events.
			terminalTarget := clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}.Push(gtx.Ops)
			event.Op(gtx.Ops, &s.input)
			key.InputHintOp{Tag: &s.input, Hint: key.HintText}.Add(gtx.Ops)
			terminalTarget.Pop()
			return layout.Dimensions{Size: dims.Size}
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
			log.Printf("iSH GUI editor change: %q", value)
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
		if err := s.session.Write(payload); err != nil {
			log.Printf("iSH GUI terminal input failed: %v", err)
		} else {
			log.Printf("iSH GUI terminal input forwarded: %q", payload)
		}
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

type terminalPalette struct {
	foreground color.NRGBA
	background color.NRGBA
	cursor     color.NRGBA
}

func (s *appState) terminalPalette() terminalPalette {
	dark := s.prefs.ColorScheme == colorAlwaysDark
	if s.prefs.ColorScheme == colorMatchSystem {
		// Gio does not expose a portable system-appearance query. Keep the
		// default theme light on the light Android/Linux environments used by
		// iSH, while allowing deterministic dark-mode tests through the env var.
		dark = os.Getenv("ISH_DARK_MODE") == "1"
	}
	palette := terminalPalette{
		foreground: color.NRGBA{R: 0, G: 0, B: 0, A: 255},
		background: color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		cursor:     color.NRGBA{R: 0, G: 0, B: 0, A: 255},
	}
	switch s.prefs.Theme {
	case "1337":
		palette.foreground = color.NRGBA{R: 0, G: 255, B: 0, A: 255}
		palette.background = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	case "Solarized":
		if dark {
			palette.foreground = color.NRGBA{R: 131, G: 148, B: 150, A: 255}
			palette.background = color.NRGBA{R: 0, G: 43, B: 54, A: 255}
		} else {
			palette.foreground = color.NRGBA{R: 101, G: 123, B: 131, A: 255}
			palette.background = color.NRGBA{R: 253, G: 246, B: 227, A: 255}
		}
	case "Hot Dog Stand":
		palette.foreground = color.NRGBA{R: 255, G: 255, B: 0, A: 255}
		palette.background = color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	default:
		if dark {
			palette.foreground = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			palette.background = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
		}
	}
	palette.cursor = palette.foreground
	return palette
}

func maxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func (s *appState) terminalTypeface() giofont.Typeface {
	if s.prefs.FontFamily == "DejaVu Sans Mono" {
		// The embedded font is DejaVu Sans; use its registered family on
		// Android/Linux instead of relying on an uninstalled Mono alias.
		return giofont.Typeface("DejaVu Sans")
	}
	return giofont.Typeface(s.prefs.FontFamily)
}

func (s *appState) renderTerminal(gtx C) D {
	cellWidthDp := maxFloat32(7, s.prefs.FontSize*0.6)
	cellHeightDp := maxFloat32(12, s.prefs.FontSize*1.35)
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
	fontStyle := material.Label(s.theme, unit.Sp(s.prefs.FontSize), "")
	fontStyle.Font.Typeface = s.terminalTypeface()
	fontStyle.LineHeight = unit.Sp(cellHeightDp)
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
			paint.FillShape(gtx.Ops, s.cellBackground(line[start]), clip.Rect{Max: image.Pt(w, cellHeight)}.Op())
			fontStyle.Text = textValue
			fontStyle.Color = s.cellForeground(line[start])
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
	if cursor.X >= 0 && cursor.X < cols && cursor.Y >= 0 && cursor.Y < rows && cursorVisible(gtx.Now, s.prefs.BlinkCursor) {
		push := op.Offset(image.Pt(cursor.X*cellWidth, cursor.Y*cellHeight)).Push(gtx.Ops)
		cursorColor := s.terminalPalette().cursor
		switch s.prefs.CursorStyle {
		case cursorBeam:
			paint.FillShape(gtx.Ops, cursorColor, clip.Rect{Max: image.Pt(maxInt(2, gtx.Dp(unit.Dp(2))), cellHeight)}.Op())
		case cursorUnderline:
			cursorHeight := maxInt(2, gtx.Dp(unit.Dp(2)))
			shift := op.Offset(image.Pt(0, cellHeight-cursorHeight)).Push(gtx.Ops)
			paint.FillShape(gtx.Ops, cursorColor, clip.Rect{Max: image.Pt(cellWidth, cursorHeight)}.Op())
			shift.Pop()
		default:
			paint.FillShape(gtx.Ops, cursorColor, clip.Rect{Max: image.Pt(cellWidth, cellHeight)}.Op())
		}
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

func cursorVisible(now time.Time, blink bool) bool {
	if !blink {
		return true
	}
	return (now.UnixNano()/500_000_000)%2 == 0
}

func (s *appState) cellForeground(cell buffer.BrushedRune) color.NRGBA {
	if cell.Brush.Invert {
		return s.cellBackground(buffer.BrushedRune{Brush: buffer.Brush{FG: cell.Brush.BG, BG: cell.Brush.FG}})
	}
	if cell.Brush.FG == buffer.DefaultFG {
		return s.terminalPalette().foreground
	}
	return ansiColor(cell.Brush.FG)
}

func (s *appState) cellBackground(cell buffer.BrushedRune) color.NRGBA {
	if cell.Brush.Invert {
		if cell.Brush.FG == buffer.DefaultFG {
			return s.terminalPalette().foreground
		}
		return ansiColor(cell.Brush.FG)
	}
	if cell.Brush.BG == buffer.DefaultBG {
		return s.terminalPalette().background
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
	clicked := s.buttons[index].Clicked(gtx)
	return s.buttons[index].Layout(gtx, func(gtx C) D {
		pressed := s.buttons[index].Pressed() || s.buttons[index].Hovered()
		if clicked {
			if index == 5 {
				s.page = pageSettings
				log.Printf("iSH GUI page navigation: Terminal -> Settings")
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}
			}
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
			if s.arrowPressed {
				paint.FillShape(gtx.Ops, keySecondary, face.Op(gtx.Ops))
			}
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}
		}),
		layout.Stacked(func(gtx C) D {
			s.arrowDrag.Add(gtx.Ops)
			for {
				event, ok := s.arrowDrag.Update(gtx.Metric, gtx.Source, gesture.Both)
				if !ok {
					break
				}
				s.handleArrowEvent(gtx, event)
			}
			s.tickArrowRepeat(gtx.Now)

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

func arrowDirectionFromDelta(dx, dy, threshold float32) int {
	if dx*dx+dy*dy < threshold*threshold {
		return arrowNone
	}
	if absFloat32(dx) > absFloat32(dy) {
		if dx > 0 {
			return 3 // ArrowRight
		}
		return 2 // ArrowLeft
	}
	if dy > 0 {
		return 1 // ArrowDown
	}
	return 0 // ArrowUp
}

func absFloat32(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}

func (s *appState) handleArrowEvent(gtx C, event pointer.Event) {
	switch event.Kind {
	case pointer.Press:
		s.arrowPressed = true
		s.arrowStart = event.Position
		s.setArrowDirection(arrowNone, gtx.Now)
	case pointer.Drag:
		if !s.arrowPressed {
			return
		}
		threshold := float32(gtx.Dp(unit.Dp(20)))
		delta := event.Position.Sub(s.arrowStart)
		s.setArrowDirection(arrowDirectionFromDelta(delta.X, delta.Y, threshold), gtx.Now)
	case pointer.Release, pointer.Cancel:
		s.arrowPressed = false
		s.arrowDirection = arrowNone
		s.arrowNextRepeat = time.Time{}
	}
}

func (s *appState) setArrowDirection(direction int, now time.Time) {
	if direction == s.arrowDirection {
		return
	}
	s.arrowDirection = direction
	s.arrowNextRepeat = time.Time{}
	if direction != arrowNone {
		s.sendArrow(direction)
		s.arrowNextRepeat = now.Add(500 * time.Millisecond)
	}
}

func (s *appState) tickArrowRepeat(now time.Time) {
	if !s.arrowPressed || s.arrowDirection == arrowNone || s.arrowNextRepeat.IsZero() || now.Before(s.arrowNextRepeat) {
		return
	}
	for !now.Before(s.arrowNextRepeat) {
		s.sendArrow(s.arrowDirection)
		s.arrowNextRepeat = s.arrowNextRepeat.Add(100 * time.Millisecond)
	}
}

func (s *appState) arrowPart(gtx C, index, width, height int) D {
	gtx.Constraints.Min = image.Pt(width, height)
	gtx.Constraints.Max = image.Pt(width, height)
	glyphs := []string{"↑", "↓", "←", "→"}
	label := material.Label(s.theme, unit.Sp(15), glyphs[index])
	label.Alignment = text.Middle
	if s.arrowPressed && s.arrowDirection != arrowNone && s.arrowDirection != index {
		label.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 64}
	} else {
		label.Color = keyForeground
	}
	return layout.Center.Layout(gtx, label.Layout)
}

func (s *appState) drawBarIcon(gtx C, index int) {
	switch index {
	case 0: // The storyboard title is exactly "⇥".
		s.drawOriginalGlyph(gtx, "⇥", unit.Sp(20))
	case 1: // The storyboard title is exactly "⌃".
		s.drawOriginalGlyph(gtx, "⌃", unit.Sp(20))
	case 2: // The storyboard title is exactly "⎋".
		s.drawOriginalGlyph(gtx, "⎋", unit.Sp(24))
	case 4: // Assets.xcassets/Paste.imageset/Paste.pdf.
		drawTemplateIcon(gtx, pasteOriginalIcon, 20)
	case 5: // UIKit buttonType=infoLight, accessibility label "Settings".
		s.drawOriginalGlyph(gtx, "ⓘ", unit.Sp(22))
	case 6: // Assets.xcassets/Hide Keyboard.imageset/Hide Keyboard.pdf.
		drawTemplateIcon(gtx, hideKeyboardOriginalIcon, 32)
	}
}

func (s *appState) drawOriginalGlyph(gtx C, glyph string, size unit.Sp) {
	label := material.Label(s.theme, size, glyph)
	label.Color = keyForeground
	label.Alignment = text.Middle
	// This family is bundled above, so the storyboard glyphs do not depend on
	// whichever Android system font happens to be installed.
	label.Font.Typeface = "DejaVu Sans"
	layout.Center.Layout(gtx, label.Layout)
}

func drawTemplateIcon(gtx C, icon image.Image, sizeDp float32) {
	if icon == nil {
		return
	}
	bounds := icon.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return
	}
	target := gtx.Dp(unit.Dp(sizeDp))
	scale := float32(target) / float32(maxInt(bounds.Dx(), bounds.Dy()))
	width := maxInt(1, int(float32(bounds.Dx())*scale))
	height := maxInt(1, int(float32(bounds.Dy())*scale))
	x := (gtx.Constraints.Max.X - width) / 2
	y := (gtx.Constraints.Max.Y - height) / 2
	transform := op.Affine(f32.AffineId().Scale(f32.Point{}, f32.Pt(scale, scale)).Offset(f32.Pt(float32(x), float32(y))))
	stack := transform.Push(gtx.Ops)
	paint.NewImageOp(icon).Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	stack.Pop()
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
