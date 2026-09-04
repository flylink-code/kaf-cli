# kaf-cli GUI (Wails)

这是基于 Wails v2 的 Windows 图形界面版本（`kaf-cli-wails.exe`）。

### 环境要求

- Go 1.22+
- Node.js 18+ / npm
- Wails CLI v2 (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- Windows WebView2 Runtime

### 开发与调试

```powershell
# 安装依赖
cd .\cmd\gui-wails\frontend
npm install

# 启动开发调试模式（热重载）
cd ..
wails dev -tags wailsgui
```

### 构建

根目录一键构建脚本会自动处理前端与后端打包：

```powershell
.\scripts\build.ps1
```

如需单独手动构建：

```powershell
cd .\cmd\gui-wails
wails build -m -tags wailsgui -ldflags "-s -w"
```

构建产物将输出在 `cmd/gui-wails/build/bin/kaf-cli-wails.exe`。
