package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"

	"gioui.org/app"
)

// These enums mirror the values in upstream/ish-ios/app/UserPreferences.h.
type capsLockMapping uint8

type optionMapping uint8

type cursorStyle uint8

type colorScheme uint8

const (
	capsLockNone capsLockMapping = iota
	capsLockControl
	capsLockEscape
)

const (
	optionNone optionMapping = iota
	optionEscape
)

const (
	cursorBlock cursorStyle = iota
	cursorBeam
	cursorUnderline
)

const (
	colorMatchSystem colorScheme = iota
	colorAlwaysLight
	colorAlwaysDark
)

// userPreferences is intentionally a small, JSON-backed counterpart of iSH's
// NSUserDefaults model. Keeping it in the app-private directory makes the
// settings survive Android process restarts without adding a second database.
type userPreferences struct {
	CapsLockMapping                   capsLockMapping `json:"caps_lock_mapping"`
	OptionMapping                     optionMapping   `json:"option_mapping"`
	BacktickMapEscape                 bool            `json:"backtick_mapping_escape"`
	HideExtraKeysWithExternalKeyboard bool            `json:"hide_extra_keys_with_external_keyboard"`
	OverrideControlSpace              bool            `json:"override_control_space"`
	HideStatusBar                     bool            `json:"hide_status_bar"`
	DisableDimming                    bool            `json:"disable_dimming"`
	FontFamily                        string          `json:"font_family"`
	FontSize                          float32         `json:"font_size"`
	Theme                             string          `json:"theme"`
	ColorScheme                       colorScheme     `json:"color_scheme"`
	CursorStyle                       cursorStyle     `json:"cursor_style"`
	BlinkCursor                       bool            `json:"blink_cursor"`
}

func defaultUserPreferences() userPreferences {
	return userPreferences{
		CapsLockMapping:   capsLockControl,
		OptionMapping:     optionNone,
		FontFamily:        "monospace",
		FontSize:          12,
		Theme:             "Default",
		ColorScheme:       colorMatchSystem,
		CursorStyle:       cursorBlock,
		BlinkCursor:       false,
		HideStatusBar:     false,
		DisableDimming:    false,
		BacktickMapEscape: false,
	}
}

func (p *userPreferences) normalize() {
	if p.CapsLockMapping > capsLockEscape {
		p.CapsLockMapping = capsLockControl
	}
	if p.OptionMapping > optionEscape {
		p.OptionMapping = optionNone
	}
	if p.FontFamily == "" {
		p.FontFamily = "monospace"
	}
	if p.FontSize < 8 || p.FontSize > 48 {
		p.FontSize = 12
	}
	if p.Theme != "Default" && p.Theme != "1337" && p.Theme != "Solarized" && p.Theme != "Hot Dog Stand" {
		p.Theme = "Default"
	}
	if p.ColorScheme > colorAlwaysDark {
		p.ColorScheme = colorMatchSystem
	}
	if p.CursorStyle > cursorUnderline {
		p.CursorStyle = cursorBlock
	}
}

func loadUserPreferences(path string) (userPreferences, error) {
	prefs := defaultUserPreferences()
	if path == "" {
		return prefs, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return prefs, nil
	}
	if err != nil {
		return prefs, err
	}
	if err := json.Unmarshal(data, &prefs); err != nil {
		return prefs, err
	}
	prefs.normalize()
	return prefs, nil
}

func saveUserPreferences(path string, prefs userPreferences) error {
	if path == "" {
		return nil
	}
	prefs.normalize()
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func appDataDir() (string, error) {
	return app.DataDir()
}

func preferenceFilePath() string {
	if path := os.Getenv("ISH_PREFERENCES_PATH"); path != "" {
		return path
	}
	dataDir, err := appDataDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dataDir, "user-preferences.json")
}

func capsLockMappingName(value capsLockMapping) string {
	switch value {
	case capsLockNone:
		return "None"
	case capsLockEscape:
		return "Escape"
	default:
		return "Control"
	}
}

func optionMappingName(value optionMapping) string {
	if value == optionEscape {
		return "Escape"
	}
	return "None"
}

func cursorStyleName(value cursorStyle) string {
	switch value {
	case cursorBeam:
		return "Beam"
	case cursorUnderline:
		return "Underline"
	default:
		return "Block"
	}
}

func colorSchemeName(value colorScheme) string {
	switch value {
	case colorAlwaysLight:
		return "Always Light"
	case colorAlwaysDark:
		return "Always Dark"
	default:
		return "Match System"
	}
}

func nextFontSize(value float32) float32 {
	for _, candidate := range []float32{10, 12, 14, 16, 18, 20} {
		if candidate > value {
			return candidate
		}
	}
	return 10
}

func isMetaKey(name string) bool {
	const metaKeys = "abcdefghijklmnopqrstuvwxyz0123456789-=[]\\;',./"
	if len(name) != 1 {
		return false
	}
	for _, candidate := range metaKeys {
		if string(candidate) == name {
			return true
		}
	}
	return false
}

func (s *appState) updatePreferences(update func(*userPreferences)) {
	before := s.prefs
	update(&s.prefs)
	s.prefs.normalize()
	if before == s.prefs {
		return
	}
	if err := saveUserPreferences(s.prefsPath, s.prefs); err != nil {
		log.Printf("iSH preferences save failed: %v", err)
	}
}

func (s *appState) syncPreferences() {
	changed := false
	if s.blinkCursor.Value != s.prefs.BlinkCursor {
		s.prefs.BlinkCursor = s.blinkCursor.Value
		changed = true
	}
	if s.hideStatusBar.Value != s.prefs.HideStatusBar {
		s.prefs.HideStatusBar = s.hideStatusBar.Value
		changed = true
	}
	if !changed {
		return
	}
	s.prefs.normalize()
	if err := saveUserPreferences(s.prefsPath, s.prefs); err != nil {
		log.Printf("iSH preferences switch save failed: %v", err)
	}
}
