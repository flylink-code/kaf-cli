package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newMockClient 启动一个 httptest 服务模拟 OpenAI 兼容端点。
// handler 决定每次请求返回的 assistant 内容（content 字段）。
func newMockClient(t *testing.T, status int, contentFn func(body []byte) (string, error)) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		content, err := contentFn(body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if status != 200 {
			http.Error(w, "simulated error", status)
			return
		}
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": content}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return NewClient(ClientConfig{
		BaseURL: srv.URL,
		APIKey:  "test-key",
		Model:   "test-model",
		Timeout: 5 * time.Second,
	})
}

func fixedContent(s string) func([]byte) (string, error) {
	return func([]byte) (string, error) { return s, nil }
}

// 响应内容用读到的请求体确认是哪个任务，便于一个 server 应对多任务。
func contentByKeyword(cases map[string]string) func([]byte) (string, error) {
	return func(body []byte) (string, error) {
		s := string(body)
		for keyword, resp := range cases {
			if strings.Contains(s, keyword) {
				return resp, nil
			}
		}
		return `{"rename":{}}`, nil
	}
}

func TestRefineDisabledReturnsOriginal(t *testing.T) {
	list := SectionList{{Title: "第1章", Content: "<p class=\"content\">正文</p>"}}
	out, err := Refine(list, RefineOptions{Enabled: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 || out[0].Title != "第1章" {
		t.Fatalf("expected original list unchanged, got %+v", out)
	}
}

func TestRefineNotReadyReturnsOriginal(t *testing.T) {
	list := SectionList{{Title: "第1章"}}
	out, _ := Refine(list, RefineOptions{Enabled: true, Client: NewClient(ClientConfig{})})
	if len(out) != 1 {
		t.Fatalf("expected original list when client not ready")
	}
}

func TestStructureRenameApplies(t *testing.T) {
	resp := `{"rename":{"0":"第1章 觉醒"}}`
	c := newMockClient(t, 200, fixedContent(resp))
	var logs []string
	list := SectionList{{Title: "请收藏本站-第1章", Content: "<p>x</p>"}}
	out, _ := Refine(list, RefineOptions{
		Enabled: true, Client: c, DoStructure: true,
		Logger: func(s string) { logs = append(logs, s) },
	})
	if out[0].Title != "第1章 觉醒" {
		t.Fatalf("expected rename applied, got %q", out[0].Title)
	}
	if !containsAny(logs, "重命名") {
		t.Fatalf("expected rename log, got %v", logs)
	}
}

func TestStructureRemoveApplies(t *testing.T) {
	resp := `{"remove":[1]}`
	c := newMockClient(t, 200, fixedContent(resp))
	list := SectionList{
		{Title: "第1章", Content: "<p>a</p>"},
		{Title: "请收藏本站-广告行", Content: "<p>广告</p>"},
		{Title: "第2章", Content: "<p>b</p>"},
	}
	out, _ := Refine(list, RefineOptions{Enabled: true, Client: c, DoStructure: true})
	if len(out) != 2 {
		t.Fatalf("expected 2 sections after remove, got %d", len(out))
	}
	if out[1].Title != "第2章" {
		t.Fatalf("unexpected remaining title: %q", out[1].Title)
	}
}

func TestStructureMergeConcatenatesContent(t *testing.T) {
	resp := `{"merge":[[0,1]]}`
	c := newMockClient(t, 200, fixedContent(resp))
	list := SectionList{
		{Title: "第1章 上", Content: "<p>段一</p>"},
		{Title: "第1章 下", Content: "<p>段二</p>"},
		{Title: "第2章", Content: "<p>段三</p>"},
	}
	out, _ := Refine(list, RefineOptions{Enabled: true, Client: c, DoStructure: true})
	if len(out) != 2 {
		t.Fatalf("expected 2 sections after merge, got %d", len(out))
	}
	if !strings.Contains(out[0].Content, "段一") || !strings.Contains(out[0].Content, "段二") {
		t.Fatalf("expected merged content, got %q", out[0].Content)
	}
}

func TestStructureIndexOutOfRangeIgnored(t *testing.T) {
	// 越界索引 99 应被 sanitize 剔除，不影响正常 0
	resp := `{"rename":{"0":"第1章 新","99":"不应生效"}}`
	c := newMockClient(t, 200, fixedContent(resp))
	list := SectionList{{Title: "请收藏本站-第1章", Content: "<p>x</p>"}}
	out, _ := Refine(list, RefineOptions{Enabled: true, Client: c, DoStructure: true})
	if out[0].Title != "第1章 新" {
		t.Fatalf("expected valid rename applied, got %q", out[0].Title)
	}
}

func TestStructureGarbledJSONDegradesGracefully(t *testing.T) {
	// 损坏 JSON + markdown 围栏包裹的正常 JSON，验证 stripFence 生效
	calls := 0
	c := newMockClient(t, 200, func([]byte) (string, error) {
		calls++
		return "这不是JSON：```json\n{\"rename\":{\"0\":\"第1章 修\"}}\n```", nil
	})
	list := SectionList{{Title: "请收藏本站-第1章", Content: "<p>x</p>"}}
	out, _ := Refine(list, RefineOptions{Enabled: true, Client: c, DoStructure: true})
	if out[0].Title != "第1章 修" {
		t.Fatalf("fenced JSON should be parsed, got %q", out[0].Title)
	}
	if calls == 0 {
		t.Fatal("expected at least one call")
	}
}

func TestStructureHTTPErrorDegrades(t *testing.T) {
	c := newMockClient(t, 500, fixedContent(`{"rename":{}}`))
	var logged string
	list := SectionList{{Title: "请收藏本站-第1章", Content: "<p>x</p>"}}
	out, _ := Refine(list, RefineOptions{
		Enabled: true, Client: c, DoStructure: true,
		Logger: func(s string) { logged = s },
	})
	// HTTP 500 时应保持原样
	if out[0].Title != "请收藏本站-第1章" {
		t.Fatalf("expected unchanged on HTTP error, got %q", out[0].Title)
	}
	if !strings.Contains(logged, "结构分析失败") {
		t.Fatalf("expected failure log, got %q", logged)
	}
}

func TestTypographyReplacementsApply(t *testing.T) {
	resp := `{"replacements":[{"from":"………","to":"……"},{"from":".,","to":"。"}]}`
	c := newMockClient(t, 200, contentByKeyword(map[string]string{
		"排版": resp,
	}))
	list := SectionList{{
		Title:   "第1章",
		Content: `<p class="content">正文………结尾。</p><p class="content">第二段.,完毕</p>`,
	}}
	out, _ := Refine(list, RefineOptions{
		Enabled: true, Client: c, DoTypography: true, SampleChars: 500,
	})
	if !strings.Contains(out[0].Content, "正文……结尾") {
		t.Fatalf("expected ellipsis fixed, got %q", out[0].Content)
	}
	if !strings.Contains(out[0].Content, "第二段。完毕") {
		t.Fatalf("expected comma fixed, got %q", out[0].Content)
	}
}

func TestNoiseRemovesMatchingParagraphs(t *testing.T) {
	resp := `{"substrings":["更多精彩尽在","请搜索"]}`
	c := newMockClient(t, 200, contentByKeyword(map[string]string{
		"噪音": resp,
	}))
	list := SectionList{{
		Title: "第1章",
		Content: `<p class="content">正常正文。</p>` +
			`<p class="content">更多精彩尽在 xx 网站</p>` +
			`<p class="content">继续正文。</p>`,
	}}
	out, _ := Refine(list, RefineOptions{
		Enabled: true, Client: c, DoNoise: true, SampleChars: 500,
	})
	if strings.Contains(out[0].Content, "更多精彩尽在") {
		t.Fatalf("noise paragraph should be removed, got %q", out[0].Content)
	}
	if !strings.Contains(out[0].Content, "正常正文") || !strings.Contains(out[0].Content, "继续正文") {
		t.Fatalf("normal paragraphs should remain, got %q", out[0].Content)
	}
}

func TestMetadataInvokesSink(t *testing.T) {
	resp := `{"summary":"一段简介。","tags":"玄幻,冒险"}`
	c := newMockClient(t, 200, contentByKeyword(map[string]string{
		"简介": resp,
	}))
	var got MetadataPlan
	list := SectionList{{Title: "第1章", Content: "<p class=\"content\">正文抽样</p>"}}
	_, _ = Refine(list, RefineOptions{
		Enabled: true, Client: c, DoMetadata: true, SampleChars: 500, Bookname: "测试书",
		MetadataSink: func(m MetadataPlan) { got = m },
	})
	if got.Summary != "一段简介。" {
		t.Fatalf("expected summary in sink, got %+v", got)
	}
}

func TestTypographySkippedWhenSampleCharsZero(t *testing.T) {
	c := newMockClient(t, 200, fixedContent(`{"replacements":[]}`))
	var logs []string
	list := SectionList{{Title: "请收藏本站-第1章", Content: "<p>x</p>"}}
	_, _ = Refine(list, RefineOptions{
		Enabled: true, Client: c, DoTypography: true, SampleChars: 0,
		Logger: func(s string) { logs = append(logs, s) },
	})
	if !containsAny(logs, "跳过") {
		t.Fatalf("expected skip log when SampleChars=0, got %v", logs)
	}
}

func TestStripJSONFence(t *testing.T) {
	cases := map[string]string{
		"```json\n{\"a\":1}\n```":         `{"a":1}`,
		"```\n{\"a\":1}\n```":             `{"a":1}`,
		`{"a":1}`:                         `{"a":1}`,
		"前缀 {\"a\":1} 后缀":              `{"a":1}`,
		"模型说：结果如下\n{\"a\":1}\n结束": `{"a":1}`,
	}
	for in, want := range cases {
		got := extractJSONObject(stripJSONFence(in))
		if got != want {
			t.Errorf("for %q: got %q want %q", in, got, want)
		}
	}
}

func TestFlattenWithNestedSections(t *testing.T) {
	list := SectionList{
		{Title: "卷一", Sections: []Section{
			{Title: "第1章", Content: "<p>a</p>"},
			{Title: "第2章", Content: "<p>b</p>"},
		}},
		{Title: "第3章", Content: "<p>c</p>"},
	}
	titles, locs := list.Flatten()
	wantTitles := []string{"第1章", "第2章", "第3章"}
	if len(titles) != 3 {
		t.Fatalf("expected 3 titles, got %d", len(titles))
	}
	for i, w := range wantTitles {
		if titles[i] != w {
			t.Errorf("title[%d]=%q want %q", i, titles[i], w)
		}
	}
	if locs[0].Parent != 0 || locs[0].Index != 0 {
		t.Errorf("locs[0] wrong: %+v", locs[0])
	}
	if locs[2].Parent != -1 || locs[2].Index != 1 {
		t.Errorf("locs[2] wrong: %+v", locs[2])
	}
}

func TestParseLang(t *testing.T) {
	// 占位：确保 client.Ready 的边界
	c := NewClient(ClientConfig{APIKey: "k"})
	if c.Ready() {
		t.Fatal("Ready should be false without model")
	}
	c2 := NewClient(ClientConfig{Model: "m"})
	if c2.Ready() {
		t.Fatal("Ready should be false without api_key")
	}
}

func TestSanitizeStructureDropsInvalid(t *testing.T) {
	plan := StructurePlan{
		Rename:  map[int]string{0: "valid", 5: "out"},
		Merge:   [][]int{{0, 1}, {99, 100}, {2}},
		Remove:  []int{3, 99},
	}
	got := sanitizeStructure(plan, 4)
	if _, ok := got.Rename[5]; ok {
		t.Fatal("out-of-range rename should be dropped")
	}
	if len(got.Merge) != 1 {
		t.Fatalf("only valid merge group should remain, got %d", len(got.Merge))
	}
	if len(got.Remove) != 1 || got.Remove[0] != 3 {
		t.Fatalf("only valid remove should remain, got %v", got.Remove)
	}
}

func TestStructureMergeContextualTruncatedChapter(t *testing.T) {
	// 模拟 AI 识别到纯数字被错误切出为章节，根据上下文指示将 1 合并到 0
	resp := `{"merge":[[0,1]]}`
	c := newMockClient(t, 200, fixedContent(resp))
	list := SectionList{
		{Title: "第1章 觉醒", Content: "<p class=\"content\">榜单前十名：01. 某某</p>"},
		{Title: "53", Content: "<p class=\"content\">02. 另外一人 53</p>"},
		{Title: "第2章 启程", Content: "<p class=\"content\">天高气爽。</p>"},
	}
	out, _ := Refine(list, RefineOptions{Enabled: true, Client: c, DoStructure: true})
	if len(out) != 2 {
		t.Fatalf("expected 2 sections after merging truncated chapter, got %d", len(out))
	}
	if out[0].Title != "第1章 觉醒" {
		t.Fatalf("unexpected title for section 0: %q", out[0].Title)
	}
	if !strings.Contains(out[0].Content, "榜单前十名") || !strings.Contains(out[0].Content, "02. 另外一人 53") {
		t.Fatalf("expected content of 53 to be merged into chapter 1, got %q", out[0].Content)
	}
}

func TestSanitizeNoiseLengthBounds(t *testing.T) {
	plan := NoisePlan{Substrings: []string{"ab", strings.Repeat("长", 80), "正常特征"}}
	got := sanitizeNoise(plan)
	if len(got.Substrings) != 1 || got.Substrings[0] != "正常特征" {
		t.Fatalf("only mid-length substring should remain, got %v", got.Substrings)
	}
}

func TestSanitizeTypographyMaxLength(t *testing.T) {
	long := strings.Repeat("字", 50)
	plan := TypographyPlan{Replacements: []ReplacementRule{
		{From: long, To: "x"},
		{From: "", To: "y"},
		{From: "abc", To: "abc"},
		{From: "ok", To: "好"},
	}}
	got := sanitizeTypography(plan)
	if len(got.Replacements) != 1 || got.Replacements[0].From != "ok" {
		t.Fatalf("only valid rule should remain, got %+v", got.Replacements)
	}
}

func containsAny(slice []string, sub string) bool {
	for _, s := range slice {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func ExampleSectionList_Flatten() {
	list := SectionList{
		{Title: "卷一", Sections: []Section{{Title: "第1章"}}},
		{Title: "第2章"},
	}
	titles, _ := list.Flatten()
	fmt.Println(titles)
	// Output: [第1章 第2章]
}

func TestMaskKeyInString(t *testing.T) {
	cases := map[string]string{
		"error: sk-abcdef1234567890 invalid":    "error: sk-" + strings.Repeat("*", 8) + "90 invalid",
		"Bearer sk_test_abcdef1234567890":       "Bearer sk_" + strings.Repeat("*", 8) + "90",
		"normal message":                        "normal message",
		"":                                      "",
	}
	for in, want := range cases {
		got := maskKeyInString(in)
		if got != want {
			t.Errorf("maskKeyInString(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClientMaskKeyInString(t *testing.T) {
	c := NewClient(ClientConfig{APIKey: "sk-verysecretkey1234567890", Model: "m"})
	out := c.maskKeyInString("error: got sk-verysecretkey1234567890 from server")
	if strings.Contains(out, "sk-verysecretkey1234567890") {
		t.Fatalf("client key leaked in masked output: %q", out)
	}
}

func TestExtractAssistantContentReasoningFallback(t *testing.T) {
	got := extractAssistantContent(chatMessage{
		Content:          "",
		ReasoningContent: `{"rename":{"0":"第1章"}}`,
	})
	if got != `{"rename":{"0":"第1章"}}` {
		t.Fatalf("expected reasoning_content fallback, got %q", got)
	}
}

func TestExtractAssistantContentPrefersJSONField(t *testing.T) {
	got := extractAssistantContent(chatMessage{
		Content:          "章节结构正常，无需调整。",
		ReasoningContent: `{"rename":{},"merge":[],"remove":[]}`,
	})
	if !strings.Contains(got, "{") {
		t.Fatalf("expected JSON from reasoning_content, got %q", got)
	}
}

func TestParseStructureNoChangeText(t *testing.T) {
	plan, err := parseStructureResponse("经检查，章节结构正常，无需调整。")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.Rename) != 0 || len(plan.Merge) != 0 || len(plan.Remove) != 0 {
		t.Fatalf("expected empty plan, got %+v", plan)
	}
}

func TestStructurePromptUsesGlobalOffset(t *testing.T) {
	prompt := buildStructurePrompt([]structureEntry{{Title: "第1章"}, {Title: "第2章"}}, 350, "")
	if !strings.Contains(prompt, "[350]") || !strings.Contains(prompt, "[351]") {
		t.Fatalf("expected global indices in prompt, got %q", prompt)
	}
}

func TestChatRetriesOnEmptyJSONContent(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		content := ""
		if calls >= 2 {
			content = `{"rename":{}}`
		}
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": content}, "finish_reason": "stop"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	c := NewClient(ClientConfig{
		BaseURL: srv.URL,
		APIKey:  "test-key",
		Model:   "test-model",
		Timeout: 5 * time.Second,
	})
	out, err := c.Chat(context.Background(), "system", "user", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != `{"rename":{}}` {
		t.Fatalf("unexpected content: %q", out)
	}
	if calls < 2 {
		t.Fatalf("expected retry on empty content, calls=%d", calls)
	}
}
