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

	kafcli "github.com/ystyle/kaf-cli/internal/kafcli"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var formatOptions = []string{"all", "epub", "mobi", "azw3"}

type App struct {
	ctx        context.Context
	lastOutDir string

	mu         sync.Mutex
	converting bool
}

type sourceInsight struct {
	Bookname string `json:"bookname"`
	Author   string `json:"author"`
	Cover    string `json:"cover"`
}

type convertRequest struct {
	TxtFile     string `json:"txtFile"`
	CoverFile   string `json:"coverFile"`
	Author      string `json:"author"`
	Format      string `json:"format"`
	Dedup       bool   `json:"dedup"`
	Tips        bool   `json:"tips"`
	Quotes      bool   `json:"quotes"`
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
	saveGUIConfig(cfg)
	return nil
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
	return kafcli.GUIOptions{
		Filename:        strings.TrimSpace(req.TxtFile),
		Cover:           strings.TrimSpace(req.CoverFile),
		Author:          strings.TrimSpace(req.Author),
		Format:          format,
		DedupTitle:      req.Dedup,
		Tips:            req.Tips,
		NormalizeQuotes: req.Quotes,
	}
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
