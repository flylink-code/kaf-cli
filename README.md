## kaf-cli

> 把 txt 文本转成 epub / mobi / azw3 电子书的命令行工具

本项目 fork 自 [ystyle/kaf-cli](https://github.com/ystyle/kaf-cli)，在保留原有功能的基础上增加了 Windows 图形界面与章节目录优化等改进。

### 本 fork 新增

- **Windows 图形版 `kaf-cli-gui.exe`**：窗口内选择 TXT 与封面图片（png/jpg），一键转换
- **章节目录去重 `-dedup-title`**：合并「目录行 + 正文标题行」重复章节（默认开启）
- **构建脚本优化**：`build.ps1` 支持一键编译 CLI / GUI，失败时明确报错

### 功能

- 傻瓜操作模式（把 txt 文件拖到 `kaf-cli.exe` 上面自动转换）
- Windows 图形界面（`kaf-cli-gui.exe`）
- 自定义封面
- 支持生成 Orly 风格的书籍封面
- 自动识别书名和章节
- 自动识别字符编码（自动解决中文乱码）
- 合并连续重复的章节目录行（`-dedup-title`）
- 自定义章节标题识别规则
- 自定义卷的标题识别规则
- 自动给章节正文生成加粗居中的标题
- 自定义标题对齐方式
- 段落自动识别、自动缩进
- 自定义段落缩进字数、段落间距、行间距
- 自定义书籍语言
- epub 格式支持嵌入字体
- 知轩藏书格式文件名会自动提取书名和作者，例：`《希灵帝国》（校对版全本）作者：远瞳.txt`
- 超快速（130 章/s 以上速度，4000 章 30s 不到）
- 自动转为 mobi / azw3 格式

### 下载

- **本 fork 发布页**：[flylink-code/kaf-cli Releases](https://github.com/flylink-code/kaf-cli/releases/latest)
- 上游原版：[ystyle/kaf-cli Releases](https://github.com/ystyle/kaf-cli/releases/latest)
- 手机版 kaf：[Github 下载](https://github.com/ystyle/kaf-cli/releases/tag/android)
- 电脑版 wifi 传书 kaf-wifi：[Github 下载](https://github.com/ystyle/kaf-wifi/releases/latest)

Windows 压缩包内包含：

| 文件 | 说明 |
|---|---|
| `kaf-cli.exe` | 64 位命令行版（支持拖拽 txt） |
| `kaf-cli-gui.exe` | 64 位图形界面版（仅 Windows） |
| `kindlegen.exe` | mobi 转换工具（如有） |

### 本地编译

需要 Go 1.21+。

```powershell
# Windows
.\build.ps1
```

编译产物：

- `kaf-cli.exe` — 64 位 CLI
- `kaf-cli-gui.exe` — 64 位 GUI（Windows 专用，无控制台窗口）
- `kaf-cli_32.exe` — 32 位 CLI

> GUI 依赖 [windigo](https://github.com/rodrigocfd/windigo)，仅提供 64 位版本。

### 使用方法

#### 图形界面（Windows）

1. 运行 `kaf-cli-gui.exe`
2. 点击「浏览...」选择 TXT 小说文件
3. （可选）选择封面 png/jpg；若同目录有同名图片会自动填入
4. 点击「开始转换」，完成后在 TXT 同目录生成 epub / mobi / azw3

#### 拖拽 / 命令行

1. 解压，把小说直接拖到 `kaf-cli.exe` 文件上面
2. 等转换完，目录下会生成 epub、azw3、mobi 文件
   - mobi 格式需要有 kindlegen 才会生成（windows、mac 版本发布包通常自带）
3. 自定义封面：拖拽模式下，若目录下有 `cover.png` 会自动添加为封面（支持 jpg、png）
4. 其它自定义功能请用命令行模式

### 效果

![效果图片](2021-06-20_12-13-34.png)
![效果图片](2020-01-21_12-02.png)

### 命令行模式参数

```text
Usage of kaf-cli:
  -align string
        标题对齐方式: left、center、righ。环境变量 KAF_CLI_ALIGN 可修改默认值 (default "center")
  -author string
        作者 (default "YSTYLE")
  -bookname string
        书名: 默认为 txt 文件名
  -bottom string
        段落间距(单位可以为 em、px) (default "1em")
  -cover string
        封面图片可为: 本地图片, 和 orly。设置为 orly 时生成 orly 风格的封面, 需要连接网络。(default "cover.png")
  -cover-orly-color string
        orly 封面的主题色, 可以为 1-16 和 hex 格式的颜色代码, 不填时随机
  -cover-orly-idx int
        orly 封面的动物, 可以为 0-41, 不填时随机 (default -1)
  -dedup-title
        合并连续重复的章节目录行（同章号且上一节无正文）(default true)
  -filename string
        txt 文件名
  -font string
        嵌入字体, 之后 epub 的正文都将使用该字体
  -format string
        书籍格式: all、epub、mobi、azw3。环境变量 KAF_CLI_FORMAT 可修改默认值 (default "all")
  -indent uint
        段落缩进字数 (default 2)
  -lang string
        设置语言: en,de,fr,it,es,zh,ja,pt,ru,nl。环境变量 KAF_CLI_LANG 可修改默认值 (default "zh")
  -line-height string
        行高(用于设置行间距, 默认为 1.5rem)
  -match string
        匹配标题的正则表达式, 不写可以自动识别
  -max uint
        标题最大字数 (default 35)
  -out string
        输出文件名，不需要包含格式后缀
  -tips
        添加本软件教程 (default true)
  -unknow-title string
        未知章节默认名称 (default "章节正文")
  -volume-match string
        卷匹配规则, 设置为 false 可以禁用卷识别
```

> PS: 在 darwin(mac、osx) 上 `-tips` 参数要设置为 false：`kaf-cli -filename 小说.txt -tips=0`

### 命令行示例

转换 `全职法师.txt`，并设置作者名为 `乱`：

```shell
# Windows
kaf-cli.exe -author 乱 -filename 全职法师.txt

# 简单模式（等同拖拽）
kaf-cli.exe 全职法师.txt

# 关闭章节目录去重
kaf-cli.exe -filename 全职法师.txt -dedup-title=false
```

### 章节目录去重说明

部分小说每章开头有两行相似标题，例如：

```text
第1章 一章「标题」
　　第1章 一章·「标题」
```

开启 `-dedup-title`（默认）后，会跳过无正文的重复目录行，避免 EPUB 目录章节翻倍。

### 自定义章节匹配规则

规则支持[正则表达式](http://deerchao.net/tutorials/regex/regex.htm)。

```shell
# 章节格式为 第x节
kaf-cli.exe -filename ebook.txt -match "第.{1,8}节"

# 章节格式为 Section 1 ~ Section 100
kaf-cli.exe -filename ebook.txt -match "Section \d+"

# 章节格式为 Chapter xxx
kaf-cli.exe -filename ebook.txt -match "Chapter .{1,8}"
```

### 手动把书转为 kindle 的 mobi 格式

> 新版如果检测到有 kindlegen 程序时会自动转为 mobi

1. 下载 [kindlegen](https://github.com/ystyle/kaf-cli/releases/kindlegen/)（github 备份，官网已经不提供下载）
2. 放到与 `kaf-cli.exe` 同目录，或 PATH 中
3. 也可手动执行：`kindlegen.exe XXXX.epub`

### 致谢

- 原版项目：[ystyle/kaf-cli](https://github.com/ystyle/kaf-cli)
- 教程：[txt 转 epub 和 mobi](https://ystyle.top/2019/12/31/txt-converto-epub-and-mobi/)
