package ai

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// structureEntry 是扁平章节在结构分析中的最小上下文（不上传正文）。
type structureEntry struct {
	Title string
	Empty bool // 无正文（仅目录行）
}

const (
	structureSuspectPadding  = 1  // 疑点区间向两侧扩展的标题数，给 AI 一点上下文
	structureMaxSuspectBatch = 30 // 单次 API 请求最多送入的标题数
	structureMaxAICalls      = 12 // 单本书结构分析 API 调用上限
)

var (
	chapterNumRe = regexp.MustCompile(`第([0-9一二三四五六七八九十零〇百千两]+)[章回节集卷]`)
	urlInTitleRe = regexp.MustCompile(`(?i)(https?://|www\.)`)
)

// titleWatermarkKeywords 采集站/广告常见水印词。
var titleWatermarkKeywords = []string{
	"请收藏", "收藏本站", "最新章节", "更多精彩", "百度搜索",
	"手机用户", "访问地址", "侵权内容", "防盗章节", "防采集",
	"备用地址", "请记住", "更新最快", "无弹窗", "笔趣阁",
	"手机阅读", "点击进入", "看最新", "下载app", "关注公众号",
	"举报后", "顶点", "小说网", "阅读网址",
}

// suspectRange 是一组需 AI 复核的连续章节（全局扁平序号，含起止）。
type suspectRange struct {
	Start   int
	End     int
	Reasons []string
}

// flattenStructureEntries 与 SectionList.Flatten 对齐，额外标记是否无正文。
func flattenStructureEntries(list SectionList) []structureEntry {
	var out []structureEntry
	for _, sec := range list {
		if len(sec.Sections) == 0 {
			out = append(out, structureEntry{
				Title: sec.Title,
				Empty: isEmptyContent(sec.Content),
			})
			continue
		}
		for _, sub := range sec.Sections {
			out = append(out, structureEntry{
				Title: sub.Title,
				Empty: isEmptyContent(sub.Content),
			})
		}
	}
	return out
}

func isEmptyContent(html string) bool {
	return strings.TrimSpace(plainText(html)) == ""
}

// detectStructureSuspects 本地扫描标题疑点，返回合并后的复核区间。
func detectStructureSuspects(entries []structureEntry) []suspectRange {
	n := len(entries)
	if n == 0 {
		return nil
	}
	flags := make(map[int][]string)

	lastKeyAt := map[string]int{}
	prevNum := -1
	for i, e := range entries {
		title := strings.TrimSpace(e.Title)
		if title == "" {
			flags[i] = append(flags[i], "空标题")
			continue
		}
		if reason := titleWatermarkReason(title); reason != "" {
			flags[i] = append(flags[i], reason)
		}
		if urlInTitleRe.MatchString(title) {
			flags[i] = append(flags[i], "标题含网址")
		}
		if rl := len([]rune(title)); rl > 60 {
			flags[i] = append(flags[i], "标题过长")
		} else if rl <= 1 {
			flags[i] = append(flags[i], "标题过短")
		}
		if looksLikeGarbageTitle(title) {
			flags[i] = append(flags[i], "标题乱码/符号异常")
		}
		if e.Empty && chapterNumRe.MatchString(title) {
			flags[i] = append(flags[i], "无正文目录行")
		}
		if key := chapterKeySimple(title); key != "" {
			if prev, ok := lastKeyAt[key]; ok && i-prev <= 4 {
				flags[i] = append(flags[i], "章号重复")
				flags[prev] = appendUnique(flags[prev], "章号重复")
			}
			lastKeyAt[key] = i
			if num, ok := parseArabicChapterNum(key); ok {
				if prevNum >= 0 && num > prevNum+1 && num-prevNum <= 20 {
					flags[i] = append(flags[i], "章号跳号")
				}
				prevNum = num
			}
		}
	}
	return mergeSuspectFlags(flags, n, structureSuspectPadding, structureMaxSuspectBatch)
}

func titleWatermarkReason(title string) string {
	lower := strings.ToLower(title)
	for _, kw := range titleWatermarkKeywords {
		if strings.Contains(title, kw) || strings.Contains(lower, strings.ToLower(kw)) {
			return "标题含水印/广告词"
		}
	}
	return ""
}

func chapterKeySimple(title string) string {
	m := chapterNumRe.FindStringSubmatch(title)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

func parseArabicChapterNum(key string) (int, bool) {
	n := 0
	for _, r := range key {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, n > 0
}

func looksLikeGarbageTitle(title string) bool {
	if title == "" {
		return false
	}
	var letters, digits, other int
	for _, r := range title {
		switch {
		case unicode.IsLetter(r):
			letters++
		case unicode.IsDigit(r):
			digits++
		default:
			other++
		}
	}
	total := letters + digits + other
	if total == 0 {
		return true
	}
	// 符号占比过高且几乎没有汉字/字母
	if letters == 0 && other*3 > total*2 {
		return true
	}
	return false
}

func mergeSuspectFlags(flags map[int][]string, total, padding, maxBatch int) []suspectRange {
	if len(flags) == 0 {
		return nil
	}
	indices := make([]int, 0, len(flags))
	for i := range flags {
		indices = append(indices, i)
	}
	sort.Ints(indices)

	var merged []suspectRange
	cur := suspectRange{
		Start:   clampIndex(indices[0]-padding, total),
		End:     clampIndex(indices[0]+padding, total),
		Reasons: uniqueStrings(flags[indices[0]]),
	}
	for _, idx := range indices[1:] {
		lo := clampIndex(idx-padding, total)
		hi := clampIndex(idx+padding, total)
		if lo <= cur.End+1 {
			if hi > cur.End {
				cur.End = hi
			}
			cur.Reasons = uniqueStrings(append(cur.Reasons, flags[idx]...))
			continue
		}
		merged = append(merged, cur)
		cur = suspectRange{Start: lo, End: hi, Reasons: uniqueStrings(flags[idx])}
	}
	merged = append(merged, cur)
	return splitSuspectRanges(merged, maxBatch)
}

func splitSuspectRanges(ranges []suspectRange, maxBatch int) []suspectRange {
	if maxBatch <= 0 {
		return ranges
	}
	var out []suspectRange
	for _, sr := range ranges {
		for start := sr.Start; start <= sr.End; start += maxBatch {
			end := start + maxBatch - 1
			if end > sr.End {
				end = sr.End
			}
			out = append(out, suspectRange{
				Start:   start,
				End:     end,
				Reasons: sr.Reasons,
			})
		}
	}
	return out
}

func clampIndex(i, total int) int {
	if i < 0 {
		return 0
	}
	if i >= total {
		return total - 1
	}
	return i
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func appendUnique(list []string, item string) []string {
	for _, s := range list {
		if s == item {
			return list
		}
	}
	return append(list, item)
}

func countSuspectTitles(ranges []suspectRange) int {
	n := 0
	for _, sr := range ranges {
		n += sr.End - sr.Start + 1
	}
	return n
}
