//go:build windows && wailsgui

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const releaseAPIURL = "https://api.github.com/repos/flylink-code/kaf-cli/releases/latest"

type UpdateInfo struct {
	Available   bool   `json:"available"`
	Current     string `json:"current"`
	Latest      string `json:"latest"`
	DownloadURL string `json:"downloadURL"`
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// CheckForUpdate reads the latest stable GitHub release and selects its x64 MSI asset.
func (a *App) CheckForUpdate() (UpdateInfo, error) {
	release, err := latestRelease()
	if err != nil {
		return UpdateInfo{Current: version}, err
	}

	info := UpdateInfo{Current: version, Latest: release.TagName}
	for _, asset := range release.Assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), "-windows-x64.msi") {
			info.DownloadURL = asset.BrowserDownloadURL
			break
		}
	}
	if info.DownloadURL == "" {
		return info, errors.New("最新版本未提供 Windows x64 MSI 安装包")
	}
	info.Available = versionLess(version, release.TagName)
	return info, nil
}

// InstallUpdate downloads the selected MSI and launches Windows Installer after this app exits.
func (a *App) InstallUpdate(downloadURL string) error {
	if !strings.HasPrefix(downloadURL, "https://github.com/") {
		return errors.New("更新包来源无效")
	}

	path, err := downloadMSI(downloadURL)
	if err != nil {
		return err
	}

	// Give Wails time to release the installed executable before MSI replaces it.
	command := fmt.Sprintf(`timeout /t 2 /nobreak >nul & start "" msiexec.exe /i "%s"`, path)
	cmd := exec.Command("cmd.exe", "/c", command)
	cmd.SysProcAttr = hideWindowAttrs()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动安装程序失败: %w", err)
	}
	runtime.Quit(a.ctx)
	return nil
}

func latestRelease() (githubRelease, error) {
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, releaseAPIURL, nil)
	if err != nil {
		return githubRelease{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "kaf-cli-updater")
	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return githubRelease{}, fmt.Errorf("无法连接 GitHub: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return githubRelease{}, fmt.Errorf("GitHub 更新检查失败: %s", response.Status)
	}
	var release githubRelease
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		return githubRelease{}, fmt.Errorf("无法读取更新信息: %w", err)
	}
	if release.TagName == "" {
		return githubRelease{}, errors.New("GitHub 发布版本无效")
	}
	return release, nil
}

func downloadMSI(downloadURL string) (string, error) {
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "kaf-cli-updater")
	client := &http.Client{Timeout: 5 * time.Minute}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("下载更新包失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载更新包失败: %s", response.Status)
	}

	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "kaf-cli", "updates")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "kaf-cli-update.msi")
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(file, response.Body)
	closeErr := file.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return path, nil
}

func versionLess(current, latest string) bool {
	current = strings.TrimPrefix(strings.TrimSpace(current), "v")
	latest = strings.TrimPrefix(strings.TrimSpace(latest), "v")
	if current == "dev" || current == "" {
		return true
	}
	var cMajor, cMinor, cPatch, lMajor, lMinor, lPatch int
	if _, err := fmt.Sscanf(current, "%d.%d.%d", &cMajor, &cMinor, &cPatch); err != nil {
		return false
	}
	if _, err := fmt.Sscanf(latest, "%d.%d.%d", &lMajor, &lMinor, &lPatch); err != nil {
		return false
	}
	return cMajor < lMajor || (cMajor == lMajor && (cMinor < lMinor || (cMinor == lMinor && cPatch < lPatch)))
}
