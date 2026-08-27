package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"gioui.org/io/key"
)

func TestControlByteMatchesISH(t *testing.T) {
	tests := []struct {
		name string
		in   rune
		want byte
	}{
		{name: "lowercase c", in: 'c', want: 3},
		{name: "uppercase c", in: 'C', want: 3},
		{name: "space", in: ' ', want: 0},
		{name: "at", in: '@', want: 0},
		{name: "two", in: '2', want: 0},
		{name: "six", in: '6', want: 30},
		{name: "left bracket", in: '[', want: 27},
		{name: "backslash", in: '\\', want: 28},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := controlByte(test.in)
			if !ok || got != test.want {
				t.Fatalf("controlByte(%q) = %#02x, %v; want %#02x, true", test.in, got, ok, test.want)
			}
		})
	}
	if _, ok := controlByte('q'); !ok {
		t.Fatal("alphabetic control key must be accepted")
	}
	if _, ok := controlByte('!'); ok {
		t.Fatal("unsupported control key must be rejected")
	}
}

func TestTerminalInputBytes(t *testing.T) {
	got, next := terminalInputBytes("echo\n", false)
	if !bytes.Equal(got, []byte("echo\r")) || next {
		t.Fatalf("ordinary newline input = %q, next=%v; want carriage return and false", got, next)
	}
	got, next = terminalInputBytes("c", true)
	if !bytes.Equal(got, []byte{3}) || next {
		t.Fatalf("Control-c input = %#v, next=%v; want [3], false", got, next)
	}
	got, next = terminalInputBytes("ab", true)
	if !bytes.Equal(got, []byte("ab")) || next {
		t.Fatalf("multi-rune Control insertion = %q, next=%v; want ordinary text and false", got, next)
	}
	got, next = terminalInputBytes("!", true)
	if got != nil || next {
		t.Fatalf("unsupported one-rune Control insertion = %#v, next=%v; want nil and false", got, next)
	}
	got, next = terminalInputBytes("", true)
	if got != nil || !next {
		t.Fatalf("empty input = %#v, next=%v; want nil, true", got, next)
	}
}

func TestArrowDirectionFromDelta(t *testing.T) {
	tests := []struct {
		name, want string
		dx, dy     float32
	}{
		{name: "below slop", dx: 19, dy: 0, want: "none"},
		{name: "up", dx: 0, dy: -21, want: "up"},
		{name: "down", dx: 0, dy: 21, want: "down"},
		{name: "left", dx: -22, dy: 3, want: "left"},
		{name: "right", dx: 22, dy: -3, want: "right"},
	}
	labels := map[int]string{arrowNone: "none", 0: "up", 1: "down", 2: "left", 3: "right"}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := labels[arrowDirectionFromDelta(test.dx, test.dy, 20)]
			if got != test.want {
				t.Fatalf("arrowDirectionFromDelta(%v, %v, 20) = %q; want %q", test.dx, test.dy, got, test.want)
			}
		})
	}
}

func TestMetricsForWidth(t *testing.T) {
	tests := []struct {
		width, button, horizontal, vertical, barHeight, gap float32
	}{
		{width: 320, button: 32, horizontal: 6, vertical: 6, barHeight: 36, gap: 6},
		{width: 429, button: 32, horizontal: 6, vertical: 6, barHeight: 36, gap: 6},
		{width: 430, button: 36, horizontal: 10, vertical: 8, barHeight: 36, gap: 6},
		{width: 599, button: 36, horizontal: 10, vertical: 8, barHeight: 36, gap: 6},
		{width: 600, button: 43, horizontal: 15, vertical: 8, barHeight: 43, gap: 6},
	}
	for _, test := range tests {
		got := metricsForWidth(test.width)
		if got.button != test.button || got.horizontal != test.horizontal || got.vertical != test.vertical || got.barHeight != test.barHeight || got.gap != test.gap {
			t.Errorf("metricsForWidth(%v) = %+v; want button=%v horizontal=%v vertical=%v barHeight=%v gap=%v", test.width, got, test.button, test.horizontal, test.vertical, test.barHeight, test.gap)
		}
	}
	if !metricsForWidth(390).showHideKeyboard {
		t.Fatal("phone layout must retain the hide-keyboard control")
	}
	if metricsForWidth(700).showHideKeyboard {
		t.Fatal("wide iPad layout must omit the hide-keyboard control")
	}
}

func TestUserPreferencesRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "user-preferences.json")
	want := defaultUserPreferences()
	want.Theme = "Solarized"
	want.FontFamily = "DejaVu Sans Mono"
	want.FontSize = 18
	want.ColorScheme = colorAlwaysDark
	want.CursorStyle = cursorUnderline
	want.BlinkCursor = true
	want.BacktickMapEscape = true
	want.OptionMapping = optionEscape
	if err := saveUserPreferences(path, want); err != nil {
		t.Fatalf("saveUserPreferences: %v", err)
	}
	got, err := loadUserPreferences(path)
	if err != nil {
		t.Fatalf("loadUserPreferences: %v", err)
	}
	if got != want {
		t.Fatalf("round trip = %+v; want %+v", got, want)
	}
}

func TestUserPreferencesNormalizeInvalidValues(t *testing.T) {
	prefs := userPreferences{
		CapsLockMapping: 99,
		OptionMapping:   99,
		FontSize:        100,
		Theme:           "unknown",
		ColorScheme:     99,
		CursorStyle:     99,
	}
	prefs.normalize()
	defaults := defaultUserPreferences()
	if prefs.CapsLockMapping != defaults.CapsLockMapping || prefs.OptionMapping != defaults.OptionMapping || prefs.FontSize != defaults.FontSize || prefs.Theme != defaults.Theme || prefs.ColorScheme != defaults.ColorScheme || prefs.CursorStyle != defaults.CursorStyle {
		t.Fatalf("normalize = %+v; expected invalid fields to return to defaults", prefs)
	}
}

func TestTerminalKeyEventBytes(t *testing.T) {
	prefs := defaultUserPreferences()
	prefs.BacktickMapEscape = true
	prefs.OptionMapping = optionEscape
	prefs.OverrideControlSpace = true
	tests := []struct {
		name    string
		e       key.Event
		want    []byte
		handled bool
	}{
		{name: "backtick escape", e: key.Event{Name: "`", Modifiers: 0}, want: []byte{0x1b}, handled: true},
		{name: "option meta", e: key.Event{Name: "x", Modifiers: key.ModAlt}, want: []byte{0x1b, 'x'}, handled: true},
		{name: "control space", e: key.Event{Name: key.NameSpace, Modifiers: key.ModCtrl}, want: []byte{0}, handled: true},
		{name: "ordinary key", e: key.Event{Name: "x", Modifiers: 0}, handled: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, handled := terminalKeyEventBytes(test.e, prefs)
			if handled != test.handled || !bytes.Equal(got, test.want) {
				t.Fatalf("terminalKeyEventBytes(%+v) = %#v, %v; want %#v, %v", test.e, got, handled, test.want, test.handled)
			}
		})
	}
}

func TestPageNavigationHierarchy(t *testing.T) {
	state := &appState{page: pageSettings}
	if got := state.pageTitle(); got != "Settings" {
		t.Fatalf("settings title = %q", got)
	}

	for _, page := range []pageID{pageAppearance, pageExternalKeyboard, pageFilesystems, pageAbout} {
		state.page = page
		state.goBack()
		if state.page != pageSettings {
			t.Fatalf("page %d did not return to settings", page)
		}
	}

	state.page = pageFilesBrowser
	state.goBack()
	if state.page != pageFilesystems {
		t.Fatalf("files browser did not return to filesystems")
	}

	state.page = pageSettings
	state.goBack()
	if state.page != pageTerminal {
		t.Fatalf("settings did not return to terminal")
	}
}
