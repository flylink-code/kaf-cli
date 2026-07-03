//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	kafcli "github.com/ystyle/kaf-cli/internal/kafcli"
	"github.com/ystyle/kaf-cli/internal/kafcli/ai"
	"github.com/rodrigocfd/windigo/co"
	"github.com/rodrigocfd/windigo/ui"
	"github.com/rodrigocfd/windigo/win"
)

var (
	secret      string
	measurement string
	version     = "dev"
)

type appWindow struct {
	wnd          *ui.Main
	headerHint   *ui.Static
	txtFile      *ui.Edit
	coverFile    *ui.Edit
	authorEdit   *ui.Edit
	booknameLbl  *ui.Static
	statusLbl    *ui.Static
	formatCombo  *ui.ComboBox
	btnMore      *ui.Button
	btnConvert   *ui.Button
	btnOpenDir   *ui.Button
	txtLog       *ui.Edit
	dedupTitle   bool
	tipsEnabled  bool
	quoteFix     bool
	matchRule    string
	volumeRule   string
	aiConfig     aiConfig
	converting   bool
	lastOutDir   string
	logBuf       logBuffer
}

func main() {
	runtime.LockOSThread()

	_, _ = win.CoInitializeEx(co.COINIT_APARTMENTTHREADED | co.COINIT_DISABLE_OLE1DDE)
	defer win.CoUninitialize()

	app := newAppWindow()
	app.wnd.RunAsMain()
}

func newAppWindow() *appWindow {
	const (
		leftPad     = 18
		contentLeft = 28
		labelWidth  = 78
		gap         = 12
		inputWidth  = 470
		browseWidth = 96
		rowHeight   = 28
	)

	fullWidth := inputWidth + browseWidth + gap
	fieldLeft := contentLeft + labelWidth + gap
	browseLeft := fieldLeft + inputWidth + gap
	rightEdge := browseLeft + browseWidth

	app := &appWindow{
		wnd: ui.NewMain(
			ui.OptsMain().
			Title("kaf-cli 电子书转换").
			Size(ui.Dpi(740, 860)).
				ClassIconId(1).
				DropFiles(true),
		),
	}

	ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("把 TXT 文件拖进来，或从下方选择文件开始转换。").
		Position(ui.Dpi(leftPad, 18)).
		Layout(ui.LAY_RESIZE_HOLD))

	ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("转换素材").
		Position(ui.Dpi(leftPad, 52)))

	ui.NewStatic(app.wnd, ui.OptsStatic().
		Position(ui.Dpi(leftPad, 78)).
		Size(ui.DpiX(rightEdge-leftPad), 2).
		Layout(ui.LAY_RESIZE_HOLD).
		CtrlStyle(co.SS_ETCHEDHORZ))

	ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("TXT 文件").
		Position(ui.Dpi(contentLeft, 102)))

	app.txtFile = ui.NewEdit(app.wnd, ui.OptsEdit().
		Position(ui.Dpi(fieldLeft, 98)).
		Width(ui.DpiX(inputWidth)).
		Height(ui.DpiY(rowHeight)).
		Layout(ui.LAY_RESIZE_HOLD))
	app.txtFile.On().EnChange(func() {
		app.onTxtPathChanged()
		app.persistConfig()
	})
	setCueBanner(app.txtFile.Hwnd(), "选择小说 TXT 文件，支持拖拽导入")

	btnBrowseTxt := ui.NewButton(app.wnd, ui.OptsButton().
		Text("选择 TXT").
		Position(ui.Dpi(browseLeft, 98)).
		Width(ui.DpiX(browseWidth)).
		Height(ui.DpiY(rowHeight)).
		Layout(ui.LAY_MOVE_HOLD))
	btnBrowseTxt.On().BnClicked(func() {
		path, ok := pickFile(app.wnd.Hwnd(), "选择 TXT 文件", []win.COMDLG_FILTERSPEC{
			{Name: "文本文件 (*.txt)", Spec: "*.txt"},
		})
		if !ok {
			return
		}
		app.setTxtPath(path)
	})

	ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("封面图片").
		Position(ui.Dpi(contentLeft, 146)))

	app.coverFile = ui.NewEdit(app.wnd, ui.OptsEdit().
		Position(ui.Dpi(fieldLeft, 142)).
		Width(ui.DpiX(inputWidth)).
		Height(ui.DpiY(rowHeight)).
		Layout(ui.LAY_RESIZE_HOLD))
	app.coverFile.On().EnChange(func() {
		app.persistConfig()
	})
	setCueBanner(app.coverFile.Hwnd(), "可选，留空时会尝试自动匹配同名封面")

	btnBrowseCover := ui.NewButton(app.wnd, ui.OptsButton().
		Text("选择封面").
		Position(ui.Dpi(browseLeft, 142)).
		Width(ui.DpiX(browseWidth)).
		Height(ui.DpiY(rowHeight)).
		Layout(ui.LAY_MOVE_HOLD))
	btnBrowseCover.On().BnClicked(func() {
		path, ok := pickFile(app.wnd.Hwnd(), "选择封面图片", []win.COMDLG_FILTERSPEC{
			{Name: "图片 (*.png;*.jpg)", Spec: "*.png;*.jpg;*.jpeg"},
		})
		if ok {
			app.coverFile.SetText(path)
			app.persistConfig()
		}
	})

	ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("作者").
		Position(ui.Dpi(contentLeft, 190)))

	app.authorEdit = ui.NewEdit(app.wnd, ui.OptsEdit().
		Position(ui.Dpi(fieldLeft, 186)).
		Width(ui.DpiX(inputWidth)).
		Height(ui.DpiY(rowHeight)).
		Layout(ui.LAY_RESIZE_HOLD))
	app.authorEdit.On().EnChange(func() {
		app.persistConfig()
	})
	setCueBanner(app.authorEdit.Hwnd(), "可留空，程序会尽量从文件名自动提取")

	ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("自动识别").
		Position(ui.Dpi(contentLeft, 234)))

	app.booknameLbl = ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("书名将在选择 TXT 后显示").
		Position(ui.Dpi(fieldLeft, 234)).
		Size(ui.DpiX(fullWidth), ui.DpiY(20)).
		Layout(ui.LAY_RESIZE_HOLD).
		CtrlStyle(co.SS_LEFTNOWORDWRAP))

	ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("转换选项").
		Position(ui.Dpi(leftPad, 278)))

	ui.NewStatic(app.wnd, ui.OptsStatic().
		Position(ui.Dpi(leftPad, 304)).
		Size(ui.DpiX(rightEdge-leftPad), 2).
		Layout(ui.LAY_RESIZE_HOLD).
		CtrlStyle(co.SS_ETCHEDHORZ))

	ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("输出格式").
		Position(ui.Dpi(contentLeft, 328)))

	app.formatCombo = ui.NewComboBox(app.wnd, ui.OptsComboBox().
		Position(ui.Dpi(fieldLeft, 324)).
		Width(ui.DpiX(130)).
		CtrlStyle(co.CBS_DROPDOWNLIST))
	app.formatCombo.AddItem("all", "epub", "mobi", "azw3")
	app.formatCombo.SelectIndex(0)
	app.formatCombo.On().CbnSelChange(func() {
		app.persistConfig()
	})

	app.btnMore = ui.NewButton(app.wnd, ui.OptsButton().
		Text("更多选项").
		Position(ui.Dpi(contentLeft, 366)).
		Width(ui.DpiX(144)).
		Height(ui.DpiY(30)))
	app.btnMore.On().BnClicked(func() {
		app.openAdvancedOptions()
	})

	btnAISettings := ui.NewButton(app.wnd, ui.OptsButton().
		Text("AI 设置").
		Position(ui.Dpi(contentLeft+156, 366)).
		Width(ui.DpiX(144)).
		Height(ui.DpiY(30)))
	btnAISettings.On().BnClicked(func() {
		app.openAISettings()
	})

	app.btnConvert = ui.NewButton(app.wnd, ui.OptsButton().
		Text("开始转换").
		Position(ui.Dpi(contentLeft, 414)).
		Width(ui.DpiX(144)).
		Height(ui.DpiY(34)))
	app.btnConvert.Hwnd().EnableWindow(false)

	app.btnOpenDir = ui.NewButton(app.wnd, ui.OptsButton().
		Text("打开输出目录").
		Position(ui.Dpi(contentLeft+156, 414)).
		Width(ui.DpiX(144)).
		Height(ui.DpiY(34)))
	app.btnOpenDir.Hwnd().EnableWindow(false)
	app.btnOpenDir.On().BnClicked(func() {
		if app.lastOutDir != "" {
			_ = openFolder(app.lastOutDir)
		}
	})

	app.statusLbl = ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("准备就绪：选择 TXT 文件后即可开始转换。").
		Position(ui.Dpi(contentLeft+320, 422)).
		Size(ui.DpiX(rightEdge-(contentLeft+320)), ui.DpiY(18)).
		Layout(ui.LAY_RESIZE_HOLD).
		CtrlStyle(co.SS_LEFTNOWORDWRAP))

	ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("转换日志").
		Position(ui.Dpi(leftPad, 474)))

	app.headerHint = ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("这里会显示解析进度、输出结果和错误信息。").
		Position(ui.Dpi(rightEdge-230, 474)).
		Size(ui.DpiX(230), ui.DpiY(18)).
		Layout(ui.LAY_MOVE_HOLD).
		CtrlStyle(co.SS_LEFTNOWORDWRAP))

	ui.NewStatic(app.wnd, ui.OptsStatic().
		Position(ui.Dpi(leftPad, 500)).
		Size(ui.DpiX(rightEdge-leftPad), 2).
		Layout(ui.LAY_RESIZE_HOLD).
		CtrlStyle(co.SS_ETCHEDHORZ))

	app.txtLog = ui.NewEdit(app.wnd, ui.OptsEdit().
		Position(ui.Dpi(leftPad, 516)).
		Width(ui.DpiX(rightEdge-leftPad)).
		Height(ui.DpiY(306)).
		Layout(ui.LAY_RESIZE_RESIZE).
		CtrlStyle(co.ES_MULTILINE|co.ES_READONLY|co.ES_AUTOVSCROLL|co.ES_WANTRETURN))
	setCueBanner(app.txtLog.Hwnd(), "转换开始后，这里会实时显示处理日志")

	app.btnConvert.On().BnClicked(func() {
		app.startConvert()
	})

	app.wnd.On().WmDropFiles(func(p ui.WmDropFiles) {
		files, err := p.HDrop().DragQueryFile()
		if err != nil || len(files) == 0 {
			return
		}
		if txt := firstTxtFromDrop(files); txt != "" {
			app.setTxtPath(txt)
		}
	})

	app.applyConfig(loadGUIConfig())
	return app
}

func (app *appWindow) applyConfig(cfg guiConfig) {
	if cfg.TxtFile != "" {
		if _, err := os.Stat(cfg.TxtFile); err == nil {
			app.txtFile.SetText(cfg.TxtFile)
		}
	}
	if cfg.CoverFile != "" {
		if _, err := os.Stat(cfg.CoverFile); err == nil {
			app.coverFile.SetText(cfg.CoverFile)
		}
	}
	if cfg.Author != "" {
		app.authorEdit.SetText(cfg.Author)
	}
	if cfg.FormatIndex >= 0 && cfg.FormatIndex < 4 {
		app.formatCombo.SelectIndex(cfg.FormatIndex)
	}
	app.dedupTitle = cfg.Dedup
	app.tipsEnabled = cfg.Tips
	app.quoteFix = cfg.Quotes
	app.matchRule = strings.TrimSpace(cfg.Match)
	app.volumeRule = strings.TrimSpace(cfg.VolumeMatch)
	app.aiConfig = cfg.AI
	app.onTxtPathChanged()
}

func (app *appWindow) setTxtPath(path string) {
	app.txtFile.SetText(path)
	app.onTxtPathChanged()
	app.persistConfig()
}

func (app *appWindow) updateBooknamePreview(txtPath string) {
	txtPath = strings.TrimSpace(txtPath)
	if txtPath == "" {
		_ = app.booknameLbl.Hwnd().SetWindowText("书名将在选择 TXT 后显示")
		return
	}
	name, _ := kafcli.FilenameMeta(txtPath)
	if name == "" {
		_ = app.booknameLbl.Hwnd().SetWindowText("未能从文件名中识别书名，可直接继续转换")
		return
	}
	_ = app.booknameLbl.Hwnd().SetWindowText(name)
}

func (app *appWindow) onTxtPathChanged() {
	path := strings.TrimSpace(app.txtFile.Text())
	app.btnConvert.Hwnd().EnableWindow(path != "" && !app.converting)
	app.updateBooknamePreview(path)
	if path == "" {
		_ = app.statusLbl.Hwnd().SetWindowText("准备就绪：选择 TXT 文件后即可开始转换。")
	} else {
		_ = app.statusLbl.Hwnd().SetWindowText("已选择 TXT，确认参数后可开始转换。")
	}

	if strings.TrimSpace(app.coverFile.Text()) == "" {
		if auto := findCover(path); auto != "" {
			app.coverFile.SetText(auto)
		}
	}
	if strings.TrimSpace(app.authorEdit.Text()) == "" {
		if _, author := kafcli.FilenameMeta(path); author != "" {
			app.authorEdit.SetText(author)
		}
	}
}

func (app *appWindow) persistConfig() {
	saveGUIConfig(app.currentConfig())
}

func (app *appWindow) currentConfig() guiConfig {
	return guiConfig{
		TxtFile:     strings.TrimSpace(app.txtFile.Text()),
		CoverFile:   strings.TrimSpace(app.coverFile.Text()),
		Author:      strings.TrimSpace(app.authorEdit.Text()),
		FormatIndex: app.formatCombo.SelectedIndex(),
		Match:       app.matchRule,
		VolumeMatch: app.volumeRule,
		Dedup:       app.dedupTitle,
		Tips:        app.tipsEnabled,
		Quotes:      app.quoteFix,
		AI:          app.aiConfig,
	}
}

func (app *appWindow) guiOptions() kafcli.GUIOptions {
	formats := []string{"all", "epub", "mobi", "azw3"}
	format := "all"
	idx := app.formatCombo.SelectedIndex()
	if idx >= 0 && idx < len(formats) {
		format = formats[idx]
	}
	opts := kafcli.GUIOptions{
		Filename:        strings.TrimSpace(app.txtFile.Text()),
		Cover:           strings.TrimSpace(app.coverFile.Text()),
		Author:          strings.TrimSpace(app.authorEdit.Text()),
		Format:          format,
		Match:           app.matchRule,
		VolumeMatch:     app.volumeRule,
		DedupTitle:      app.dedupTitle,
		Tips:            app.tipsEnabled,
		NormalizeQuotes: app.quoteFix,
	}
	// AI：保留用户意图(Enabled)；是否真正调用由核心库依据 Client.Ready() 决定并打日志。
	cfg := app.aiConfig
	if cfg.Enabled {
		client := ai.NewClient(ai.ClientConfig{
			BaseURL: cfg.BaseURL,
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
		})
		opts.AI = kafcli.AIRefineOptions{
			Enabled:      true,
			Client:       client,
			SampleChars:  cfg.SampleChars,
			DoStructure:  cfg.Tasks.Structure,
			DoTypography: cfg.Tasks.Typography,
			DoNoise:      cfg.Tasks.Noise,
			DoMetadata:   cfg.Tasks.Metadata,
		}
	}
	return opts
}

func (app *appWindow) appendLog(chunk string) {
	text := app.logBuf.append(chunk)
	app.txtLog.SetText(text)
}

func (app *appWindow) openAdvancedOptions() {
	modal := ui.NewModal(app.wnd, ui.OptsModal().
		Title("更多选项").
		Size(ui.DpiX(640), ui.DpiY(340)).
		ClassIconId(1))

	ui.NewStatic(modal, ui.OptsStatic().
		Text("这些选项更适合针对特定书源单独调整。").
		Position(ui.Dpi(20, 18)))

	chkDedup := ui.NewCheckBox(modal, ui.OptsCheckBox().
		Text("合并重复目录行").
		Position(ui.Dpi(20, 56)))
	chkDedup.SetCheck(app.dedupTitle)

	chkTips := ui.NewCheckBox(modal, ui.OptsCheckBox().
		Text("添加制作说明").
		Position(ui.Dpi(220, 56)))
	chkTips.SetCheck(app.tipsEnabled)

	chkQuotes := ui.NewCheckBox(modal, ui.OptsCheckBox().
		Text("对话引号优化（「」→ “”）").
		Position(ui.Dpi(20, 92)))
	chkQuotes.SetCheck(app.quoteFix)

	ui.NewStatic(modal, ui.OptsStatic().
		Text("章节匹配规则").
		Position(ui.Dpi(20, 138)))

	matchEdit := ui.NewEdit(modal, ui.OptsEdit().
		Text(app.matchRule).
		Position(ui.Dpi(132, 134)).
		Width(ui.DpiX(468)).
		Height(ui.DpiY(28)))
	setCueBanner(matchEdit.Hwnd(), "可选：自定义章节匹配正则；留空时自动识别")

	ui.NewStatic(modal, ui.OptsStatic().
		Text("卷匹配规则").
		Position(ui.Dpi(20, 182)))

	volumeEdit := ui.NewEdit(modal, ui.OptsEdit().
		Text(app.volumeRule).
		Position(ui.Dpi(132, 178)).
		Width(ui.DpiX(468)).
		Height(ui.DpiY(28)))
	setCueBanner(volumeEdit.Hwnd(), "可选：自定义卷匹配正则；填 false 可禁用卷识别")

	btnCancel := ui.NewButton(modal, ui.OptsButton().
		Text("取消").
		Position(ui.Dpi(364, 250)).
		Width(ui.DpiX(108)).
		Height(ui.DpiY(32)))
	btnCancel.On().BnClicked(func() {
		_, _ = modal.Hwnd().SendMessage(co.WM_CLOSE, 0, 0)
	})

	btnSave := ui.NewButton(modal, ui.OptsButton().
		Text("保存").
		Position(ui.Dpi(492, 250)).
		Width(ui.DpiX(108)).
		Height(ui.DpiY(32)))
	btnSave.On().BnClicked(func() {
		app.dedupTitle = chkDedup.IsChecked()
		app.tipsEnabled = chkTips.IsChecked()
		app.quoteFix = chkQuotes.IsChecked()
		app.matchRule = strings.TrimSpace(matchEdit.Text())
		app.volumeRule = strings.TrimSpace(volumeEdit.Text())
		app.persistConfig()
		_, _ = modal.Hwnd().SendMessage(co.WM_CLOSE, 0, 0)
	})

	modal.ShowModal()
}

// openAISettings 弹出 AI 后处理配置窗口：总开关、连接参数、4 项任务勾选。
func (app *appWindow) openAISettings() {
	modal := ui.NewModal(app.wnd, ui.OptsModal().
		Title("AI 优化设置").
		Size(ui.DpiX(640), ui.DpiY(420)).
		ClassIconId(1))

	ui.NewStatic(modal, ui.OptsStatic().
		Text("AI 用于章节结构/排版/噪音/简介的后处理；默认关闭，离线不影响转换。").
		Position(ui.Dpi(20, 18)))

	chkEnabled := ui.NewCheckBox(modal, ui.OptsCheckBox().
		Text("启用 AI 后处理").
		Position(ui.Dpi(20, 50)))
	chkEnabled.SetCheck(app.aiConfig.Enabled)

	ui.NewStatic(modal, ui.OptsStatic().
		Text("Base URL").
		Position(ui.Dpi(20, 86)))
	editBaseURL := ui.NewEdit(modal, ui.OptsEdit().
		Text(app.aiConfig.BaseURL).
		Position(ui.Dpi(132, 82)).
		Width(ui.DpiX(468)).
		Height(ui.DpiY(28)))
	setCueBanner(editBaseURL.Hwnd(), "https://api.deepseek.com/v1（留空用 OpenAI 官方）")

	ui.NewStatic(modal, ui.OptsStatic().
		Text("Model").
		Position(ui.Dpi(20, 122)))
	editModel := ui.NewEdit(modal, ui.OptsEdit().
		Text(app.aiConfig.Model).
		Position(ui.Dpi(132, 118)).
		Width(ui.DpiX(468)).
		Height(ui.DpiY(28)))
	setCueBanner(editModel.Hwnd(), "例 deepseek-chat / gpt-4o-mini")

	ui.NewStatic(modal, ui.OptsStatic().
		Text("API Key").
		Position(ui.Dpi(20, 158)))
	editAPIKey := ui.NewEdit(modal, ui.OptsEdit().
		Text(app.aiConfig.APIKey).
		Position(ui.Dpi(132, 154)).
		Width(ui.DpiX(396)).
		Height(ui.DpiY(28)).
		CtrlStyle(co.ES_AUTOHSCROLL))
	setCueBanner(editAPIKey.Hwnd(), "sk-...（DPAPI 加密存储，仅本机可用）")
	// 默认密码遮罩：发 EM_SETPASSWORDCHAR，用 ● 遮蔽。
	setPasswordMask(editAPIKey.Hwnd(), true)

	btnToggleKey := ui.NewButton(modal, ui.OptsButton().
		Text("显示").
		Position(ui.Dpi(536, 154)).
		Width(ui.DpiX(64)).
		Height(ui.DpiY(28)))
	keyMasked := true
	btnToggleKey.On().BnClicked(func() {
		keyMasked = !keyMasked
		setPasswordMask(editAPIKey.Hwnd(), keyMasked)
		btnToggleKey.SetText(ternary(keyMasked, "显示", "隐藏"))
	})

	ui.NewStatic(modal, ui.OptsStatic().
		Text("抽样上限").
		Position(ui.Dpi(20, 194)))
	editSampleChars := ui.NewEdit(modal, ui.OptsEdit().
		Text(fmt.Sprintf("%d", app.aiConfig.SampleChars)).
		Position(ui.Dpi(132, 190)).
		Width(ui.DpiX(120)).
		Height(ui.DpiY(28)))
	setCueBanner(editSampleChars.Hwnd(), "正文抽样字符数，0=仅分析目录")

	chkStructure := ui.NewCheckBox(modal, ui.OptsCheckBox().
		Text("章节结构分析").
		Position(ui.Dpi(20, 230)))
	chkStructure.SetCheck(app.aiConfig.Tasks.Structure)

	chkTypography := ui.NewCheckBox(modal, ui.OptsCheckBox().
		Text("排版修正").
		Position(ui.Dpi(220, 230)))
	chkTypography.SetCheck(app.aiConfig.Tasks.Typography)

	chkNoise := ui.NewCheckBox(modal, ui.OptsCheckBox().
		Text("噪音清理").
		Position(ui.Dpi(20, 262)))
	chkNoise.SetCheck(app.aiConfig.Tasks.Noise)

	chkMetadata := ui.NewCheckBox(modal, ui.OptsCheckBox().
		Text("生成简介").
		Position(ui.Dpi(220, 262)))
	chkMetadata.SetCheck(app.aiConfig.Tasks.Metadata)

	btnCancel := ui.NewButton(modal, ui.OptsButton().
		Text("取消").
		Position(ui.Dpi(364, 330)).
		Width(ui.DpiX(108)).
		Height(ui.DpiY(32)))
	btnCancel.On().BnClicked(func() {
		_, _ = modal.Hwnd().SendMessage(co.WM_CLOSE, 0, 0)
	})

	btnSave := ui.NewButton(modal, ui.OptsButton().
		Text("保存").
		Position(ui.Dpi(492, 330)).
		Width(ui.DpiX(108)).
		Height(ui.DpiY(32)))
	btnSave.On().BnClicked(func() {
		app.aiConfig.Enabled = chkEnabled.IsChecked()
		app.aiConfig.BaseURL = strings.TrimSpace(editBaseURL.Text())
		app.aiConfig.Model = strings.TrimSpace(editModel.Text())
		app.aiConfig.APIKey = strings.TrimSpace(editAPIKey.Text())
		app.aiConfig.SampleChars = kafcli.ParseInt(strings.TrimSpace(editSampleChars.Text()))
		app.aiConfig.Tasks.Structure = chkStructure.IsChecked()
		app.aiConfig.Tasks.Typography = chkTypography.IsChecked()
		app.aiConfig.Tasks.Noise = chkNoise.IsChecked()
		app.aiConfig.Tasks.Metadata = chkMetadata.IsChecked()
		app.persistConfig()
		_, _ = modal.Hwnd().SendMessage(co.WM_CLOSE, 0, 0)
	})

	modal.ShowModal()
}

func (app *appWindow) startConvert() {
	txtPath := strings.TrimSpace(app.txtFile.Text())
	if txtPath == "" {
		app.wnd.Hwnd().MessageBox("请先选择 TXT 文件", "提示", co.MB_ICONWARNING)
		return
	}
	if app.converting {
		return
	}

	app.converting = true
	app.lastOutDir = filepath.Dir(txtPath)
	app.btnConvert.SetText("转换中...")
	app.btnConvert.Hwnd().EnableWindow(false)
	app.btnOpenDir.Hwnd().EnableWindow(false)
	_ = app.statusLbl.Hwnd().SetWindowText("正在转换，请稍候...")
	app.logBuf.reset()
	app.txtLog.SetText("")

	opts := app.guiOptions()
	go func() {
		err := runConvert(opts, func(line string) {
			app.wnd.UiThread(func() {
				app.appendLog(line)
			})
		})
		app.wnd.UiThread(func() {
			app.converting = false
			app.btnConvert.SetText("开始转换")
			app.onTxtPathChanged()
			app.persistConfig()

			if err != nil {
				_ = app.statusLbl.Hwnd().SetWindowText("转换失败，请检查日志或参数设置。")
				app.wnd.Hwnd().MessageBox(err.Error(), "转换失败", co.MB_ICONERROR)
				return
			}

			app.btnOpenDir.Hwnd().EnableWindow(true)
			_ = app.statusLbl.Hwnd().SetWindowText("转换完成，可以直接打开输出目录。")
			id, _ := app.wnd.Hwnd().MessageBox(
				"电子书转换完成！\n是否打开输出目录？",
				"完成",
				co.MB_YESNO|co.MB_ICONINFORMATION,
			)
			if id == co.ID_YES && app.lastOutDir != "" {
				_ = openFolder(app.lastOutDir)
			}
		})
	}()
}

func setCueBanner(hWnd win.HWND, text string) {
	if text == "" || hWnd == 0 {
		return
	}
	ptr, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return
	}
	_, _ = hWnd.SendMessage(co.EM_SETCUEBANNER, 0, win.LPARAM(uintptr(unsafe.Pointer(ptr))))
}

// setPasswordMask 开关单行 Edit 的密码遮罩。
// 开启时发 EM_SETPASSWORDCHAR 设置遮蔽字符 ●，并用 user32 直接改 ES_PASSWORD 样式；
// 关闭时移除样式并清空密码字符，恢复明文。
// 直接调用 user32 而非 windigo 封装，因为 windigo 的 HWND 没有 Invalidate/SetStyle 之类便捷方法。
var (
	modUser32           = syscall.NewLazyDLL("user32.dll")
	procGetWindowLongW  = modUser32.NewProc("GetWindowLongW")
	procSetWindowLongW  = modUser32.NewProc("SetWindowLongW")
)

const (
	emSetPasswordChar = 0x00CC
	esPassword        = 0x0020
	swpFrameChanged   = 0x0020
	swpNoMove         = 0x0002
	swpNoSize         = 0x0001
	swpNoZOrder       = 0x0004
)

func setPasswordMask(hWnd win.HWND, mask bool) {
	if hWnd == 0 {
		return
	}
	// GWL_STYLE = -16，用 ^uintptr(0) - 15 计算出无符号表示，规避负常量溢出。
	gwlStyle := ^uintptr(0) - 15
	h := uintptr(hWnd)
	style, _, _ := procGetWindowLongW.Call(h, gwlStyle)
	if mask {
		_, _ = hWnd.SendMessage(emSetPasswordChar, win.WPARAM('●'), 0)
		procSetWindowLongW.Call(h, gwlStyle, style|esPassword)
	} else {
		procSetWindowLongW.Call(h, gwlStyle, style & ^uintptr(esPassword))
		_, _ = hWnd.SendMessage(emSetPasswordChar, 0, 0)
	}
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
