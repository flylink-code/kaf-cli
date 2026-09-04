# 项目目录结构

```
kaf-cli/
├── cmd/                    # 可执行程序入口
│   ├── cli.go              # 命令行 kaf-cli
│   └── gui-wails/          # Windows 图形界面（Wails v2 + Web 前端）
├── internal/kafcli/        # 核心转换库（解析、epub/mobi/azw3）
├── lib/                    # CGO 导出（Android 等）
├── assets/                 # 图标等资源
├── scripts/                # 构建与安装脚本
│   ├── build.ps1 / build.sh
│   └── 注册右键菜单.ps1
├── build/                  # 本地编译输出（git 忽略）
│   └── windows-amd64/      # kaf-cli.exe、kaf-cli-wails.exe
├── docs/                   # 文档与截图
│   ├── images/
│   ├── README_wasi.md
│   └── dev/                # 开发笔记（可选，已忽略）
├── examples/book/          # 样例小说（git 忽略）
├── .github/workflows/      # CI 发布
├── build.ps1 / build.sh    # 根目录快捷入口 → scripts/
├── kaf-cli.exe.manifest    # Windows 清单（CI 嵌入图标）
├── README.md
├── go.mod
└── LICENSE
```
