package ai

import (
	"fmt"
	"strings"
	"testing"
)

func TestDetectStructureSuspectsWatermark(t *testing.T) {
	entries := []structureEntry{
		{Title: "第1章 开始", Empty: false},
		{Title: "请收藏本站最新章节", Empty: true},
		{Title: "第2章 继续", Empty: false},
	}
	ranges := detectStructureSuspects(entries)
	if len(ranges) == 0 {
		t.Fatal("expected suspect range for watermark title")
	}
	if ranges[0].Start > 1 || ranges[0].End < 1 {
		t.Fatalf("expected range covering index 1, got [%d]-[%d]", ranges[0].Start, ranges[0].End)
	}
}

func TestDetectStructureSuspectsDuplicateChapter(t *testing.T) {
	entries := []structureEntry{
		{Title: "第5章 甲", Empty: false},
		{Title: "第5章 乙", Empty: false},
	}
	ranges := detectStructureSuspects(entries)
	if len(ranges) == 0 {
		t.Fatal("expected suspect range for duplicate chapter key")
	}
}

func TestDetectStructureSuspectsCleanBook(t *testing.T) {
	var entries []structureEntry
	for i := 1; i <= 200; i++ {
		entries = append(entries, structureEntry{
			Title:     fmt.Sprintf("第%d章 正文", i),
			Empty:     false,
			CharCount: 2000,
		})
	}
	if len(detectStructureSuspects(entries)) != 0 {
		t.Fatal("clean chapter list should have no suspects")
	}
}

func TestDetectStructureSuspectsPureDigits(t *testing.T) {
	entries := []structureEntry{
		{Title: "第1章 开始", Empty: false, CharCount: 3000},
		{Title: "53", Empty: false, CharCount: 15, SnippetHead: "02. 灵梦最好了 53"},
		{Title: "第2章 结束", Empty: false, CharCount: 2500},
	}
	ranges := detectStructureSuspects(entries)
	if len(ranges) == 0 {
		t.Fatal("expected suspect range for pure digits title")
	}
	if ranges[0].Start > 1 || ranges[0].End < 1 {
		t.Fatalf("expected range covering index 1, got [%d]-[%d]", ranges[0].Start, ranges[0].End)
	}
}

func TestDetectStructureSuspectsShortContent(t *testing.T) {
	entries := []structureEntry{
		{Title: "第1章 开始", Empty: false, CharCount: 3000},
		{Title: "奇怪的短句子", Empty: false, CharCount: 12, SnippetHead: "只有一句话。"},
		{Title: "第2章 结束", Empty: false, CharCount: 2500},
	}
	ranges := detectStructureSuspects(entries)
	if len(ranges) == 0 {
		t.Fatal("expected suspect range for very short content snippet")
	}
}

func TestMergeSuspectFlagsSplitsLargeRange(t *testing.T) {
	flags := map[int][]string{0: {"test"}, 50: {"test"}}
	ranges := mergeSuspectFlags(flags, 100, 0, 30)
	if len(ranges) < 2 {
		t.Fatalf("expected split ranges, got %d", len(ranges))
	}
	for _, sr := range ranges {
		if sr.End-sr.Start+1 > 30 {
			t.Fatalf("range too large: [%d]-[%d]", sr.Start, sr.End)
		}
	}
}

func TestStructureSkipsAIWhenNoSuspects(t *testing.T) {
	calls := 0
	c := newCountingMockClient(t, &calls, `{"rename":{}}`)
	list := SectionList{}
	for i := 1; i <= 50; i++ {
		list = append(list, Section{
			Title:   fmt.Sprintf("第%d章 正文", i),
			Content: "<p class=\"content\">" + strings.Repeat("这是正常的章节正文段落。", 10) + "</p>",
		})
	}
	var logs []string
	_, _ = Refine(list, RefineOptions{
		Enabled: true, Client: c, DoStructure: true,
		Logger: func(s string) { logs = append(logs, s) },
	})
	if calls != 0 {
		t.Fatalf("expected 0 API calls for clean book, got %d", calls)
	}
	if !containsAny(logs, "跳过远程分析") {
		t.Fatalf("expected skip log, got %v", logs)
	}
}

func TestStructureCallsAIForSuspects(t *testing.T) {
	calls := 0
	c := newCountingMockClient(t, &calls, `{"remove":[1]}`)
	list := SectionList{
		{Title: "第1章", Content: "<p>x</p>"},
		{Title: "请收藏本站", Content: ""},
		{Title: "第2章", Content: "<p>y</p>"},
	}
	out, _ := Refine(list, RefineOptions{Enabled: true, Client: c, DoStructure: true})
	if calls == 0 {
		t.Fatal("expected API call when suspects present")
	}
	if len(out) != 2 {
		t.Fatalf("expected remove to shrink list to 2, got %d", len(out))
	}
}

func newCountingMockClient(t *testing.T, calls *int, resp string) *Client {
	t.Helper()
	return newMockClient(t, 200, func([]byte) (string, error) {
		*calls++
		return resp, nil
	})
}

func TestStructurePromptIncludesReasons(t *testing.T) {
	prompt := buildStructurePrompt([]structureEntry{{Title: "第1章"}}, 0, "标题含水印")
	if !strings.Contains(prompt, "疑点") || !strings.Contains(prompt, "标题含水印") {
		t.Fatalf("expected reasons in prompt, got %q", prompt)
	}
}

func TestDetectStructureSuspectsChineseOrdinalJump(t *testing.T) {
	entries := []structureEntry{
		{Title: "第一章 开始", Empty: false, CharCount: 2000},
		{Title: "第二章 过程", Empty: false, CharCount: 2000},
		{Title: "第五章 突然跳跃", Empty: false, CharCount: 2000},
	}
	ranges := detectStructureSuspects(entries)
	if len(ranges) == 0 {
		t.Fatal("expected suspect range for Chinese ordinal chapter jump")
	}
	found := false
	for _, r := range ranges {
		for _, re := range r.Reasons {
			if strings.Contains(re, "章号跳号") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("expected reason to include 章号跳号, got %+v", ranges)
	}
}
