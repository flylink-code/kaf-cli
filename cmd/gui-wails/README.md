# Wails Prototype

这是一个保留现有 `cmd/gui` 的前提下，新增的 Wails 原型入口。

当前约束：

- Go 文件挂在自定义构建标签 `wailsgui` 下，不会影响现有 `go build ./...`
- 需要先安装 Wails CLI，并把 `github.com/wailsapp/wails/v2` 拉到本地
- Windows 运行时依赖 WebView2 Runtime

建议流程：

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
go get github.com/wailsapp/wails/v2@latest
wails doctor
cd .\cmd\gui-wails
wails dev -tags wailsgui
```

如果本机缺少 WebView2 Runtime，可按 Wails 官方文档处理：

- Wails Windows guide: https://wails.io/docs/guides/windows/
- Microsoft WebView2 distribution: https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/distribution
