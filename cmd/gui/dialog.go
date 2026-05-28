//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rodrigocfd/windigo/co"
	"github.com/rodrigocfd/windigo/win"
)

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

func openFolder(dir string) error {
	dir = filepath.Clean(dir)
	return exec.Command("explorer.exe", dir).Start()
}

func firstTxtFromDrop(files []string) string {
	for _, f := range files {
		if strings.EqualFold(filepath.Ext(f), ".txt") {
			return f
		}
	}
	return ""
}
