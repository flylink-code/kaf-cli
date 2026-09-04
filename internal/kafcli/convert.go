package kafcli

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	_ "golang.org/x/image/webp"
	"github.com/ystyle/kaf-cli/internal/kafcli/ai"
)

type Book struct {
	Filename       string    // 目录
	Bookname       string    // 书名
	Match          string    // 正则
	VolumeMatch    string    // 卷匹配规则
	Author         string    // 作者
	Max            uint      // 标题最大字数
	Indent         uint      // 段落缩进字段
	Align          string    // 标题对齐方式
	UnknowTitle    string    // 未知章节名称
	Cover          string    // 封面图片
	CoverOrlyColor string    // 生成封面图片的颜色
	CoverOrlyIdx   int       // 生成封面图片的动物
	Font           string    // 嵌入字体
	Bottom         string    // 段阿落间距
	LineHeight     string    // 行高
	Tips           bool      // 是否添加教程文本
	Lang           string    // 设置语言
	Out            string    // 输出文件名
	Format         string    // 书籍格式
	DedupTitle      bool      // 合并连续重复的章节目录行
	NormalizeQuotes bool      // 正文「」转为弯引号 “”
	SectionList     []Section // 章节
	// AI 后处理选项；零值表示关闭。仅在 GUI/Wails 流程注入，CLI 不受影响。
	AIOptions  AIRefineOptions
	AISummary  string // AI 生成的简介，供 epub 等格式写入描述字段
	AITags     string // AI 生成的标签
	Decoder        *encoding.Decoder
	PageStylesFile string
	Reg            *regexp.Regexp
	VolumeReg      *regexp.Regexp
	version        string
}

// AIRefineOptions 是 Book 持有的 AI 后处理配置。
// Client 为 nil 或 Enabled=false 时跳过整个 AI 流程。
type AIRefineOptions struct {
	Enabled      bool
	Client       *ai.Client
	SampleChars  int
	DoStructure  bool
	DoTypography bool
	DoNoise      bool
	DoMetadata   bool
}

type Section struct {
	Title    string
	Content  string
	Sections []Section
}

type Converter interface {
	Build(book Book) error
}

const (
	htmlPStart         = `<p class="content">`
	htmlPEnd           = "</p>"
	htmlTitleStart     = `<h3 class="title">`
	mobiTtmlTitleStart = `<h3 style="text-align:%s;">`
	htmlTitleEnd       = "</h3>"
	VolumeMatch        = "^第[0-9一二三四五六七八九十零〇百千两 ]+卷"
	DefaultMatchTips   = "^第[0-9一二三四五六七八九十零〇百千两 ]+[章回节集卷].*|^[零〇一二三四五六七八九十百千两0-9]{1,12}[章回节集卷]·.*|^[Ss]ection.{1,20}$|^[Cc]hapter.{1,20}$|^[Pp]age.{1,20}$|^\\d{1,4}$|^引子.*|^楔子.*|^章节目录$|^章节$|^序章.*|^上架感言.*|^完本感言.*|^番外.*|^后记.*|^尾声.*"
	cssContent         = `
.title {text-align:%s}
.content {
  margin-bottom: %s;
  margin-top: 0;
  text-indent: %dem;
  %s
}
.divider {
  border: 0;
  border-top: 1px dashed #cccccc;
  margin: 1.5em 0;
}
`
	Tutorial = `本书由kaf-cli生成: <br/>
制作教程: <a href='https://ystyle.top/2019/12/31/txt-converto-epub-and-mobi/'>https://ystyle.top/2019/12/31/txt-converto-epub-and-mobi</a>
`
)

func NewBookSimple(filename string) (*Book, error) {
	book := Book{
		Filename:   filename,
		DedupTitle: true,
	}
	book.SetDefault()
	return &book, nil
}

func NewBookArgs() *Book {
	var book Book
	flag.StringVar(&book.Filename, "filename", "", "txt 文件名")
	flag.StringVar(&book.Bookname, "bookname", "", "书名: 默认为txt文件名")
	flag.StringVar(&book.Author, "author", "YSTYLE", "作者")
	flag.StringVar(&book.Match, "match", "", "匹配标题的正则表达式, 不写可以自动识别, 如果没生成章节就参考教程。例: -match 第.{1,8}章 表示第和章字之间可以有1-8个任意文字")
	flag.StringVar(&book.VolumeMatch, "volume-match", VolumeMatch, "卷匹配规则,设置为false可以禁用卷识别")
	flag.StringVar(&book.UnknowTitle, "unknow-title", "章节正文", "未知章节默认名称")
	flag.StringVar(&book.Cover, "cover", "cover.png", "封面图片可为: 本地图片, 和orly。 设置为orly时生成orly风格的封面, 需要连接网络。")
	flag.StringVar(&book.CoverOrlyColor, "cover-orly-color", "", "orly封面的主题色, 可以为1-16和hex格式的颜色代码, 不填时随机")
	flag.IntVar(&book.CoverOrlyIdx, "cover-orly-idx", -1, "orly封面的动物, 可以为0-41, 不填时随机, 具体图案可以查看: https://orly.nanmu.me")
	flag.UintVar(&book.Max, "max", 35, "标题最大字数")
	flag.UintVar(&book.Indent, "indent", 2, "段落缩进字数")
	flag.StringVar(&book.Align, "align", GetEnv("KAF_CLI_ALIGN", "center"), "标题对齐方式: left、center、righ。环境变量KAF_CLI_ALIGN可修改默认值")
	flag.StringVar(&book.Bottom, "bottom", "1em", "段落间距(单位可以为em、px)")
	flag.StringVar(&book.LineHeight, "line-height", "", "行高(用于设置行间距, 默认为1.5rem)")
	flag.StringVar(&book.Font, "font", "", "嵌入字体, 之后epub的正文都将使用该字体")
	flag.StringVar(&book.Lang, "lang", GetEnv("KAF_CLI_LANG", "zh"), "设置语言: en,de,fr,it,es,zh,ja,pt,ru,nl。环境变量KAF_CLI_LANG可修改默认值")
	flag.StringVar(&book.Format, "format", GetEnv("KAF_CLI_FORMAT", "all"), "书籍格式: all、epub、mobi、azw3。环境变量KAF_CLI_FORMAT可修改默认值")
	flag.StringVar(&book.Out, "out", "", "输出文件名，不需要包含格式后缀")
	flag.BoolVar(&book.Tips, "tips", true, "添加本软件教程")
	flag.BoolVar(&book.DedupTitle, "dedup-title", true, "合并连续重复的章节目录行（同章号且上一节无正文）")
	flag.BoolVar(&book.NormalizeQuotes, "normalize-quotes", false, "正文对话引号优化：「」转为 “”（不影响【】）")
	flag.Parse()
	return &book
}
func (book *Book) SetDefault() {
	book.Match = defaultString(book.Match, DefaultMatchTips)
	book.VolumeMatch = defaultString(book.VolumeMatch, VolumeMatch)
	book.Author = defaultString(book.Author, "YSTYLE")
	book.UnknowTitle = defaultString(book.UnknowTitle, "章节正文")
	book.Max = defalutInt(book.Max, 35)
	book.Indent = defalutInt(book.Indent, 2)
	book.Align = defaultString(book.Align, GetEnv("KAF_CLI_ALIGN", "center"))
	book.Cover = defaultString(book.Cover, "cover.png")
	book.Bottom = defaultString(book.Bottom, "1em")
	book.Lang = defaultString(book.Lang, GetEnv("KAF_CLI_LANG", "zh"))
	book.Format = defaultString(book.Format, GetEnv("KAF_CLI_FORMAT", "all"))
}
func (book *Book) Check(version string) error {
	book.version = version
	if book.Filename == "" {
		return book.usageError()
	}
	if !strings.HasSuffix(book.Filename, ".txt") {
		return errors.New("不是txt文件")
	}
	// 通过文件名解析书名
	reg, _ := regexp.Compile(`《(.*)》.*作者[：:](.*).txt`)
	if reg.MatchString(book.Filename) {
		group := reg.FindAllStringSubmatch(book.Filename, -1)
		if len(group) == 1 && len(group[0]) >= 3 {
			if book.Bookname == "" {
				book.Bookname = group[0][1]
			}
			if book.Author == "" || book.Author == "YSTYLE" {
				book.Author = group[0][2]
			}
		}
	}
	if book.Bookname == "" {
		book.Bookname = strings.Split(filepath.Base(book.Filename), ".")[0]
	}
	if book.Out == "" {
		book.Out = book.Bookname
	}
	absFile, absErr := filepath.Abs(book.Filename)
	if absErr != nil {
		return fmt.Errorf("文件路径无效: %w", absErr)
	}
	book.Filename = absFile
	book.Out = resolveOutputPath(book.Filename, book.Out)
	book.Lang = parseLang(book.Lang)
	switch book.Cover {
	case "none":
		book.Cover = ""
	case "gen", "orly":
		cover, err := GenCover(book.Bookname, book.Author, book.CoverOrlyColor, book.CoverOrlyIdx)
		if err != nil {
			return fmt.Errorf("生成封面失败: %w", err)
		}
		book.Cover = cover
	default:
		book.Cover = resolveCoverPath(book.Filename, book.Cover)
	}

	// 编译正则表达式
	if book.Match == "" {
		book.Match = DefaultMatchTips
	}
	reg, err := regexp.Compile(book.Match)
	if err != nil {
		return fmt.Errorf("生成匹配规则出错: %s\n%s\n", book.Match, err.Error())
	}
	book.Reg = reg
	reg2, err := regexp.Compile(book.VolumeMatch)
	if err != nil {
		return fmt.Errorf("生成匹配规则出错: %s\n%s\n", book.VolumeMatch, err.Error())
	}
	book.VolumeReg = reg2
	return nil
}

func (book *Book) usageError() error {
	var buf bytes.Buffer
	origOutput := flag.CommandLine.Output()
	flag.CommandLine.SetOutput(&buf)
	defer flag.CommandLine.SetOutput(origOutput)

	fmt.Fprintln(&buf, "错误: 文件名不能为空")
	fmt.Fprintln(&buf, "软件版本:\t", book.version)
	fmt.Fprintln(&buf, "简洁模式:\t把文件拖放到kaf-cli上")
	fmt.Fprintln(&buf, "命令行简单模式: kaf-cli ebook.txt")
	fmt.Fprintln(&buf, "\n以下为kaf-cli的全部参数")
	flag.PrintDefaults()
	return errors.New(strings.TrimRight(buf.String(), "\n"))
}

func (book *Book) openTextReader(filename string) (io.ReadCloser, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("读取文件出错: %w", err)
	}
	temBuf := bufio.NewReader(f)
	bs, _ := temBuf.Peek(1024)
	encodig, encodename, _ := charset.DetermineEncoding(bs, "text/plain")
	if _, err := f.Seek(0, 0); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("读取文件出错: %w", err)
	}
	if encodename == "utf-8" {
		return f, nil
	}

	bs, err = io.ReadAll(f)
	closeErr := f.Close()
	if err != nil {
		return nil, fmt.Errorf("读取文件出错: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("关闭文件出错: %w", closeErr)
	}

	book.Decoder = encodig.NewDecoder()
	if encodename == "windows-1252" {
		book.Decoder = simplifiedchinese.GB18030.NewDecoder()
	}
	bs, _, err = transform.Bytes(book.Decoder, bs)
	if err != nil {
		return nil, fmt.Errorf("文本转码失败: %w", err)
	}
	return io.NopCloser(bytes.NewReader(bs)), nil
}

func (book *Book) ToString() {
	fmt.Println("转换信息:")
	fmt.Println("软件版本:", book.version)
	fmt.Println("文件名:\t", book.Filename)
	fmt.Println("书籍书名:", book.Bookname)
	fmt.Println("书籍作者:", book.Author)
	if book.Cover != "" {
		fmt.Println("书籍封面:", book.Cover)
	}
	fmt.Println("书籍语言:", book.Lang)
	if book.Match == DefaultMatchTips {
		fmt.Println("匹配条件:", "自动匹配")
	} else {
		fmt.Println("匹配条件:", book.Match)
	}
	fmt.Println("卷匹配条件:", book.VolumeMatch)
	fmt.Println("转换格式:", book.Format)
	fmt.Println()
}

func (book *Book) Parse() error {
	var contentList []Section
	fmt.Println("正在读取txt文件...")
	start := time.Now()
	reader, err := book.openTextReader(book.Filename)
	if err != nil {
		return err
	}
	defer reader.Close()
	buf := bufio.NewReader(reader)

	// 自动嗅探对话引号排版习惯（直角引号 「」『』）
	sampleBytes, _ := buf.Peek(65536)
	if len(sampleBytes) == 0 {
		sampleBytes, _ = buf.Peek(4096)
	}
	if len(sampleBytes) > 0 {
		sampleStr := string(sampleBytes)
		cornerCount := strings.Count(sampleStr, "「") + strings.Count(sampleStr, "」") +
			strings.Count(sampleStr, "『") + strings.Count(sampleStr, "』")
		if cornerCount >= 2 {
			if !book.NormalizeQuotes {
				book.NormalizeQuotes = true
				fmt.Println("AI/智能排版: 检测到正文包含「」『』直角引号，已自动启用对话引号规范化")
			}
		}
	}

	var title string
	var content bytes.Buffer
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if line != "" {
					if line = strings.TrimSpace(line); line != "" {
						line = strings.ReplaceAll(line, "<", "&lt;")
						line = strings.ReplaceAll(line, ">", "&gt;")
						if book.NormalizeQuotes {
							line = normalizeLineQuotes(line)
						}
						addPart(&content, line)
					}
				}
				contentList = append(contentList, Section{
					Title:   title,
					Content: content.String(),
				})
				content.Reset()
				break
			}
			return fmt.Errorf("读取文件出错: %w", err)
		}
		line = strings.TrimSpace(line)
		line = strings.ReplaceAll(line, "<", "&lt;")
		line = strings.ReplaceAll(line, ">", "&gt;")
		// 空行直接跳过
		if len(line) == 0 {
			continue
		}
		// 「第一部」「第三部」等正文分篇标注，不作为章节/卷
		if isPartDivisionLabel(line) {
			if book.NormalizeQuotes {
				line = normalizeLineQuotes(line)
			}
			addPart(&content, line)
			continue
		}
		// 「第二回合」「双方行棋，第三回合」等过程标签，不作为章节
		if isRoundLabel(line) {
			if book.NormalizeQuotes {
				line = normalizeLineQuotes(line)
			}
			addPart(&content, line)
			continue
		}
		// 处理标题
		cleanedTitle := cleanChapterTitle(line)
		if (utf8.RuneCountInString(line) <= int(book.Max) || utf8.RuneCountInString(cleanedTitle) <= int(book.Max)) &&
			(book.Reg.MatchString(line) || book.VolumeReg.MatchString(line) || book.Reg.MatchString(cleanedTitle)) &&
			!isFalsePositiveOrdinalTitle(line) &&
			!looksLikeInlineChapterLine(line) {
			if title == "" {
				title = book.UnknowTitle
			}
			if content.Len() > 0 || title != book.UnknowTitle {
				if title != book.UnknowTitle || !isTrivialPreface(content.String()) {
					contentList = append(contentList, Section{
						Title:   title,
						Content: content.String(),
					})
				}
			}
			title = cleanedTitle
			content.Reset()
			continue
		}
		if book.NormalizeQuotes {
			line = normalizeLineQuotes(line)
		}
		addPart(&content, line)
	}
	// 没识别到章节又没识别到 EOF 时，把所有的内容写到最后一章
	if content.Len() != 0 {
		if title == "" {
			title = "章节正文"
		}
		contentList = append(contentList, Section{
			Title:   title,
			Content: content.String(),
		})
	}
	// 智能目录去重自适应识别：若用户未开启但处于 AI 模式或检测到重复疑点，自动开启
	if !book.DedupTitle && (book.AIOptions.Enabled || hasDuplicateOrIsolatedSections(contentList)) {
		book.DedupTitle = true
		fmt.Println("AI/智能排版: 检测到章节目录存在重复或误切分疑点，已自动启用智能目录去重")
	}
	if book.DedupTitle {
		beforeLen := len(contentList)
		contentList = dedupTitleSections(contentList)
		contentList = dedupRepeatedSections(contentList)
		contentList = mergeIsolatedDigitSections(contentList)
		if diff := beforeLen - len(contentList); diff > 0 {
			fmt.Printf("智能目录去重: 已自动合并/清理 %d 处重复或误切分章节\n", diff)
		}
	}
	var sectionList []Section
	var volumeSection *Section
	for _, section := range contentList {
		if book.VolumeMatch != "false" && book.VolumeReg.MatchString(section.Title) {
			if volumeSection != nil {
				sectionList = append(sectionList, *volumeSection)
				volumeSection = nil
			}
			temp := section
			volumeSection = &temp
		} else {
			if volumeSection == nil {
				sectionList = append(sectionList, section)
			} else {
				volumeSection.Sections = append(volumeSection.Sections, section)
			}
		}
	}
	// 如果有最后一卷,添加到章节列表
	if volumeSection != nil {
		sectionList = append(sectionList, *volumeSection)
		volumeSection = nil
	}
	end := time.Now().Sub(start)
	fmt.Println("读取文件耗时:", end)
	fmt.Println("匹配章节:", sectionCount(sectionList))
	// 添加提示
	if book.Tips {
		tuorialSection := Section{
			Title:   "制作说明",
			Content: Tutorial,
		}
		sectionList = append([]Section{tuorialSection}, sectionList...)
		sectionList = append(sectionList, tuorialSection)
	}
	book.SectionList = sectionList
	return nil
}

func sectionCount(sections []Section) int {
	var count int
	for _, section := range sections {
		count += 1 + len(section.Sections)
	}
	return count
}

func (book *Book) Convert() error {
	start := time.Now()
	// 解析文本
	fmt.Println()

	// 判断要生成的格式
	var isEpub, isMobi, isAzw3 bool
	switch book.Format {
	case "epub":
		isEpub = true
	case "mobi":
		isEpub = true
		isMobi = true
	case "azw3":
		isAzw3 = true
	default:
		isEpub = true
		isMobi = true
		isAzw3 = true
	}

	hasKinldegen := lookKindlegen()
	if book.Format == "mobi" && hasKinldegen == "" {
		isEpub = false
	}

	var convert Converter
	// 生成epub
	if isEpub {
		convert = EpubConverter{}
		if err := convert.Build(*book); err != nil {
			return fmt.Errorf("生成epub失败: %w", err)
		}
		fmt.Println()
	}
	// 生成azw3格式
	if isAzw3 {
		convert = Azw3Converter{}
		// 生成kindle格式
		if err := convert.Build(*book); err != nil {
			return fmt.Errorf("生成azw3失败: %w", err)
		}
	}
	// 生成mobi格式
	if isMobi {
		if hasKinldegen == "" {
			convert = MobiConverter{}
			if err := convert.Build(*book); err != nil {
				return fmt.Errorf("生成mobi失败: %w", err)
			}
		} else {
			if err := converToMobi(fmt.Sprintf("%s.epub", book.Out), book.Lang); err != nil {
				return err
			}
		}
	}
	end := time.Now().Sub(start)
	fmt.Println("\n转换完成! 总耗时:", end)
	return nil
}

func addPart(buff *bytes.Buffer, content string) {
	if isDividerLine(content) {
		buff.WriteString(`<hr class="divider"/>`)
		return
	}
	if strings.HasSuffix(content, "==") ||
		strings.HasSuffix(content, "**") ||
		strings.HasSuffix(content, "--") ||
		strings.HasSuffix(content, "//") {
		buff.WriteString(content)
		return
	}
	buff.WriteString(htmlPStart)
	buff.WriteString(content)
	buff.WriteString(htmlPEnd)
}

func isTrivialPreface(content string) bool {
	var text strings.Builder
	inTag := false
	for _, r := range content {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag && !unicode.IsSpace(r) {
			text.WriteRune(r)
		}
	}
	return text.Len() == 0
}

func hasDuplicateOrIsolatedSections(sections []Section) bool {
	if len(sections) <= 1 {
		return false
	}
	for i := 0; i < len(sections)-1; i++ {
		if sections[i].Content == "" {
			k1 := chapterKey(sections[i].Title)
			k2 := chapterKey(sections[i+1].Title)
			if k1 != "" && k1 == k2 {
				return true
			}
		}
	}
	stdCount := 0
	digitCount := 0
	for _, sec := range sections {
		if chapterKeyReg.MatchString(sec.Title) || chapterDualKeyReg.MatchString(sec.Title) ||
			strings.HasPrefix(strings.ToLower(sec.Title), "chapter") {
			stdCount++
		} else if pureDigitTitleRegex.MatchString(strings.TrimSpace(sec.Title)) {
			digitCount++
		}
	}
	if stdCount >= 3 && digitCount > 0 && float64(stdCount)/float64(len(sections)) >= 0.6 {
		return true
	}
	return false
}
