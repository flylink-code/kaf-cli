package kafcli

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var (
	chapterDualKeyReg   = regexp.MustCompile(`^第[0-9一二三四五六七八九十零〇百千两]+[章回节集卷]\s*([0-9一二三四五六七八九十零〇百千两]+)[章回节集卷]`)
	chapterKeyReg       = regexp.MustCompile(`^第([0-9一二三四五六七八九十零〇百千两]+)[章回节集卷]`)
	chapterPlainKeyReg  = regexp.MustCompile(`^([0-9一二三四五六七八九十零〇百千两]+)[章回节集卷]`)
	filenameMetaReg     = regexp.MustCompile(`《(.*)》.*作者[：:](.*)\.txt`)
	partDivisionLabelRe = regexp.MustCompile(`^第[0-9一二三四五六七八九十零〇百千两]+部$`)
	roundLabelRe        = regexp.MustCompile(`^(第?[0-9一二三四五六七八九十零〇百千两]+回合|【?第?[0-9一二三四五六七八九十零〇百千两]+回合结束|【?双方行棋，第?[0-9一二三四五六七八九十零〇百千两]+回合)`)

	falsePositiveOrdinalRegs = []*regexp.Regexp{
		regexp.MustCompile(`^第[0-9一二三四五六七八九十零〇百千两]+节课`),
		regexp.MustCompile(`^第[0-9一二三四五六七八九十零〇百千两]+章·`),
		regexp.MustCompile(`^第[0-9一二三四五六七八九十零〇百千两]+章[^「“『"‘【\[\(（《<〔〈\s　\t：:、，,\-—–~～\.．！!？?]`),
	}

	titleTrailingParenRegex     = regexp.MustCompile(`\s*[(（][^()（）]*(?:字|更|月票|求票|求追读|求收藏|求推荐|求订阅|求支持|打赏|加更|补更|二合一|[pP][kK])[^()（）]*[)）]?\s*$`)
	titleTrailingWordCountRegex = regexp.MustCompile(`\s*[(（]\s*[0-9一二两三四五六七八九十百千]{2,6}\s*(?:字)?\s*[)）]?\s*$`)
)

// cleanChapterTitle 清理章节标题尾部的作话、字数统计、求月票等杂质。
func cleanChapterTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return title
	}
	for {
		prev := title
		title = titleTrailingParenRegex.ReplaceAllString(title, "")
		title = titleTrailingWordCountRegex.ReplaceAllString(title, "")
		title = strings.TrimSpace(title)
		if title == prev || title == "" {
			break
		}
	}
	if title == "" {
		return strings.TrimSpace(title)
	}
	return title
}

// isPartDivisionLabel 判断是否为「第一部」「第三部」等正文分篇行（非目录章节）。
func isPartDivisionLabel(line string) bool {
	return partDivisionLabelRe.MatchString(line)
}

func isRoundLabel(line string) bool {
	return roundLabelRe.MatchString(strings.TrimSpace(line))
}

// isDividerLine 判断是否为段落分割线，例如 "------------"、"------"、"***" 等。
func isDividerLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}
	r := rune(trimmed[0])
	if r != '-' && r != '—' && r != '*' && r != '=' && r != '_' {
		return false
	}
	for _, ch := range trimmed {
		if ch != r {
			return false
		}
	}
	return true
}

func isFalsePositiveOrdinalTitle(line string) bool {
	line = normalizeLineQuotes(strings.TrimSpace(line))
	if line == "" {
		return false
	}
	for _, re := range falsePositiveOrdinalRegs {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

// resolveOutputPath 解析输出路径为绝对路径；仅文件名时写入 txt 所在目录。
func resolveOutputPath(txtPath, out string) string {
	out = strings.TrimSpace(out)
	if out == "" {
		return out
	}
	if filepath.IsAbs(out) {
		return filepath.Clean(out)
	}
	if dir := filepath.Dir(out); dir != "." {
		if abs, err := filepath.Abs(out); err == nil {
			return abs
		}
	}
	return filepath.Join(filepath.Dir(txtPath), filepath.Base(out))
}

// resolveCoverPath 尝试解析封面路径；默认封面支持 cover.png / cover.jpg / cover.jpeg。
func resolveCoverPath(txtPath, cover string) string {
	cover = strings.TrimSpace(cover)
	if cover == "" {
		return ""
	}
	candidates := []string{cover}
	if cover == "cover.png" {
		candidates = []string{"cover.png", "cover.jpg", "cover.jpeg"}
	}

	baseDir := filepath.Dir(txtPath)
	for _, candidate := range candidates {
		path := candidate
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		if exists, _ := isExists(path); exists {
			return path
		}
	}
	return ""
}

// FilenameMeta 从 txt 路径推测书名与作者（规则与 Check 一致）。
func FilenameMeta(filename string) (bookname, author string) {
	if filenameMetaReg.MatchString(filename) {
		group := filenameMetaReg.FindStringSubmatch(filename)
		if len(group) >= 3 {
			return strings.TrimSpace(group[1]), strings.TrimSpace(group[2])
		}
	}
	base := filepath.Base(filename)
	return strings.TrimSuffix(base, filepath.Ext(base)), ""
}

func parseLang(lang string) string {
	if lang == "" {
		return "en"
	}
	for _, supported := range []string{"en", "de", "fr", "it", "es", "zh", "ja", "pt", "ru", "nl"} {
		if lang == supported {
			return lang
		}
	}
	return "en"
}

func run(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func lookKindlegen() string {
	command := "kindlegen"
	if runtime.GOOS == "windows" {
		command = "kindlegen.exe"
	}
	kindlegen, err := exec.LookPath(command)
	if err != nil {
		currentDir, err := os.Executable()
		if err != nil {
			return ""
		}
		kindlegen = filepath.Join(filepath.Dir(currentDir), command)
		if exist, _ := isExists(kindlegen); !exist {
			return ""
		}
		fmt.Println("kindlegen: ", kindlegen)
	}
	return kindlegen
}

func converToMobi(bookname, lang string) error {
	command := lookKindlegen()
	if command == "" {
		return fmt.Errorf("未找到Kindle格式转换器 kindlegen")
	}
	fmt.Printf("\n检测到Kindle格式转换器: %s，正在把书籍转换成Kindle格式...\n", command)
	fmt.Println("转换mobi比较花时间, 大约耗时1-10分钟, 请等待...")
	start := time.Now()
	if err := run(command, "-dont_append_source", "-locale", lang, "-c1", bookname); err != nil {
		return fmt.Errorf("转换mobi失败: %w", err)
	}
	// 计算耗时
	end := time.Now().Sub(start)
	fmt.Println("转换为mobi格式耗时:", end)
	return nil
}

func isExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func getClientID() string {
	clientID := fmt.Sprintf("%d", rand.Uint32())
	config, err := os.UserConfigDir()
	if err != nil {
		return clientID
	}
	filepath := fmt.Sprintf("%s/kaf-wifi/config", config)
	if exist, _ := isExists(filepath); exist {
		bs, err := ioutil.ReadFile(filepath)
		if err != nil {
			return clientID
		}
		clientID = string(bs)
	} else {
		err := os.MkdirAll(fmt.Sprintf("%s/kaf-wifi", config), 0700)
		if err != nil {
			return clientID
		}
		_ = os.WriteFile(filepath, []byte(clientID), 0700)
	}
	return clientID
}

func GetEnv(key, defaultvalue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultvalue
}

var colors = []string{
	"#61005e",
	"#70706d",
	"#890029",
	"#c4000e",
	"#6d001d",
	"#6a00bd",
	"#f10000",
	"#0071b1",
	"#f9bc00",
	"#2c0077",
	"#ba009a",
	"#009047",
	"#009d9e",
	"#222e85",
	"#bd002e",
	"#009d1a",
	"#75a500",
}

func ParseInt(v string) int {
	v = strings.ReplaceAll(v, ",", "")
	i, err := strconv.ParseInt(v, 0, 32)
	if err != nil {
		return 0
	}
	return int(i)
}

func GenCover(title, author, color string, img int) (string, error) {
	query := url.Values{}
	query.Add("title", title)
	query.Add("author", author)
	query.Add("g_loc", "BR")
	query.Add("top_text", "kaf")
	query.Add("g_text", "")
	if img >= 0 && img <= 41 {
		query.Add("img_id", fmt.Sprintf("%d", img))
	} else {
		query.Add("img_id", fmt.Sprintf("%d", rand.Intn(41)))
	}
	if strings.HasPrefix(color, "#") {
		query.Add("color", strings.TrimLeft(color, "#"))
	} else {
		i := ParseInt(color)
		if i == 0 {
			i = rand.Intn(17)
		}
		color := colors[i]
		query.Add("color", strings.TrimLeft(color, "#"))
	}

	uri := fmt.Sprintf("https://orly.nanmu.me/api/generate?%s", query.Encode())
	res, err := http.Get(uri)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("封面服务返回异常状态: %s", res.Status)
	}
	tempDir, err := os.MkdirTemp("", "kaf-cli")
	if err != nil {
		return "", err
	}
	bs, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	coverfile := filepath.Join(tempDir, fmt.Sprintf("%s.jpg", title))
	err = os.WriteFile(coverfile, bs, 0666)
	if err != nil {
		return "", err
	}
	return coverfile, nil
}

type Number interface {
	~int | ~uint
}

func defaultString(src, dst string) string {
	if src == "" {
		return dst
	}
	return src
}
func defalutInt[T Number](src, dst T) T {
	if src == 0 {
		return dst
	}
	return src
}
func defaultBool(src, dst bool) bool {
	if src {
		return src
	}
	return dst
}

func normalizeChapterOrdinal(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if n, err := strconv.Atoi(raw); err == nil {
		return strconv.Itoa(n)
	}

	digits := map[rune]int{
		'零': 0,
		'〇': 0,
		'一': 1,
		'二': 2,
		'两': 2,
		'三': 3,
		'四': 4,
		'五': 5,
		'六': 6,
		'七': 7,
		'八': 8,
		'九': 9,
	}
	units := map[rune]int{
		'十': 10,
		'百': 100,
		'千': 1000,
	}

	total := 0
	current := 0
	seen := false
	for _, r := range raw {
		if v, ok := digits[r]; ok {
			current = v
			seen = true
			continue
		}
		if unit, ok := units[r]; ok {
			if current == 0 {
				current = 1
			}
			total += current * unit
			current = 0
			seen = true
			continue
		}
		return ""
	}
	if !seen {
		return ""
	}
	return strconv.Itoa(total + current)
}

func chapterKey(title string) string {
	title = strings.TrimSpace(title)
	for _, re := range []*regexp.Regexp{chapterDualKeyReg, chapterKeyReg, chapterPlainKeyReg} {
		m := re.FindStringSubmatch(title)
		if len(m) >= 2 {
			return normalizeChapterOrdinal(m[1])
		}
	}
	return ""
}

// normalizeLineQuotes 将正文直角引号替换为弯引号（仅处理「」『』，不改动【】）。
func normalizeLineQuotes(line string) string {
	if !strings.ContainsAny(line, "「」『』") {
		return line
	}
	return strings.NewReplacer(
		"「", "\u201c",
		"」", "\u201d",
		"『", "\u2018",
		"』", "\u2019",
	).Replace(line)
}

// dedupTitleSections 去掉「目录行 + 正文标题行」重复：上一节无正文且与下一节章号相同则跳过。
func dedupTitleSections(sections []Section) []Section {
	if len(sections) <= 1 {
		return sections
	}
	result := make([]Section, 0, len(sections))
	for i := 0; i < len(sections); i++ {
		sec := sections[i]
		if i+1 < len(sections) && sec.Content == "" {
			nextKey := chapterKey(sections[i+1].Title)
			if key := chapterKey(sec.Title); key != "" && key == nextKey {
				continue
			}
		}
		result = append(result, sec)
	}
	return result
}

var pureDigitTitleRegex = regexp.MustCompile(`^\d{1,5}$`)

// mergeIsolatedDigitSections 合并中文网文中被正则误切为章节的孤立纯数字行。
// 当全书中绝大多数章节（>=60% 且至少 3 章）为规范的「第X章/卷」或「Chapter」时，
// 零散出现在正文中的纯数字（如打赏积分、页码、点赞数）判定为正文误切分，将其拼回前一章正文。
func mergeIsolatedDigitSections(sections []Section) []Section {
	if len(sections) <= 2 {
		return sections
	}
	standardChapterCount := 0
	for _, sec := range sections {
		if chapterKeyReg.MatchString(sec.Title) || chapterDualKeyReg.MatchString(sec.Title) ||
			strings.HasPrefix(strings.ToLower(sec.Title), "chapter") {
			standardChapterCount++
		}
	}
	if standardChapterCount < 3 || float64(standardChapterCount)/float64(len(sections)) < 0.6 {
		return sections
	}

	var merged []Section
	for _, sec := range sections {
		if pureDigitTitleRegex.MatchString(strings.TrimSpace(sec.Title)) && len(merged) > 0 {
			lastIdx := len(merged) - 1
			var sb strings.Builder
			sb.WriteString(merged[lastIdx].Content)
			if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "\n") {
				sb.WriteString("\n")
			}
			sb.WriteString(fmt.Sprintf(`<p class="content">%s</p>`, strings.TrimSpace(sec.Title)))
			sb.WriteString("\n")
			if sec.Content != "" {
				sb.WriteString(sec.Content)
			}
			merged[lastIdx].Content = sb.String()
			continue
		}
		merged = append(merged, sec)
	}
	return merged
}

func normalizeChapterContent(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	content = normalizeLineQuotes(content)
	replacer := strings.NewReplacer(
		htmlPStart, "\n",
		htmlPEnd, "",
		"\r", "",
		"\u3000", "",
		" ", "",
		"\t", "",
	)
	lines := strings.Split(replacer.Replace(content), "\n")
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		normalized = append(normalized, line)
	}
	return strings.Join(normalized, "\n")
}

func compactDedupText(text string) string {
	text = normalizeLineQuotes(strings.TrimSpace(strings.ToLower(text)))
	if text == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"\r", "",
		"\n", "",
		"\t", "",
		" ", "",
		"\u3000", "",
		"“", "",
		"”", "",
		"‘", "",
		"’", "",
		"「", "",
		"」", "",
		"『", "",
		"』", "",
		"【", "",
		"】", "",
		"（", "",
		"）", "",
		"(", "",
		")", "",
		"《", "",
		"》", "",
		"·", "",
		"—", "",
		"-", "",
		"…", "",
		"，", "",
		"。", "",
		"：", "",
		":", "",
		"；", "",
		";", "",
		"、", "",
		"！", "",
		"!", "",
		"？", "",
		"?", "",
		".", "",
		",", "",
	)
	return replacer.Replace(text)
}

func comparableDedupRune(r rune) (rune, bool) {
	switch r {
	case '\r', '\n', '\t', ' ', '\u3000',
		'“', '”', '‘', '’', '「', '」', '『', '』',
		'【', '】', '（', '）', '(', ')', '《', '》',
		'·', '—', '-', '…',
		'，', '。', '：', ':', '；', ';', '、',
		'！', '!', '？', '?', '.', ',':
		return 0, false
	}
	return unicode.ToLower(r), true
}

func trimLeadingTitlePrefix(title, line string) (string, bool) {
	title = normalizeLineQuotes(strings.TrimSpace(title))
	line = normalizeLineQuotes(strings.TrimSpace(line))
	if title == "" || line == "" {
		return line, false
	}
	if key := chapterKey(title); key == "" || chapterKey(line) != key {
		return line, false
	}

	titleRunes := []rune(title)
	lineRunes := []rune(line)
	titleIdx := 0
	end := 0

	for i, r := range lineRunes {
		lineCmp, ok := comparableDedupRune(r)
		if !ok {
			continue
		}
		for titleIdx < len(titleRunes) {
			titleCmp, titleOk := comparableDedupRune(titleRunes[titleIdx])
			titleIdx++
			if !titleOk {
				continue
			}
			if titleCmp != lineCmp {
				return line, false
			}
			end = i + 1
			break
		}
		if titleIdx >= len(titleRunes) {
			break
		}
	}

	for titleIdx < len(titleRunes) {
		if _, ok := comparableDedupRune(titleRunes[titleIdx]); ok {
			return line, false
		}
		titleIdx++
	}
	if end == 0 {
		return line, false
	}

	rest := strings.TrimSpace(string(lineRunes[end:]))
	rest = strings.TrimLeft(rest, `·"“”'‘’「」『』】）》)）]`)
	return strings.TrimSpace(rest), true
}

func dedupContentVariants(text string) []string {
	text = normalizeChapterContent(text)
	if text == "" {
		return nil
	}
	var variants []string
	seen := make(map[string]struct{})
	add := func(s string) {
		s = compactDedupText(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		variants = append(variants, s)
	}
	add(text)
	if trimmed := trimLeadingShortSentence(text); trimmed != "" && trimmed != text {
		add(trimmed)
	}
	return variants
}

func trimLeadingShortSentence(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return text
	}
	first := strings.TrimSpace(lines[0])
	if first == "" {
		return text
	}
	runes := []rune(first)
	limit := len(runes)
	if limit > 24 {
		limit = 24
	}
	for i := 0; i < limit; i++ {
		switch runes[i] {
		case '。', '！', '？', '!', '?':
			rest := strings.TrimSpace(string(runes[i+1:]))
			if rest == "" {
				return strings.Join(lines[1:], "\n")
			}
			lines[0] = rest
			return strings.Join(lines, "\n")
		}
	}
	return text
}

func commonPrefixRuneCount(a, b string) int {
	ar := []rune(a)
	br := []rune(b)
	limit := len(ar)
	if len(br) < limit {
		limit = len(br)
	}
	count := 0
	for count < limit && ar[count] == br[count] {
		count++
	}
	return count
}

func titlesLikelyRepeated(a, b string) bool {
	ta := compactDedupText(a)
	tb := compactDedupText(b)
	if ta == "" || tb == "" {
		return false
	}
	if ta == tb {
		return true
	}
	shorter, longer := ta, tb
	if len([]rune(shorter)) > len([]rune(longer)) {
		shorter, longer = longer, shorter
	}
	if len([]rune(shorter)) >= 6 && strings.Contains(longer, shorter) {
		return true
	}
	return commonPrefixRuneCount(ta, tb) >= 8
}

func contentsLikelyRepeated(a, b string) bool {
	variantsA := dedupContentVariants(a)
	variantsB := dedupContentVariants(b)
	for _, ca := range variantsA {
		for _, cb := range variantsB {
			if ca == cb {
				return true
			}
			prefix := commonPrefixRuneCount(ca, cb)
			shorter := len([]rune(ca))
			if len([]rune(cb)) < shorter {
				shorter = len([]rune(cb))
			}
			if shorter == 0 {
				continue
			}
			if shorter <= 80 {
				if prefix >= 20 && float64(prefix)/float64(shorter) >= 0.85 {
					return true
				}
				continue
			}
			if prefix >= 80 {
				return true
			}
		}
	}
	return false
}

func looksLikeInlineChapterLine(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	key := chapterKey(line)
	if key == "" {
		return false
	}
	cleaned := cleanChapterTitle(line)

	// 1. 如果标题包含成对引号，检查闭合引号后面是否粘连了正文叙述
	for _, closing := range []string{"”", "」", "』", "\""} {
		if idx := strings.LastIndex(cleaned, closing); idx >= 0 && idx+len(closing) < len(cleaned) {
			rest := strings.TrimSpace(cleaned[idx+len(closing):])
			rest = strings.Trim(rest, "· \t\r\n")
			if rest != "" {
				return true
			}
		}
	}

	// 2. 如果包含陈述句句号「。」，且句号后还有正文文字，说明是标题后粘连了正文句子
	if idx := strings.Index(cleaned, "。"); idx >= 0 && idx+len("。") < len(cleaned) {
		rest := strings.TrimSpace(cleaned[idx+len("。"):])
		rest = strings.Trim(rest, `"'”’」』】）》)）]》· \t\r\n`)
		if rest != "" {
			return true
		}
	}

	return false
}

func stripLeadingTitleLine(title, content string) string {
	title = strings.TrimSpace(title)
	content = normalizeChapterContent(content)
	if title == "" || content == "" {
		return content
	}
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return content
	}
	first := strings.TrimSpace(lines[0])
	if trimmed, ok := trimLeadingTitlePrefix(title, first); ok {
		if trimmed == "" {
			return strings.Join(lines[1:], "\n")
		}
		lines[0] = trimmed
		return strings.Join(lines, "\n")
	}
	if first == title || strings.HasPrefix(first, title) {
		trimmed := strings.TrimSpace(strings.TrimPrefix(first, title))
		if trimmed == "" {
			return strings.Join(lines[1:], "\n")
		}
		lines[0] = trimmed
		return strings.Join(lines, "\n")
	}
	if alt := strings.Replace(title, "「", "“", 1); alt != title {
		alt = strings.Replace(alt, "」", "”", 1)
		if first == alt || strings.HasPrefix(first, alt) {
			trimmed := strings.TrimSpace(strings.TrimPrefix(first, alt))
			if trimmed == "" {
				return strings.Join(lines[1:], "\n")
			}
			lines[0] = trimmed
			return strings.Join(lines, "\n")
		}
	}
	if key := chapterKey(title); key != "" {
		if firstKey := chapterKey(first); firstKey == key {
			for _, mark := range []string{"？", "?", "。", "！", "!"} {
				if idx := strings.Index(first, mark); idx >= 0 && idx+len(mark) < len(first) {
					rest := strings.TrimSpace(first[idx+len(mark):])
					rest = strings.TrimLeft(rest, `”"」』·`)
					rest = strings.TrimSpace(rest)
					if rest == "" {
						return strings.Join(lines[1:], "\n")
					}
					lines[0] = rest
					return strings.Join(lines, "\n")
				}
			}
		}
	}
	return content
}

// dedupRepeatedSections 去掉后续整章重复内容：同章号且正文归一化后相同则保留第一次出现。
func dedupRepeatedSections(sections []Section) []Section {
	if len(sections) <= 1 {
		return sections
	}
	seen := make(map[string][]Section)
	result := make([]Section, 0, len(sections))
	for _, sec := range sections {
		key := chapterKey(sec.Title)
		if key == "" {
			result = append(result, sec)
			continue
		}
		contentKey := normalizeChapterContent(sec.Content)
		contentKey = stripLeadingTitleLine(sec.Title, contentKey)
		if contentKey == "" {
			result = append(result, sec)
			continue
		}
		repeated := false
		for _, prev := range seen[key] {
			prevContentKey := normalizeChapterContent(prev.Content)
			prevContentKey = stripLeadingTitleLine(prev.Title, prevContentKey)
			if titlesLikelyRepeated(prev.Title, sec.Title) && contentsLikelyRepeated(prevContentKey, contentKey) {
				repeated = true
				break
			}
		}
		if repeated {
			continue
		}
		seen[key] = append(seen[key], sec)
		result = append(result, sec)
	}
	return result
}
