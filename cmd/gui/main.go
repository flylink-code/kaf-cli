//go:build windows

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

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
	txtFile      *ui.Edit
	coverFile    *ui.Edit
	authorEdit   *ui.Edit
	booknameLbl  *ui.Static
	formatCombo  *ui.ComboBox
	chkDedup     *ui.CheckBox
	chkTips      *ui.CheckBox
	chkQuotes    *ui.CheckBox
	btnConvert   *ui.Button
	btnOpenDir   *ui.Button
	txtLog       *ui.Edit
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
	app := &appWindow{
		wnd: ui.NewMain(
			ui.OptsMain().
				Title("kaf-cli 电子书转换").
				Size(ui.Dpi(580, 600)).
				ClassIconId(1).
				DropFiles(true),
		),
	}

	ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("TXT 文件:").
		Position(ui.Dpi(10, 18)))

	app.txtFile = ui.NewEdit(app.wnd, ui.OptsEdit().
		Position(ui.Dpi(80, 15)).
		Width(ui.DpiX(360)))

	btnBrowseTxt := ui.NewButton(app.wnd, ui.OptsButton().
		Text("浏览...").
		Position(ui.Dpi(450, 13)).
		Width(ui.DpiX(100)))
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
		Text("封面图片:").
		Position(ui.Dpi(10, 53)))

	app.coverFile = ui.NewEdit(app.wnd, ui.OptsEdit().
		Position(ui.Dpi(80, 50)).
		Width(ui.DpiX(360)))

	btnBrowseCover := ui.NewButton(app.wnd, ui.OptsButton().
		Text("浏览...").
		Position(ui.Dpi(450, 48)).
		Width(ui.DpiX(100)))
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
		Text("作者:").
		Position(ui.Dpi(10, 88)))

	app.authorEdit = ui.NewEdit(app.wnd, ui.OptsEdit().
		Position(ui.Dpi(80, 85)).
		Width(ui.DpiX(360)))

	ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("书名:").
		Position(ui.Dpi(10, 123)))

	app.booknameLbl = ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("（选择 TXT 后显示）").
		Position(ui.Dpi(80, 120)))

	ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("输出格式:").
		Position(ui.Dpi(10, 158)))

	app.formatCombo = ui.NewComboBox(app.wnd, ui.OptsComboBox().
		Position(ui.Dpi(80, 155)).
		Width(ui.DpiX(120)).
		CtrlStyle(co.CBS_DROPDOWNLIST))
	app.formatCombo.AddItem("all", "epub", "mobi", "azw3")
	app.formatCombo.SelectIndex(0)

	app.chkDedup = ui.NewCheckBox(app.wnd, ui.OptsCheckBox().
		Text("合并重复目录行").
		Position(ui.Dpi(10, 193)).
		State(co.BST_CHECKED))
	app.chkTips = ui.NewCheckBox(app.wnd, ui.OptsCheckBox().
		Text("添加制作说明").
		Position(ui.Dpi(200, 193)).
		State(co.BST_CHECKED))
	app.chkQuotes = ui.NewCheckBox(app.wnd, ui.OptsCheckBox().
		Text("对话引号优化（「」→ “”）").
		Position(ui.Dpi(10, 223)))

	app.btnConvert = ui.NewButton(app.wnd, ui.OptsButton().
		Text("开始转换").
		Position(ui.Dpi(10, 256)).
		Width(ui.DpiX(120)))
	app.btnConvert.Hwnd().EnableWindow(false)

	app.btnOpenDir = ui.NewButton(app.wnd, ui.OptsButton().
		Text("打开输出目录").
		Position(ui.Dpi(140, 256)).
		Width(ui.DpiX(120)))
	app.btnOpenDir.Hwnd().EnableWindow(false)
	app.btnOpenDir.On().BnClicked(func() {
		if app.lastOutDir != "" {
			_ = openFolder(app.lastOutDir)
		}
	})

	app.txtLog = ui.NewEdit(app.wnd, ui.OptsEdit().
		Position(ui.Dpi(10, 292)).
		Width(ui.DpiX(540)).
		Height(ui.DpiY(280)).
		Layout(ui.LAY_RESIZE_RESIZE).
		CtrlStyle(co.ES_MULTILINE|co.ES_READONLY|co.ES_AUTOVSCROLL|co.ES_WANTRETURN))

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
			app.updateBooknamePreview(cfg.TxtFile)
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
	app.chkDedup.SetCheck(cfg.Dedup)
	app.chkTips.SetCheck(cfg.Tips)
	app.chkQuotes.SetCheck(cfg.Quotes)
	app.onTxtPathChanged()
}

func (app *appWindow) setTxtPath(path string) {
	app.txtFile.SetText(path)
	app.onTxtPathChanged()
	app.persistConfig()
}

func (app *appWindow) updateBooknamePreview(txtPath string) {
	name, _ := kafcli.FilenameMeta(txtPath)
	if name == "" {
		app.booknameLbl.SetTextAndResize("（无法识别书名）")
		return
	}
	app.booknameLbl.SetTextAndResize(name)
}

func (app *appWindow) onTxtPathChanged() {
	path := strings.TrimSpace(app.txtFile.Text())
	app.btnConvert.Hwnd().EnableWindow(path != "" && !app.converting)
	app.updateBooknamePreview(path)

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
		Dedup:       app.chkDedup.IsChecked(),
		Tips:        app.chkTips.IsChecked(),
		Quotes:      app.chkQuotes.IsChecked(),
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
		DedupTitle:      app.chkDedup.IsChecked(),
		Tips:            app.chkTips.IsChecked(),
		NormalizeQuotes: app.chkQuotes.IsChecked(),
	}
}

func (app *appWindow) appendLog(chunk string) {
	text := app.logBuf.append(chunk)
	app.txtLog.SetText(text)
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
				app.wnd.Hwnd().MessageBox(err.Error(), "转换失败", co.MB_ICONERROR)
				return
			}

			app.btnOpenDir.Hwnd().EnableWindow(true)
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
