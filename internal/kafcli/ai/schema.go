package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Section 是 AI 子包自有的章节表示，避免与父包 kafcli 形成循环依赖。
// run.go 负责在 kafcli.Section 与 ai.Section 之间转换。
type Section struct {
	Title    string
	Content  string
	Sections []Section
}

// SectionList 是可变章节切片，便于在原地应用 patch。
type SectionList []Section

// Snapshot 返回切片的深拷贝，用于失败时回滚。
func (s SectionList) Snapshot() SectionList {
	out := make(SectionList, len(s))
	for i, sec := range s {
		out[i] = sec.Copy()
	}
	return out
}

// Copy 深拷贝单个 Section（含子章节）。
func (s Section) Copy() Section {
	c := s
	if len(s.Sections) > 0 {
		c.Sections = make([]Section, len(s.Sections))
		for i, sub := range s.Sections {
			c.Sections[i] = sub.Copy()
		}
	}
	return c
}

// Flatten 将嵌套章节展平为带全局序号的标题列表，供结构分析 prompt 使用。
// 返回 (扁平标题切片, 元素→原 SectionList 中索引 或 父索引+子索引)。
type Located struct {
	Parent int // -1 表示顶层章节
	Index  int // 在父下的位置；Parent==-1 时为顶层索引
}

func (s SectionList) Flatten() ([]string, []Located) {
	var titles []string
	var locs []Located
	for i, sec := range s {
		if len(sec.Sections) == 0 {
			titles = append(titles, sec.Title)
			locs = append(locs, Located{Parent: -1, Index: i})
			continue
		}
		for j, sub := range sec.Sections {
			titles = append(titles, sub.Title)
			locs = append(locs, Located{Parent: i, Index: j})
		}
	}
	return titles, locs
}

// ---- AI 输出契约 ----

// StructurePlan 是「章节结构分析」任务的 AI 返回契约。
// 所有操作均为对扁平章节列表的 patch；Apply 前会做索引校验。
type StructurePlan struct {
	// Rename: 章节序号 -> 新标题（修正被污染的标题）。
	Rename map[int]string `json:"rename,omitempty"`
	// Merge: 每组为应合并的章节序号列表，保留首个，其余内容拼接到首个。
	Merge [][]int `json:"merge,omitempty"`
	// Remove: 应删除的章节序号（噪音目录行、采集尾巴等，无正文或正文重复）。
	Remove []int `json:"remove,omitempty"`
}

// TypographyPlan 是「排版修正」任务的 AI 返回契约。
// AI 给出一组「原文片段 -> 修正片段」的替换规则，本地全量应用，
// 避免把整本书正文都送进 LLM。
type TypographyPlan struct {
	Replacements []ReplacementRule `json:"replacements,omitempty"`
	Notes        string            `json:"notes,omitempty"`
}

// ReplacementRule 单条替换规则。
type ReplacementRule struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// NoisePlan 是「噪音清理」任务的 AI 返回契约。
// 给出应删除的文本模式（子串或正则），本地全量匹配删除。
type NoisePlan struct {
	// Substrings: 命中即整段删除该行的子串（广告/水印/采集站尾巴）。
	Substrings []string `json:"substrings,omitempty"`
	Notes      string   `json:"notes,omitempty"`
}

// MetadataPlan 是「生成书籍简介元数据」任务的 AI 返回契约。
type MetadataPlan struct {
	Summary string `json:"summary,omitempty"`
	Tags    string `json:"tags,omitempty"`
}

// parseJSONObject 解析 AI 返回的内容为指定结构。
// 容错：去掉 ```json / ``` 围栏、提取首个 { 到末尾 } 的子串。
func parseJSONObject[T any](content string) (T, error) {
	var zero T
	cleaned := strings.TrimSpace(stripJSONFence(content))
	if cleaned == "" {
		return zero, errors.New("响应为空")
	}
	if !strings.Contains(cleaned, "{") {
		return zero, fmt.Errorf("响应中未找到 JSON 对象，片段: %s", truncateForLog(cleaned, 160))
	}
	cleaned = extractJSONObject(cleaned)
	var out T
	if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
		return zero, fmt.Errorf("JSON 解析失败: %w", err)
	}
	return out, nil
}

// truncateForLog 截断日志/错误中的 AI 原文，避免刷屏。
func truncateForLog(s string, limit int) string {
	s = strings.TrimSpace(s)
	if limit <= 0 || len([]rune(s)) <= limit {
		return s
	}
	runes := []rune(s)
	return string(runes[:limit]) + "…"
}

// stripJSONFence 去除 markdown 代码围栏。
func stripJSONFence(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

// extractJSONObject 截取首个 '{' 到最后一个 '}' 之间的内容，
// 用于处理模型在 JSON 前后掺杂自然语言说明的情况。
func extractJSONObject(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end < 0 || end <= start {
		return s
	}
	return s[start : end+1]
}
