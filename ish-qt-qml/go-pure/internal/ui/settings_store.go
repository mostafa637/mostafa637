package ui

import (
	"encoding/json"
	"os"
)

func loadSettings(path string) *settingsState {
	v := defaultSettings
	if path == "" {
		return &v
	}
	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, &v)
	}
	return &v
}

func (s *Screen) saveSettings() {
	if s.SettingsPath == "" || s.settingsState == nil {
		return
	}
	data, err := json.MarshalIndent(s.settingsState, "", "  ")
	if err == nil {
		_ = os.WriteFile(s.SettingsPath, data, 0o600)
	}
}
