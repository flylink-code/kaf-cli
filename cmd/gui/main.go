//go:build windows

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	kafcli "github.com/ystyle/kaf-cli/internal/kafcli"
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
	}
}

func (app *appWindow) guiOptions() kafcli.GUIOptions {
	formats := []string{"all", "epub", "mobi", "azw3"}
	format := "all"
	idx := app.formatCombo.SelectedIndex()
	if idx >= 0 && idx < len(formats) {
		format = formats[idx]
	}
	return kafcli.GUIOptions{
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
