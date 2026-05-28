//go:build windows

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
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

	authorFromFilename = regexp.MustCompile(`《(.*)》.*作者[：:](.*)\.txt$`)
)

type appWindow struct {
	wnd         *ui.Main
	txtFile     *ui.Edit
	coverFile   *ui.Edit
	authorEdit  *ui.Edit
	formatCombo *ui.ComboBox
	chkDedup    *ui.CheckBox
	chkTips     *ui.CheckBox
	chkQuotes   *ui.CheckBox
	btnConvert  *ui.Button
	txtLog      *ui.Edit
	converting  bool
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
				Size(ui.Dpi(580, 560)),
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
		app.txtFile.SetText(path)
		app.onTxtPathChanged()
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
		}
	})

	ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("作者:").
		Position(ui.Dpi(10, 88)))

	app.authorEdit = ui.NewEdit(app.wnd, ui.OptsEdit().
		Position(ui.Dpi(80, 85)).
		Width(ui.DpiX(360)))

	ui.NewStatic(app.wnd, ui.OptsStatic().
		Text("输出格式:").
		Position(ui.Dpi(10, 123)))

	app.formatCombo = ui.NewComboBox(app.wnd, ui.OptsComboBox().
		Position(ui.Dpi(80, 120)).
		Width(ui.DpiX(120)).
		CtrlStyle(co.CBS_DROPDOWNLIST))
	app.formatCombo.AddItem("all", "epub", "mobi", "azw3")
	app.formatCombo.SelectIndex(0)

	app.chkDedup = ui.NewCheckBox(app.wnd, ui.OptsCheckBox().
		Text("合并重复目录行").
		Position(ui.Dpi(10, 158)).
		State(co.BST_CHECKED))
	app.chkTips = ui.NewCheckBox(app.wnd, ui.OptsCheckBox().
		Text("添加制作说明").
		Position(ui.Dpi(200, 158)).
		State(co.BST_CHECKED))
	app.chkQuotes = ui.NewCheckBox(app.wnd, ui.OptsCheckBox().
		Text("对话引号优化（「」→ “”）").
		Position(ui.Dpi(10, 188)))

	app.btnConvert = ui.NewButton(app.wnd, ui.OptsButton().
		Text("开始转换").
		Position(ui.Dpi(10, 222)).
		Width(ui.DpiX(120)))
	app.btnConvert.Hwnd().EnableWindow(false)

	app.txtLog = ui.NewEdit(app.wnd, ui.OptsEdit().
		Position(ui.Dpi(10, 258)).
		Width(ui.DpiX(540)).
		Height(ui.DpiY(270)).
		Layout(ui.LAY_RESIZE_RESIZE).
		CtrlStyle(co.ES_MULTILINE|co.ES_READONLY|co.ES_AUTOVSCROLL|co.ES_WANTRETURN))

	app.btnConvert.On().BnClicked(func() {
		app.startConvert()
	})

	return app
}

func (app *appWindow) onTxtPathChanged() {
	path := strings.TrimSpace(app.txtFile.Text())
	app.btnConvert.Hwnd().EnableWindow(path != "" && !app.converting)

	if strings.TrimSpace(app.coverFile.Text()) == "" {
		if auto := findCover(path); auto != "" {
			app.coverFile.SetText(auto)
		}
	}
	if strings.TrimSpace(app.authorEdit.Text()) == "" {
		if author := authorFromTxtPath(path); author != "" {
			app.authorEdit.SetText(author)
		}
	}
}

func authorFromTxtPath(path string) string {
	base := filepath.Base(path)
	m := authorFromFilename.FindStringSubmatch(base)
	if len(m) >= 3 {
		return strings.TrimSpace(m[2])
	}
	return ""
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
	app.btnConvert.SetText("转换中...")
	app.btnConvert.Hwnd().EnableWindow(false)
	app.txtLog.SetText("")

	opts := app.guiOptions()
	go func() {
		err := runConvert(opts, func(line string) {
			app.wnd.UiThread(func() {
				app.txtLog.SetText(app.txtLog.Text() + line)
			})
		})
		app.wnd.UiThread(func() {
			app.converting = false
			app.btnConvert.SetText("开始转换")
			app.onTxtPathChanged()
			if err != nil {
				app.wnd.Hwnd().MessageBox(err.Error(), "转换失败", co.MB_ICONERROR)
				return
			}
			app.wnd.Hwnd().MessageBox("电子书转换完成！", "完成", co.MB_ICONINFORMATION)
		})
	}()
}

func pickFile(parent win.HWND, title string, filters []win.COMDLG_FILTERSPEC) (string, bool) {
	releaser := win.NewOleReleaser()
	defer releaser.Release()

	var fod *win.IFileOpenDialog
	if err := win.CoCreateInstance(releaser, &co.CLSID_FileOpenDialog, nil, co.CLSCTX_INPROC_SERVER, &fod); err != nil {
		return "", false
	}

	defOpts, _ := fod.GetOptions()
	_ = fod.SetOptions(defOpts | co.FOS_FORCEFILESYSTEM | co.FOS_FILEMUSTEXIST)
	_ = fod.SetFileTypes(filters)
	_ = fod.SetFileTypeIndex(1)
	_ = fod.SetTitle(title)

	ok, _ := fod.Show(parent)
	if !ok {
		return "", false
	}

	item, err := fod.GetResult(releaser)
	if err != nil {
		return "", false
	}
	path, err := item.GetDisplayName(co.SIGDN_FILESYSPATH)
	if err != nil {
		return "", false
	}
	return path, true
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

func runConvert(opts kafcli.GUIOptions, appendLog func(string)) error {
	book := kafcli.NewBookGUI(opts)

	oldStdout := os.Stdout
	r, wPipe, err := os.Pipe()
	if err != nil {
		return err
	}
	os.Stdout = wPipe

	done := make(chan struct{})
	go func() {
		buf := make([]byte, 4096)
		for {
			n, readErr := r.Read(buf)
			if n > 0 {
				appendLog(string(buf[:n]))
			}
			if readErr != nil {
				if readErr != io.EOF {
					appendLog(readErr.Error())
				}
				break
			}
		}
		close(done)
	}()

	runErr := kafcli.Run(book, version, secret, measurement)
	wPipe.Close()
	os.Stdout = oldStdout
	<-done

	if runErr != nil {
		return fmt.Errorf("转换失败: %w", runErr)
	}
	return nil
}
