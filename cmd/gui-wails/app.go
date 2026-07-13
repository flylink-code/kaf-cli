//go:build windows && wailsgui

package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	kafcli "github.com/ystyle/kaf-cli/internal/kafcli"
	"github.com/ystyle/kaf-cli/internal/kafcli/ai"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var formatOptions = []string{"all", "epub", "mobi", "azw3"}

type App struct {
	ctx        context.Context
	lastOutDir string

	mu         sync.Mutex
	converting bool
}

// GetVersion returns the build version displayed by the desktop application.
func (a *App) GetVersion() string {
	return version
}

type sourceInsight struct {
	Bookname string `json:"bookname"`
	Author   string `json:"author"`
	Cover    string `json:"cover"`
}

type convertRequest struct {
	TxtFile     string    `json:"txtFile"`
	CoverFile   string    `json:"coverFile"`
	Author      string    `json:"author"`
	Format      string    `json:"format"`
	Match       string    `json:"match"`
	VolumeMatch string    `json:"volumeMatch"`
	Dedup       bool      `json:"dedup"`
	Tips        bool      `json:"tips"`
	Quotes      bool      `json:"quotes"`
	// AI 选项中的密钥不从请求取，而从持久化配置读，避免明文在前端流转。
	AI convertAIRequest `json:"ai"`
}

// convertAIRequest 是转换时携带的 AI 开关；密钥始终从持久化配置注入。
type convertAIRequest struct {
	Enabled     bool `json:"enabled"`
	Structure   bool `json:"structure"`
	Typography  bool `json:"typography"`
	Noise       bool `json:"noise"`
	Metadata    bool `json:"metadata"`
	SampleChars int  `json:"sampleChars"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetConfig() guiConfig {
	return loadGUIConfig()
}

func (a *App) SaveConfig(cfg guiConfig) error {
	// 前端 saveConfig 不含 ai 段；须合并已有配置，避免覆盖 GUI/AI 设置里保存的密钥。
	existing := loadGUIConfig()
	existing.TxtFile = cfg.TxtFile
	existing.CoverFile = cfg.CoverFile
	existing.Author = cfg.Author
	existing.FormatIndex = cfg.FormatIndex
	existing.Match = cfg.Match
	existing.VolumeMatch = cfg.VolumeMatch
	existing.Dedup = cfg.Dedup
	existing.Tips = cfg.Tips
	existing.Quotes = cfg.Quotes
	saveGUIConfig(existing)
	return nil
}

// GetAIConfig 返回持久化的 AI 配置（含密钥，仅供本地 UI 回填）。
func (a *App) GetAIConfig() aiConfig {
	cfg := loadGUIConfig()
	return cfg.AI
}

// SaveAIConfig 单独保存 AI 配置段，保留其余字段。
func (a *App) SaveAIConfig(aiCfg aiConfig) error {
	cfg := loadGUIConfig()
	cfg.AI = aiCfg
	saveGUIConfig(cfg)
	return nil
}

// TestAIConnection 用给定配置发起一次最小请求，验证连通性。
// 返回 ok=true 与模型实际回显；失败时 ok=false 并带错误信息。
func (a *App) TestAIConnection(aiCfg aiConfig) aiTestResult {
	client := ai.NewClient(ai.ClientConfig{
		BaseURL: aiCfg.BaseURL,
		APIKey:  aiCfg.APIKey,
		Model:   aiCfg.Model,
		Timeout: 30 * time.Second,
	})
	if !client.Ready() {
		return aiTestResult{OK: false, Message: "缺少 API Key 或 Model"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := client.Chat(ctx,
		"你是连通性测试助手，请回复『连接成功』四个字。",
		"ping", false)
	if err != nil {
		return aiTestResult{OK: false, Message: err.Error()}
	}
	return aiTestResult{OK: true, Message: resp}
}

type aiTestResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func (a *App) PickTXT() (string, error) {
	if a.ctx == nil {
		return "", errors.New("应用尚未初始化")
	}
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 TXT 文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "文本文件 (*.txt)", Pattern: "*.txt"},
		},
	})
}

func (a *App) PickCover() (string, error) {
	if a.ctx == nil {
		return "", errors.New("应用尚未初始化")
	}
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择封面图片",
		Filters: []runtime.FileFilter{
			{DisplayName: "图片文件", Pattern: "*.png;*.jpg;*.jpeg"},
		},
	})
}

func (a *App) InspectSource(txtPath string) sourceInsight {
	txtPath = strings.TrimSpace(txtPath)
	if txtPath == "" {
		return sourceInsight{}
	}
	bookname, author := kafcli.FilenameMeta(txtPath)
	return sourceInsight{
		Bookname: bookname,
		Author:   author,
		Cover:    findCover(txtPath),
	}
}

func (a *App) Convert(req convertRequest) error {
	txtPath := strings.TrimSpace(req.TxtFile)
	if txtPath == "" {
		return errors.New("请先选择 TXT 文件")
	}

	a.mu.Lock()
	if a.converting {
		a.mu.Unlock()
		return errors.New("当前已有转换任务正在执行")
	}
	a.converting = true
	a.lastOutDir = filepath.Dir(txtPath)
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.converting = false
		a.mu.Unlock()
	}()

	runtime.EventsEmit(a.ctx, "convert:state", "running")
	err := runConvert(req.toGUIOptions(), func(chunk string) {
		runtime.EventsEmit(a.ctx, "convert:log", chunk)
	})
	if err != nil {
		runtime.EventsEmit(a.ctx, "convert:state", "error")
		return err
	}

	runtime.EventsEmit(a.ctx, "convert:state", "done")
	return nil
}

func (a *App) OpenLastOutputDir() error {
	a.mu.Lock()
	dir := a.lastOutDir
	a.mu.Unlock()
	if strings.TrimSpace(dir) == "" {
		return errors.New("当前还没有可打开的输出目录")
	}
	return openFolder(dir)
}

func (req convertRequest) toGUIOptions() kafcli.GUIOptions {
	format := "all"
	for _, item := range formatOptions {
		if req.Format == item {
			format = req.Format
			break
		}
	}
	opts := kafcli.GUIOptions{
		Filename:        strings.TrimSpace(req.TxtFile),
		Cover:           strings.TrimSpace(req.CoverFile),
		Author:          strings.TrimSpace(req.Author),
		Format:          format,
		Match:           strings.TrimSpace(req.Match),
		VolumeMatch:     strings.TrimSpace(req.VolumeMatch),
		DedupTitle:      req.Dedup,
		Tips:            req.Tips,
		NormalizeQuotes: req.Quotes,
	}
	// AI：开关取自请求，密钥与模型取自持久化配置（避免明文在前端流转）。
	// Enabled 保留用户意图；是否真正调用由核心库依据 Client.Ready() 决定并打日志。
	if req.AI.Enabled {
		saved := loadGUIConfig().AI
		client := ai.NewClient(ai.ClientConfig{
			BaseURL: saved.BaseURL,
			APIKey:  saved.APIKey,
			Model:   saved.Model,
		})
		opts.AI = kafcli.AIRefineOptions{
			Enabled:      true,
			Client:       client,
			SampleChars:  req.AI.SampleChars,
			DoStructure:  req.AI.Structure,
			DoTypography: req.AI.Typography,
			DoNoise:      req.AI.Noise,
			DoMetadata:   req.AI.Metadata,
		}
	}
	return opts
}

func findCover(txtPath string) string {
	base := strings.TrimSuffix(txtPath, filepath.Ext(txtPath))
	for _, ext := range []string{".png", ".jpg", ".jpeg"} {
		p := base + ext
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func openFolder(dir string) error {
	dir = filepath.Clean(dir)
	cmd := exec.Command("explorer.exe", dir)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	return cmd.Start()
}
