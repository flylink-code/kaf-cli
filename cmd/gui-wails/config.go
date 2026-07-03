//go:build windows && wailsgui

package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	kafcli "github.com/ystyle/kaf-cli/internal/kafcli"
)

type guiConfig struct {
	TxtFile     string   `json:"txt_file"`
	CoverFile   string   `json:"cover_file"`
	Author      string   `json:"author"`
	FormatIndex int      `json:"format_index"`
	Match       string   `json:"match"`
	VolumeMatch string   `json:"volume_match"`
	Dedup       bool     `json:"dedup"`
	Tips        bool     `json:"tips"`
	Quotes      bool     `json:"quotes"`
	AI          aiConfig `json:"ai"`
}

// aiConfig 存储 AI 后处理配置。
// APIKey 在内存中为明文，写入 gui-config.json 前会用 DPAPI 加密成密文，
// 仅当前 Windows 账户可解密；换机器/账户后需重填。
type aiConfig struct {
	Enabled     bool    `json:"enabled"`      // 总开关，默认关闭
	BaseURL     string  `json:"base_url"`     // OpenAI 兼容 baseURL
	APIKey      string  `json:"api_key"`      // 内存明文；落盘为 DPAPI 密文(base64)
	Model       string  `json:"model"`        // 例 deepseek-chat
	SampleChars int     `json:"sample_chars"` // 正文抽样上限，0=仅分析目录
	Tasks       aiTasks `json:"tasks"`
}

type aiTasks struct {
	Structure  bool `json:"structure"`  // 章节结构分析，默认开
	Typography bool `json:"typography"` // 排版修正
	Noise      bool `json:"noise"`      // 噪音清理
	Metadata   bool `json:"metadata"`   // 简介生成
}

// defaultAIConfig 返回推荐默认值：仅启用结构分析，不抽正文。
func defaultAIConfig() aiConfig {
	return aiConfig{
		Tasks: aiTasks{Structure: true},
	}
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "kaf-cli", "gui-config.json"), nil
}

func loadGUIConfig() guiConfig {
	cfg := guiConfig{Dedup: true, Tips: true, AI: defaultAIConfig()}
	path, err := configPath()
	if err != nil {
		return cfg
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	// 先探测文件中是否存在 ai 段；缺失则保留代码默认值，避免被零值覆盖。
	raw := map[string]json.RawMessage{}
	_ = json.Unmarshal(data, &raw)
	if _, ok := raw["ai"]; ok {
		_ = json.Unmarshal(data, &cfg)
	} else {
		aiDefault := defaultAIConfig()
		_ = json.Unmarshal(data, &cfg)
		cfg.AI = aiDefault
	}
	// 解密 APIKey：DPAPI 密文 → 明文。
	// 兼容旧版明文：解密失败但内容像明文 key 时，按明文保留（下次保存自动转密文）。
	cfg.AI.APIKey = decryptAPIKey(cfg.AI.APIKey)
	return cfg
}

func saveGUIConfig(cfg guiConfig) {
	path, err := configPath()
	if err != nil {
		return
	}
	// 落盘前加密 APIKey；失败则保留原值（不致阻断保存，但会落盘明文）。
	if encrypted, err := kafcli.Protect(cfg.AI.APIKey); err == nil {
		cfg.AI.APIKey = encrypted
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o600)
}

// decryptAPIKey 把存储中的密文还原为明文。
// 兼容三种情形：空、DPAPI 密文、旧版明文。
func decryptAPIKey(stored string) string {
	if stored == "" {
		return ""
	}
	if plain, err := kafcli.Unprotect(stored); err == nil {
		return plain
	}
	// 解密失败：可能是旧版明文 key，也可能是换机器后的别账户密文。
	// 这里按明文返回，下次保存会被加密迁移；若确实是无效密文，用户使用时会看到连接失败。
	return stored
}
