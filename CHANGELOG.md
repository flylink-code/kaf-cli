# Changelog

## v1.5.1 (2026-05-28)

### 修复

- 输出文件写入 txt 所在目录，不再落到 GUI/CLI 的当前工作目录（exe 目录）

## v1.5.0 (2026-05-28)

### GUI

- 作者、输出格式、合并目录行 / 制作说明 / 引号优化等选项
- 拖拽 TXT 到窗口、书名预览、转换后打开输出目录
- 记住配置（`%AppData%\kaf-cli\gui-config.json`）
- 日志捕获 stderr，512KB 上限；构建时嵌入窗口图标

### 转换

- `-normalize-quotes`：正文 `「」` → `“”`
- 单独一行的 `第一部`、`第三部` 等不作为章节/卷（默认卷规则仅识别「卷」）

### 工程

- 核心库迁至 `internal/kafcli/`
- 文档与脚本：`docs/`、`scripts/`、`examples/`
- 编译产物输出到 `build/` 目录

## v1.4.0

- Windows GUI（Windigo）、`-dedup-title`、统一 `Run()` 入口
