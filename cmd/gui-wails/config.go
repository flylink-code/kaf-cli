//go:build windows && wailsgui

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type guiConfig struct {
	TxtFile     string `json:"txt_file"`
	CoverFile   string `json:"cover_file"`
	Author      string `json:"author"`
	FormatIndex int    `json:"format_index"`
	Match       string `json:"match"`
	VolumeMatch string `json:"volume_match"`
	Dedup       bool   `json:"dedup"`
	Tips        bool   `json:"tips"`
	Quotes      bool   `json:"quotes"`
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "kaf-cli", "gui-config.json"), nil
}

func loadGUIConfig() guiConfig {
	cfg := guiConfig{Dedup: true, Tips: true}
	path, err := configPath()
	if err != nil {
		return cfg
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, &cfg)
	return cfg
}

func saveGUIConfig(cfg guiConfig) {
	path, err := configPath()
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o600)
}
