package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RefineOptions 控制 Refine 的行为。零值表示全部关闭，Refine 会原样返回。
type RefineOptions struct {
	Enabled        bool
	Client         *Client
	Bookname       string
	SampleChars    int // 抽样正文字符上限；0 表示仅做结构分析（不抽正文）
	DoStructure    bool
	DoTypography   bool
	DoNoise        bool
	DoMetadata     bool
	MetadataSink   func(MetadataPlan) // 元数据结果回调（写入 Book 字段）
	Logger         func(string)       // 进度日志，接现有 fmt.Println 流
	RequestTimeout time.Duration      // 单任务超时，默认与 client 一致
}

// Refine 是 AI 后处理入口。
// 在 Parse 之后、Convert 之前对 SectionList 做增强。
// 任何任务失败均告警并跳过，绝不返回错误中断流程；
// 最坏情况下原样返回输入。
func Refine(list SectionList, opts RefineOptions) (SectionList, error) {
	log := opts.Logger
	if log == nil {
		log = func(string) {}
	}
	if !opts.Enabled {
		log("AI 优化：未启用，已跳过")
		return list, nil
	}
	if opts.Client == nil || !opts.Client.Ready() {
		log("AI 优化：缺少 API Key 或 Model，已跳过")
		return list, nil
	}

	// 默认只做结构分析；结构分析是其它任务的基础，先跑。
	ctx := context.Background()
	if opts.RequestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.RequestTimeout)
		defer cancel()
	}

	// 先拍快照，任何任务失败时回滚到该状态。
	snapshot := list.Snapshot()
	current := list

	// ---- 结构分析（默认开） ----
	if opts.DoStructure {
		log("AI: 正在分析章节结构...")
		titles, locs := current.Flatten()
		if len(titles) == 0 {
			log("AI: 未发现章节，跳过结构分析")
		} else {
			plan, err := runStructure(ctx, opts.Client, current, log)
			if err != nil {
				log("AI: 结构分析失败，已跳过: " + err.Error())
			} else {
				changed := applyStructure(&current, plan, locs, log)
				if changed {
					log(fmt.Sprintf("AI: 结构分析完成，已修正 %d 处", countStructureOps(plan)))
				} else {
					log("AI: 结构分析完成，无需调整")
				}
			}
		}
	}

	// ---- 需要正文抽样的任务 ----
	needSample := opts.DoTypography || opts.DoNoise || opts.DoMetadata
	sample := ""
	if needSample && opts.SampleChars > 0 {
		sample = buildContentSample(current, opts.SampleChars)
		if sample == "" {
			log("AI: 未取到正文抽样，跳过依赖正文的任务")
			needSample = false
		}
	} else if needSample && opts.SampleChars <= 0 {
		log("AI: 排版/噪音/简介任务需正文抽样，但 SampleChars<=0，已跳过")
		needSample = false
	}

	if needSample {
		if opts.DoTypography {
			current = runTypographyTask(ctx, opts.Client, sample, current, log)
		}
		if opts.DoNoise {
			current = runNoiseTask(ctx, opts.Client, sample, current, log)
		}
		if opts.DoMetadata {
			runMetadataTask(ctx, opts.Client, opts.Bookname, sample, opts.MetadataSink, log)
		}
	}

	_ = snapshot // 快照保留以便未来支持「对比/回滚」UI；当前直接采用 current
	return current, nil
}

// ---- patch 应用 ----

// applyStructure 将 StructurePlan 应用到 SectionList。
// locs 由 Flatten 产生，把扁平序号映射回原结构。
func applyStructure(list *SectionList, plan StructurePlan, locs []Located, log func(string)) bool {
	changed := false

	// 1. rename
	for idx, title := range plan.Rename {
		if setSectionTitle(list, locs, idx, title) {
			changed = true
			log(fmt.Sprintf("AI: 重命名 [%d] -> %s", idx, title))
		}
	}

	// 2. merge —— 合并到组内首个；记录被合并的序号，供后续 remove 跳过
	mergedAway := map[int]bool{}
	for _, group := range plan.Merge {
		if merged, ok := mergeSections(list, locs, group, log); ok {
			changed = true
			for _, idx := range group[1:] {
				mergedAway[idx] = true
			}
			_ = merged
		}
	}

	// 3. remove —— 合并被删除序号，含已合并掉的
	removeSet := map[int]bool{}
	for _, idx := range plan.Remove {
		removeSet[idx] = true
	}
	for idx := range mergedAway {
		removeSet[idx] = true
	}
	if len(removeSet) > 0 {
		if removed := removeSections(list, locs, removeSet); removed > 0 {
			changed = true
			log(fmt.Sprintf("AI: 删除 %d 个冗余章节", removed))
		}
	}

	return changed
}

func setSectionTitle(list *SectionList, locs []Located, idx int, title string) bool {
	if idx < 0 || idx >= len(locs) {
		return false
	}
	loc := locs[idx]
	if loc.Parent < 0 {
		if loc.Index >= len(*list) {
			return false
		}
		if (*list)[loc.Index].Title == title {
			return false
		}
		(*list)[loc.Index].Title = title
		return true
	}
	if loc.Parent >= len(*list) {
		return false
	}
	subs := (*list)[loc.Parent].Sections
	if loc.Index >= len(subs) {
		return false
	}
	if subs[loc.Index].Title == title {
		return false
	}
	subs[loc.Index].Title = title
	(*list)[loc.Parent].Sections = subs
	return true
}

// mergeSections 把 group[1:] 的正文拼到 group[0]，标题保留 group[0]。
// 实际删除留给 remove 阶段统一处理（避免索引错乱）。
func mergeSections(list *SectionList, locs []Located, group []int, log func(string)) (bool, bool) {
	if len(group) < 2 {
		return false, false
	}
	// 收集每个序号对应的正文
	firstLoc := locs[group[0]]
	var combined strings.Builder
	if sec, ok := sectionAt(*list, firstLoc); ok {
		combined.WriteString(sec.Content)
	}
	for _, idx := range group[1:] {
		if idx < 0 || idx >= len(locs) {
			continue
		}
		if sec, ok := sectionAt(*list, locs[idx]); ok && sec.Content != "" {
			if combined.Len() > 0 && !strings.HasSuffix(combined.String(), "\n") {
				combined.WriteString("\n")
			}
			combined.WriteString(sec.Content)
		}
	}
	if combined.Len() == 0 {
		return false, false
	}
	ok := setSectionContent(list, firstLoc, combined.String())
	if ok {
		log(fmt.Sprintf("AI: 合并章节 %v 至 [%d]", group, group[0]))
	}
	return ok, true
}

func setSectionContent(list *SectionList, loc Located, content string) bool {
	if loc.Parent < 0 {
		if loc.Index >= len(*list) {
			return false
		}
		(*list)[loc.Index].Content = content
		return true
	}
	if loc.Parent >= len(*list) {
		return false
	}
	subs := (*list)[loc.Parent].Sections
	if loc.Index >= len(subs) {
		return false
	}
	subs[loc.Index].Content = content
	(*list)[loc.Parent].Sections = subs
	return true
}

func sectionAt(list SectionList, loc Located) (Section, bool) {
	if loc.Parent < 0 {
		if loc.Index >= len(list) {
			return Section{}, false
		}
		return list[loc.Index], true
	}
	if loc.Parent >= len(list) {
		return Section{}, false
	}
	subs := list[loc.Parent].Sections
	if loc.Index >= len(subs) {
		return Section{}, false
	}
	return subs[loc.Index], true
}

// removeSections 按扁平序号删除章节。先删子章节，再删顶层。
func removeSections(list *SectionList, locs []Located, removeSet map[int]bool) int {
	// 按父分组
	childRemovals := map[int]map[int]bool{} // parent -> set of child idx
	var topRemovals []int
	for idx, loc := range locs {
		if !removeSet[idx] {
			continue
		}
		if loc.Parent < 0 {
			topRemovals = append(topRemovals, loc.Index)
		} else {
			if childRemovals[loc.Parent] == nil {
				childRemovals[loc.Parent] = map[int]bool{}
			}
			childRemovals[loc.Parent][loc.Index] = true
		}
	}
	removed := 0
	// 删子章节
	for parent, set := range childRemovals {
		if parent >= len(*list) {
			continue
		}
		subs := (*list)[parent].Sections
		newSubs := subs[:0]
		for j, sub := range subs {
			if set[j] {
				removed++
				continue
			}
			newSubs = append(newSubs, sub)
		}
		(*list)[parent].Sections = newSubs
	}
	// 删顶层章节
	if len(topRemovals) > 0 {
		topSet := map[int]bool{}
		for _, i := range topRemovals {
			topSet[i] = true
		}
		newList := (*list)[:0]
		for i, sec := range *list {
			if topSet[i] {
				removed++
				continue
			}
			newList = append(newList, sec)
		}
		*list = newList
	}
	return removed
}

func countStructureOps(plan StructurePlan) int {
	n := len(plan.Rename) + len(plan.Merge) + len(plan.Remove)
	return n
}

// ---- 正文抽样与剩余任务 ----

// buildContentSample 从前若干章收集正文，直到达到 charLimit 字符。
func buildContentSample(list SectionList, charLimit int) string {
	var b strings.Builder
	for _, sec := range list {
		if b.Len() >= charLimit {
			break
		}
		appendSampleText(&b, sec, charLimit)
		for _, sub := range sec.Sections {
			if b.Len() >= charLimit {
				break
			}
			appendSampleText(&b, sub, charLimit)
		}
	}
	s := strings.TrimSpace(b.String())
	if len([]rune(s)) > charLimit {
		s = string([]rune(s)[:charLimit])
	}
	return s
}

func appendSampleText(b *strings.Builder, sec Section, limit int) {
	text := strings.TrimSpace(plainText(sec.Content))
	if text == "" {
		return
	}
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	b.WriteString(text)
}

// plainText 把 Section.Content 中的 HTML 标签去掉，便于 AI 阅读。
// 当前 Content 形如 <p class="content">正文</p>，简单剥标签即可。
func plainText(html string) string {
	var b strings.Builder
	inTag := false
	for _, r := range html {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
			b.WriteByte('\n')
		default:
			if !inTag {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

func runTypographyTask(ctx context.Context, c *Client, sample string, list SectionList, log func(string)) SectionList {
	log("AI: 正在分析排版问题...")
	plan, err := runTypography(ctx, c, sample)
	if err != nil {
		log("AI: 排版分析失败，已跳过: " + err.Error())
		return list
	}
	if len(plan.Replacements) == 0 {
		log("AI: 未发现可机械修正的排版问题")
		return list
	}
	changed := applyReplacements(list, plan.Replacements)
	log(fmt.Sprintf("AI: 排版修正完成，应用 %d 条规则，命中 %d 处", len(plan.Replacements), changed))
	return list
}

// applyReplacements 就地把替换规则应用到所有章节正文，返回总命中次数。
func applyReplacements(list SectionList, rules []ReplacementRule) int {
	hits := 0
	for i := range list {
		if list[i].Content != "" {
			new, n := applyRules(list[i].Content, rules)
			if n > 0 {
				list[i].Content = new
				hits += n
			}
		}
		for j := range list[i].Sections {
			if list[i].Sections[j].Content == "" {
				continue
			}
			new, n := applyRules(list[i].Sections[j].Content, rules)
			if n > 0 {
				list[i].Sections[j].Content = new
				hits += n
			}
		}
	}
	return hits
}

func applyRules(text string, rules []ReplacementRule) (string, int) {
	total := 0
	for _, r := range rules {
		count := strings.Count(text, r.From)
		if count > 0 {
			text = strings.ReplaceAll(text, r.From, r.To)
			total += count
		}
	}
	return text, total
}

func runNoiseTask(ctx context.Context, c *Client, sample string, list SectionList, log func(string)) SectionList {
	log("AI: 正在识别噪音内容...")
	plan, err := runNoise(ctx, c, sample)
	if err != nil {
		log("AI: 噪音识别失败，已跳过: " + err.Error())
		return list
	}
	if len(plan.Substrings) == 0 {
		log("AI: 未发现噪音特征")
		return list
	}
	removed := removeNoiseLines(list, plan.Substrings)
	log(fmt.Sprintf("AI: 噪音清理完成，按 %d 个特征删除 %d 行", len(plan.Substrings), removed))
	return list
}

// removeNoiseLines 删除正文 HTML 段落中命中任一噪音子串的 <p> 段。
// 返回删除的段数。
func removeNoiseLines(list SectionList, substrings []string) int {
	removed := 0
	for i := range list {
		if list[i].Content != "" {
			new, n := filterNoiseParagraphs(list[i].Content, substrings)
			if n > 0 {
				list[i].Content = new
				removed += n
			}
		}
		for j := range list[i].Sections {
			if list[i].Sections[j].Content == "" {
				continue
			}
			new, n := filterNoiseParagraphs(list[i].Sections[j].Content, substrings)
			if n > 0 {
				list[i].Sections[j].Content = new
				removed += n
			}
		}
	}
	return removed
}

// filterNoiseParagraphs 解析 <p class="content">...</p> 段，命中噪音子串则丢弃。
func filterNoiseParagraphs(html string, substrings []string) (string, int) {
	// 段以 </p> 分隔，每段以 <p ...> 起始
	segs := strings.Split(html, "</p>")
	out := make([]string, 0, len(segs))
	removed := 0
	for _, seg := range segs {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		// 取 <p ...> 之后的纯文本
		text := seg
		if gt := strings.Index(seg, ">"); gt >= 0 {
			text = seg[gt+1:]
		}
		hit := false
		for _, sub := range substrings {
			if strings.Contains(text, sub) {
				hit = true
				break
			}
		}
		if hit {
			removed++
			continue
		}
		out = append(out, seg+"</p>")
	}
	return strings.Join(out, ""), removed
}

func runMetadataTask(ctx context.Context, c *Client, bookname, sample string, sink func(MetadataPlan), log func(string)) {
	log("AI: 正在生成书籍简介...")
	plan, err := runMetadata(ctx, c, bookname, sample)
	if err != nil {
		log("AI: 简介生成失败，已跳过: " + err.Error())
		return
	}
	if plan.Summary == "" && plan.Tags == "" {
		log("AI: 未生成有效简介")
		return
	}
	if sink != nil {
		sink(plan)
	}
	log("AI: 简介生成完成")
}
