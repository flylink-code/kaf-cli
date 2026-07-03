package kafcli

import (
	"fmt"

	"github.com/ystyle/kaf-cli/internal/kafcli/ai"
)

// Run 执行完整的电子书转换流程。
func Run(book *Book, version, secret, measurement string) error {
	if err := book.Check(version); err != nil {
		return err
	}
	Analytics(version, secret, measurement, book.Format)
	book.ToString()
	if err := book.Parse(); err != nil {
		return fmt.Errorf("解析失败: %w", err)
	}
	// AI 后处理：可选增强，未配置或失败时静默降级到规则引擎结果。
	if book.AIOptions.Enabled {
		applyAIRefine(book)
	}
	if err := book.Convert(); err != nil {
		return fmt.Errorf("转换失败: %w", err)
	}
	return nil
}

// applyAIRefine 把 Book.SectionList 转入 ai 子包做精炼，再写回。
// 转换过程中产生的元数据简介通过回调写回 Book 字段。
func applyAIRefine(book *Book) {
	list := toAISections(book.SectionList)
	opts := ai.RefineOptions{
		Enabled:      book.AIOptions.Enabled,
		Client:       book.AIOptions.Client,
		Bookname:     book.Bookname,
		SampleChars:  book.AIOptions.SampleChars,
		DoStructure:  book.AIOptions.DoStructure,
		DoTypography: book.AIOptions.DoTypography,
		DoNoise:      book.AIOptions.DoNoise,
		DoMetadata:   book.AIOptions.DoMetadata,
		Logger: func(msg string) {
			fmt.Println(msg)
		},
		MetadataSink: func(plan ai.MetadataPlan) {
			if book.AISummary == "" && plan.Summary != "" {
				book.AISummary = plan.Summary
			}
			if book.AITags == "" && plan.Tags != "" {
				book.AITags = plan.Tags
			}
		},
	}
	refined, err := ai.Refine(list, opts)
	if err != nil {
		fmt.Printf("AI 优化已跳过: %s\n", err)
		return
	}
	book.SectionList = fromAISections(refined)
}

// toAISections 把 kafcli.Section 列表转为 ai.Section 列表。
func toAISections(list []Section) ai.SectionList {
	out := make(ai.SectionList, len(list))
	for i, s := range list {
		out[i] = toAISection(s)
	}
	return out
}

func toAISection(s Section) ai.Section {
	out := ai.Section{Title: s.Title, Content: s.Content}
	if len(s.Sections) > 0 {
		out.Sections = make([]ai.Section, len(s.Sections))
		for i, sub := range s.Sections {
			out.Sections[i] = toAISection(sub)
		}
	}
	return out
}

// fromAISections 是 toAISections 的逆转换。
func fromAISections(list ai.SectionList) []Section {
	out := make([]Section, len(list))
	for i, s := range list {
		out[i] = fromAISection(s)
	}
	return out
}

func fromAISection(s ai.Section) Section {
	out := Section{Title: s.Title, Content: s.Content}
	if len(s.Sections) > 0 {
		out.Sections = make([]Section, len(s.Sections))
		for i, sub := range s.Sections {
			out.Sections[i] = fromAISection(sub)
		}
	}
	return out
}

// GUIOptions 图形界面转换选项。
type GUIOptions struct {
	Filename        string
	Cover           string
	Author          string
	Format          string
	Match           string
	VolumeMatch     string
	DedupTitle      bool
	Tips            bool
	NormalizeQuotes bool
	// AI 后处理选项。Enabled=false 时忽略其余 AI 字段。
	AI AIRefineOptions
}

// NewBookGUI 根据 GUI 选项创建 Book。
func NewBookGUI(opts GUIOptions) *Book {
	book := &Book{
		Filename:        opts.Filename,
		DedupTitle:      opts.DedupTitle,
		Tips:            opts.Tips,
		NormalizeQuotes: opts.NormalizeQuotes,
		Format:          opts.Format,
		Lang:            "zh",
		AIOptions:       opts.AI,
	}
	if opts.Author != "" {
		book.Author = opts.Author
	}
	if opts.Match != "" {
		book.Match = opts.Match
	}
	if opts.VolumeMatch != "" {
		book.VolumeMatch = opts.VolumeMatch
	}
	book.SetDefault()
	if book.Format == "" {
		book.Format = "all"
	}
	if opts.Cover != "" {
		book.Cover = opts.Cover
	} else {
		book.Cover = "none"
	}
	return book
}
