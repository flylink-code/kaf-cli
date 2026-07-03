package ai

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// TaskKind 标识一个 AI 优化任务。
type TaskKind int

const (
	TaskStructure TaskKind = iota
	TaskTypography
	TaskNoise
	TaskMetadata
)

// Name 返回任务的中文名，用于日志。
func (t TaskKind) Name() string {
	switch t {
	case TaskStructure:
		return "章节结构分析"
	case TaskTypography:
		return "排版细节修正"
	case TaskNoise:
		return "噪音内容清理"
	case TaskMetadata:
		return "书籍简介生成"
	}
	return "未知任务"
}

// ---- Prompt 构造 ----

const structureSystem = `你是电子书章节结构校对专家，精通中文网文章节标题规律。
用户会给你一本小说经规则引擎粗解析后的章节标题列表（已带全局序号）。
请找出以下结构问题并返回 JSON 操作建议：
1. rename: 被污染/截断/带采集站水印的标题，给出干净的新标题。
2. merge: 内容本属同一章但被规则引擎拆成多节，给出序号组（保留首个，其余正文拼到首个）。
3. remove: 明显的噪音目录行、采集尾巴、无正文且与相邻章节重复的空目录行。

严格要求：
- 宁缺毋滥，只返回高置信度（>90%）的建议，避免改坏正常章节。
- 序号必须严格对应输入列表中的 [n] 编号。
- 不要臆测正文内容，只依据给定标题判断。
- 只输出合法 json 对象，不要 markdown 代码块，不要任何额外解释。
- 若无问题也必须返回：{"rename":{},"merge":[],"remove":[]}`

func buildStructurePrompt(titles []string, indexOffset int, reasons string) string {
	var b strings.Builder
	if reasons != "" {
		b.WriteString("本地规则标记的疑点：")
		b.WriteString(reasons)
		b.WriteString("\n\n")
	}
	b.WriteString("章节标题列表（共 ")
	fmt.Fprintf(&b, "%d", len(titles))
	b.WriteString(" 章")
	if indexOffset > 0 {
		fmt.Fprintf(&b, "，全局序号从 [%d] 起", indexOffset)
	}
	b.WriteString("）：\n")
	for i, t := range titles {
		fmt.Fprintf(&b, "[%d] %s\n", indexOffset+i, t)
	}
	b.WriteString("\n请只输出合法 json，格式如下：\n")
	b.WriteString(`{"rename":{"序号":"新标题"}, "merge":[[序号,序号,...]], "remove":[序号]}
示例：{"rename":{"3":"第3章 觉醒"}, "merge":[[12,13]], "remove":[7,8]}
若无问题：{"rename":{},"merge":[],"remove":[]}`)
	return b.String()
}

const typographySystem = `你是中文网文排版校对专家。用户会给你一本书前若干章节的正文抽样。
请诊断其中的排版问题（引号配对混乱、省略号不统一、标点全半角混用、
多余的空格/制表符、段首缩进异常等），并返回一组「原文片段→修正片段」的替换规则。
这些规则会被本地程序全文应用，所以：
- 每条 from 必须是精确的、能唯一定位错误模式的子串（通常是连续的标点/符号片段）。
- 不要给出整句替换，避免误伤。
- 优先覆盖高频、可机械替换的模式。
- 只返回 JSON 对象，不要任何额外解释。`

func buildTypographyPrompt(sample string) string {
	return "正文抽样：\n" + sample + "\n\n返回 JSON 格式：\n" +
		`{"replacements":[{"from":"原文片段","to":"修正片段"}], "notes":"可选说明"}
示例：{"replacements":[{"from":"………","to":"……"},{"from":".,","to":"。"}]}`
}

const noiseSystem = `你是中文网文内容清洗专家。用户会给你一本书前若干章节的正文抽样。
请识别其中的噪音内容（广告水印、采集站尾巴、乱码行、与正文无关的推广信息），
返回应整段删除的子串列表。这些子串会被本地程序用于命中即删除整行。
严格要求：
- 只返回确实是噪音的子串，宁缺毋滥，避免误删正文。
- 子串应是广告/水印中稳定出现的特征片段（如「更多精彩尽在」「请搜索」等）。
- 只返回 JSON 对象，不要任何额外解释。`

func buildNoisePrompt(sample string) string {
	return "正文抽样：\n" + sample + "\n\n返回 JSON 格式：\n" +
		`{"substrings":["噪音特征片段"], "notes":"可选说明"}`
}

const metadataSystem = `你是书籍内容分析专家。用户会给你一本书的书名和正文抽样。
请生成一段简洁的内容简介（80-200 字，不剧透关键反转）和可选的主题标签。
只返回 JSON 对象，不要任何额外解释。`

func buildMetadataPrompt(bookname, sample string) string {
	return fmt.Sprintf("书名：%s\n正文抽样：\n%s\n\n返回 JSON 格式：\n%s",
		bookname, sample,
		`{"summary":"内容简介","tags":"标签1,标签2"}`)
}

// ---- 调用与解析 ----

// runStructure 先本地扫描疑点，仅对可疑区间调用 AI；正常章节不上传。
func runStructure(ctx context.Context, c *Client, list SectionList, log func(string)) (StructurePlan, error) {
	if log == nil {
		log = func(string) {}
	}
	entries := flattenStructureEntries(list)
	total := len(entries)
	if total == 0 {
		return StructurePlan{Rename: map[int]string{}}, nil
	}
	titles := make([]string, total)
	for i, e := range entries {
		titles[i] = e.Title
	}

	suspects := detectStructureSuspects(entries)
	if len(suspects) == 0 {
		log("AI: 未发现可疑章节结构，跳过远程分析")
		return StructurePlan{Rename: map[int]string{}}, nil
	}
	log(fmt.Sprintf("AI: 本地发现 %d 处疑点，复核 %d 个标题（全书 %d 章）",
		len(suspects), countSuspectTitles(suspects), total))

	merged := StructurePlan{Rename: make(map[int]string)}
	for i, sr := range suspects {
		if i >= structureMaxAICalls {
			log(fmt.Sprintf("AI: 疑点过多，剩余 %d 处已跳过（上限 %d 次请求）",
				len(suspects)-i, structureMaxAICalls))
			break
		}
		reasons := strings.Join(sr.Reasons, "；")
		log(fmt.Sprintf("AI: 复核 [%d]-[%d]（%s）", sr.Start, sr.End, reasons))
		batch := titles[sr.Start : sr.End+1]
		plan, err := runStructureChunk(ctx, c, batch, sr.Start, total, reasons)
		if err != nil {
			return StructurePlan{}, err
		}
		appendStructurePlan(&merged, plan)
	}
	return sanitizeStructure(merged, total), nil
}

func runStructureChunk(ctx context.Context, c *Client, titles []string, indexOffset, total int, reasons string) (StructurePlan, error) {
	prompt := buildStructurePrompt(titles, indexOffset, reasons)
	resp, err := c.Chat(ctx, structureSystem, prompt, true)
	if err != nil {
		return StructurePlan{}, err
	}
	plan, err := parseStructureResponse(resp)
	if err != nil {
		// json_object 偶发非 JSON 说明，回退普通模式再试一次。
		resp2, err2 := c.Chat(ctx, structureSystem, prompt, false)
		if err2 != nil {
			return StructurePlan{}, err
		}
		plan, err = parseStructureResponse(resp2)
		if err != nil {
			return StructurePlan{}, err
		}
	}
	return sanitizeStructure(plan, total), nil
}

func parseStructureResponse(content string) (StructurePlan, error) {
	cleaned := strings.TrimSpace(stripJSONFence(content))
	if cleaned == "" {
		return StructurePlan{Rename: map[int]string{}}, nil
	}
	if !strings.Contains(cleaned, "{") {
		if isStructureNoChangeText(cleaned) {
			return StructurePlan{Rename: map[int]string{}}, nil
		}
		return StructurePlan{}, fmt.Errorf("响应中未找到 JSON 对象，片段: %s", truncateForLog(cleaned, 160))
	}
	plan, err := parseJSONObject[StructurePlan](content)
	if err != nil {
		return StructurePlan{}, err
	}
	if plan.Rename == nil {
		plan.Rename = map[int]string{}
	}
	return plan, nil
}

func isStructureNoChangeText(s string) bool {
	if s == "" {
		return true
	}
	lower := strings.ToLower(s)
	for _, kw := range []string{"无需", "无问题", "没有问题", "结构正常", "无需调整", "无需修改", "no change", "no issues"} {
		if strings.Contains(s, kw) || strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func appendStructurePlan(dst *StructurePlan, src StructurePlan) {
	for k, v := range src.Rename {
		dst.Rename[k] = v
	}
	dst.Merge = append(dst.Merge, src.Merge...)
	dst.Remove = append(dst.Remove, src.Remove...)
}

// sanitizeStructure 剔除索引越界与自相矛盾的建议。
func sanitizeStructure(plan StructurePlan, total int) StructurePlan {
	cleaned := StructurePlan{}
	cleaned.Rename = make(map[int]string)
	for idx, title := range plan.Rename {
		if idx < 0 || idx >= total {
			continue
		}
		title = strings.TrimSpace(title)
		if title == "" {
			continue
		}
		cleaned.Rename[idx] = title
	}
	for _, group := range plan.Merge {
		if len(group) < 2 {
			continue
		}
		valid := group[:0]
		sorted := append([]int(nil), group...)
		sort.Ints(sorted)
		seen := map[int]bool{}
		for _, idx := range sorted {
			if idx < 0 || idx >= total || seen[idx] {
				continue
			}
			seen[idx] = true
			valid = append(valid, idx)
		}
		if len(valid) >= 2 {
			cleaned.Merge = append(cleaned.Merge, valid)
		}
	}
	for _, idx := range plan.Remove {
		if idx < 0 || idx >= total {
			continue
		}
		cleaned.Remove = append(cleaned.Remove, idx)
	}
	return cleaned
}

func runTypography(ctx context.Context, c *Client, sample string) (TypographyPlan, error) {
	resp, err := c.Chat(ctx, typographySystem, buildTypographyPrompt(sample), true)
	if err != nil {
		return TypographyPlan{}, err
	}
	plan, err := parseJSONObject[TypographyPlan](resp)
	if err != nil {
		return TypographyPlan{}, err
	}
	return sanitizeTypography(plan), nil
}

func sanitizeTypography(plan TypographyPlan) TypographyPlan {
	cleaned := TypographyPlan{Notes: plan.Notes}
	for _, r := range plan.Replacements {
		r.From = strings.TrimSpace(r.From)
		r.To = strings.TrimSpace(r.To)
		if r.From == "" || r.From == r.To {
			continue
		}
		// 防止过长的 from（>40 字符）误伤正文。
		if len([]rune(r.From)) > 40 {
			continue
		}
		cleaned.Replacements = append(cleaned.Replacements, r)
	}
	return cleaned
}

func runNoise(ctx context.Context, c *Client, sample string) (NoisePlan, error) {
	resp, err := c.Chat(ctx, noiseSystem, buildNoisePrompt(sample), true)
	if err != nil {
		return NoisePlan{}, err
	}
	plan, err := parseJSONObject[NoisePlan](resp)
	if err != nil {
		return NoisePlan{}, err
	}
	return sanitizeNoise(plan), nil
}

func sanitizeNoise(plan NoisePlan) NoisePlan {
	cleaned := NoisePlan{Notes: plan.Notes}
	seen := map[string]bool{}
	for _, s := range plan.Substrings {
		s = strings.TrimSpace(s)
		if s == "" || len([]rune(s)) < 3 || len([]rune(s)) > 60 {
			continue
		}
		if seen[s] {
			continue
		}
		seen[s] = true
		cleaned.Substrings = append(cleaned.Substrings, s)
	}
	return cleaned
}

func runMetadata(ctx context.Context, c *Client, bookname, sample string) (MetadataPlan, error) {
	resp, err := c.Chat(ctx, metadataSystem, buildMetadataPrompt(bookname, sample), true)
	if err != nil {
		return MetadataPlan{}, err
	}
	plan, err := parseJSONObject[MetadataPlan](resp)
	if err != nil {
		return MetadataPlan{}, err
	}
	plan.Summary = strings.TrimSpace(plan.Summary)
	plan.Tags = strings.TrimSpace(plan.Tags)
	return plan, nil
}
