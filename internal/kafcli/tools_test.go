package kafcli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestResolveOutputPath(t *testing.T) {
	txt := `H:\books\《第一玩家》作者：流泪猫安头.txt`
	got := resolveOutputPath(txt, "《第一玩家》作者：流泪猫安头")
	want := `H:\books\《第一玩家》作者：流泪猫安头`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	absOut := resolveOutputPath(txt, `D:\output\custom`)
	if absOut != `D:\output\custom` {
		t.Fatalf("abs out got %q", absOut)
	}
}

func TestIsPartDivisionLabel(t *testing.T) {
	for _, line := range []string{"第一部", "第三部", "第12部", "第零部"} {
		if !isPartDivisionLabel(line) {
			t.Fatalf("%q should be part label", line)
		}
	}
	for _, line := range []string{"第1章 开端", "第一卷", "第三部 崛起", "某个部门", "第1部开始"} {
		if isPartDivisionLabel(line) {
			t.Fatalf("%q should not be part label", line)
		}
	}
}

func TestFilenameMeta(t *testing.T) {
	name, author := FilenameMeta(`book/《第一玩家》作者：流泪猫安头.txt`)
	if name != "第一玩家" || author != "流泪猫安头" {
		t.Fatalf("got name=%q author=%q", name, author)
	}
	name2, _ := FilenameMeta(`D:/全职法师.txt`)
	if name2 != "全职法师" {
		t.Fatalf("got %q", name2)
	}
}

func TestNormalizeLineQuotes(t *testing.T) {
	in := "「苏先生」女子说：「您好。」【系统】「误触」"
	want := "“苏先生”女子说：“您好。”【系统】“误触”"
	if got := normalizeLineQuotes(in); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := normalizeLineQuotes("无引号一行"); got != "无引号一行" {
		t.Fatalf("unchanged line modified: %q", got)
	}
}

func TestDedupTitleSections(t *testing.T) {
	sections := []Section{
		{Title: "第1章 一章「abc」", Content: ""},
		{Title: "第1章 一章·「abc」", Content: "<p>body</p>"},
		{Title: "第2章 二章", Content: ""},
		{Title: "第2章 二章·", Content: "<p>body2</p>"},
		{Title: "第3章 三章", Content: "<p>has content</p>"},
	}
	got := dedupTitleSections(sections)
	if len(got) != 3 {
		t.Fatalf("want 3 sections, got %d", len(got))
	}
	if got[0].Title != "第1章 一章·「abc」" {
		t.Fatalf("unexpected first title: %s", got[0].Title)
	}
	if got[2].Title != "第3章 三章" {
		t.Fatalf("unexpected third title: %s", got[2].Title)
	}
}

func TestDedupTitleSectionsSupportsChineseOrdinalDotTitles(t *testing.T) {
	sections := []Section{
		{Title: "一百二十章·第二天·白天", Content: ""},
		{Title: "第122章 一百二十章·第二天·白天", Content: "<p>body</p>"},
		{Title: "一百二十一章·夜间", Content: "<p>body2</p>"},
	}
	got := dedupTitleSections(sections)
	if len(got) != 2 {
		t.Fatalf("want 2 sections, got %d", len(got))
	}
	if got[0].Title != "第122章 一百二十章·第二天·白天" {
		t.Fatalf("unexpected first title: %q", got[0].Title)
	}
}

func TestDedupRepeatedSections(t *testing.T) {
	sections := []Section{
		{Title: "第29章 二十八章「第一玩家？」", Content: "<p class=\"content\">浩瀚的星海下，是巨大的黑白棋盘。</p><p class=\"content\">弹幕疯狂地滚动起来。</p>"},
		{Title: "第30章 下一章", Content: "<p class=\"content\">新的正文。</p>"},
		{Title: "第29章 二十八章“第一玩家？”", Content: "<p class=\"content\">第29章 二十八章·“第一玩家？”浩瀚的星海下，是巨大的黑白棋盘。</p><p class=\"content\">弹幕疯狂地滚动起来。</p>"},
	}
	got := dedupRepeatedSections(sections)
	if len(got) != 2 {
		t.Fatalf("want 2 sections after repeated-content dedup, got %d", len(got))
	}
	if got[0].Title != "第29章 二十八章「第一玩家？」" {
		t.Fatalf("unexpected first section kept: %q", got[0].Title)
	}
	if got[1].Title != "第30章 下一章" {
		t.Fatalf("unexpected second section kept: %q", got[1].Title)
	}
}

func TestDedupRepeatedSectionsKeepsDifferentContent(t *testing.T) {
	sections := []Section{
		{Title: "第1章 开始", Content: "<p class=\"content\">正文一。</p>"},
		{Title: "第1章 开始·", Content: "<p class=\"content\">正文二。</p>"},
	}
	got := dedupRepeatedSections(sections)
	if len(got) != 2 {
		t.Fatalf("different content should be kept, got %d sections", len(got))
	}
}

func TestDedupRepeatedSectionsActualInlineRepeatPattern(t *testing.T) {
	sections := []Section{
		{
			Title:   "第29章 二十八章·「第一玩家？」",
			Content: "<p class=\"content\">浩瀚的星海下，是巨大的黑白棋盘。</p><p class=\"content\">几乎聚集了全世界目光的艾尼，立于黑白棋盘之上，站立在蓝方的最后方。</p><p class=\"content\">弹幕疯狂地滚动起来：</p>",
		},
		{
			Title:   "第29章 二十八章“第一玩家？”",
			Content: "<p class=\"content\">第29章 二十八章·“第一玩家？”浩瀚的星海下，是巨大的黑白棋盘。</p><p class=\"content\">几乎聚集了全世界目光的艾尼，立于黑白棋盘之上，站立在蓝方的最后方。</p><p class=\"content\">弹幕疯狂地滚动起来：</p>",
		},
	}

	got := dedupRepeatedSections(sections)
	if len(got) != 1 {
		t.Fatalf("actual inline repeat should dedup to 1, got %d", len(got))
	}
}

func TestDedupRepeatedSectionsAllowsMinorTextDrift(t *testing.T) {
	sections := []Section{
		{
			Title:   "第67章 六十六章·「她是完美的诺丽雅。」",
			Content: "<p class=\"content\">一阵白光过后，苏明安回归个人小空间中。</p><p class=\"content\">他直视眼前窄小的房间，听到一阵系统提示声。</p><p class=\"content\">【世界回顾：获得「反抗军首领」身份，在别墅与潜伏者「关娜」碰头，获得关键晶片。</p>",
		},
		{
			Title:   "第67章 六十六章“她是完美的诺丽雅。”",
			Content: "<p class=\"content\">第67章 六十六章·“她是完美的诺丽雅。”一阵白光过后，苏明安回归个人小空间中。</p><p class=\"content\">他直视眼前窄小的房间，听到一阵系统提示声。</p><p class=\"content\">【世界回顾：获得“反抗军首领”身份，在别墅与潜伏者“关娜”碰头，获得关键芯片。</p>",
		},
	}

	got := dedupRepeatedSections(sections)
	if len(got) != 1 {
		t.Fatalf("minor text drift repeat should dedup to 1, got %d", len(got))
	}
}

func TestDedupRepeatedSectionsSupportsChineseOrdinalDotTitles(t *testing.T) {
	sections := []Section{
		{
			Title:   "一百二十章·第二天·白天",
			Content: "<p class=\"content\">正文第一段。</p><p class=\"content\">正文第二段。</p>",
		},
		{
			Title:   "第122章 一百二十章·第二天·白天",
			Content: "<p class=\"content\">第122章 一百二十章·第二天·白天正文第一段。</p><p class=\"content\">正文第二段。</p>",
		},
		{
			Title:   "一百二十一章·夜间",
			Content: "<p class=\"content\">下一章正文。</p>",
		},
	}

	got := dedupRepeatedSections(sections)
	if len(got) != 2 {
		t.Fatalf("new-source repeated-content dedup should keep 2 sections, got %d", len(got))
	}
	if got[0].Title != "一百二十章·第二天·白天" {
		t.Fatalf("unexpected first section kept: %q", got[0].Title)
	}
}

func TestTrimLeadingTitlePrefix(t *testing.T) {
	title := "第331章 三百二十九章“这游戏给你玩明白了”"
	line := "第331章 三百二十九章·「这游戏给你玩明白了」【金蔷薇的话术说服了红方军师，红方军师决定采用围剿计划，战局维持平衡。】"
	got, ok := trimLeadingTitlePrefix(title, line)
	if !ok {
		t.Fatal("expected title prefix to be trimmed")
	}
	want := "【金蔷薇的话术说服了红方军师，红方军师决定采用围剿计划，战局维持平衡。】"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestParseLang(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty defaults to en", in: "", want: "en"},
		{name: "supported zh", in: "zh", want: "zh"},
		{name: "unsupported token", in: "cn", want: "en"},
		{name: "substring no longer accepted", in: "h", want: "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseLang(tt.in); got != tt.want {
				t.Fatalf("parseLang(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestCheckRequiresFilename(t *testing.T) {
	book := &Book{}
	book.SetDefault()

	err := book.Check("test-version")
	if err == nil {
		t.Fatal("expected error for empty filename")
	}
	msg := err.Error()
	if !strings.Contains(msg, "文件名不能为空") {
		t.Fatalf("unexpected error message: %q", msg)
	}
	if !strings.Contains(msg, "test-version") {
		t.Fatalf("expected version in error message: %q", msg)
	}
}

func TestCheckMissingCoverFallsBackToNoCover(t *testing.T) {
	txtPath := filepath.Join(t.TempDir(), "book.txt")
	if err := os.WriteFile(txtPath, []byte("第1章 开头\n正文\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	book := &Book{
		Filename: txtPath,
		Cover:    filepath.Join(t.TempDir(), "missing-cover.png"),
	}
	book.SetDefault()

	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if book.Cover != "" {
		t.Fatalf("expected missing cover to be cleared, got %q", book.Cover)
	}
}

func TestCheckDefaultCoverFallsBackToJPG(t *testing.T) {
	dir := t.TempDir()
	txtPath := filepath.Join(dir, "book.txt")
	if err := os.WriteFile(txtPath, []byte("第1章 开头\n正文\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	jpgPath := filepath.Join(dir, "cover.jpg")
	if err := os.WriteFile(jpgPath, []byte("fake jpg"), 0o600); err != nil {
		t.Fatal(err)
	}

	book := &Book{
		Filename: txtPath,
	}
	book.SetDefault()

	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if book.Cover != jpgPath {
		t.Fatalf("expected jpg cover fallback %q, got %q", jpgPath, book.Cover)
	}
}

func TestCheckDefaultCoverFallsBackToJPEG(t *testing.T) {
	dir := t.TempDir()
	txtPath := filepath.Join(dir, "book.txt")
	if err := os.WriteFile(txtPath, []byte("第1章 开头\n正文\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	jpegPath := filepath.Join(dir, "cover.jpeg")
	if err := os.WriteFile(jpegPath, []byte("fake jpeg"), 0o600); err != nil {
		t.Fatal(err)
	}

	book := &Book{
		Filename: txtPath,
	}
	book.SetDefault()

	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if book.Cover != jpegPath {
		t.Fatalf("expected jpeg cover fallback %q, got %q", jpegPath, book.Cover)
	}
}

func TestParseMissingFileReturnsError(t *testing.T) {
	book := &Book{
		Filename: filepath.Join(t.TempDir(), "missing.txt"),
	}
	book.SetDefault()

	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
	if err := book.Parse(); err == nil {
		t.Fatal("expected parse to fail for missing file")
	}
}

func TestParseHonorsDedupAndNormalizeQuotes(t *testing.T) {
	dir := t.TempDir()
	txtPath := filepath.Join(dir, "novel.txt")
	content := strings.Join([]string{
		"第1章 一章「abc」",
		"第1章 一章·「abc」",
		"「苏先生」女子说：「您好。」",
		"第一部",
		"第2章 二章",
		"正文第二章",
	}, "\n")
	if err := os.WriteFile(txtPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	book := &Book{
		Filename:        txtPath,
		DedupTitle:      true,
		NormalizeQuotes: true,
		Tips:            false,
	}
	book.SetDefault()
	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
	if err := book.Parse(); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(book.SectionList) != 2 {
		t.Fatalf("expected 2 sections after dedup, got %d", len(book.SectionList))
	}
	if book.SectionList[0].Title != "第1章 一章·「abc」" {
		t.Fatalf("unexpected first section title: %q", book.SectionList[0].Title)
	}
	if !strings.Contains(book.SectionList[0].Content, "“苏先生”女子说：“您好。”") {
		t.Fatalf("quotes were not normalized: %q", book.SectionList[0].Content)
	}
	if !strings.Contains(book.SectionList[0].Content, "<p class=\"content\">第一部</p>") {
		t.Fatalf("part division label should stay in content: %q", book.SectionList[0].Content)
	}
}

func TestParseGroupsVolumes(t *testing.T) {
	dir := t.TempDir()
	txtPath := filepath.Join(dir, "novel.txt")
	content := strings.Join([]string{
		"第一卷 初始",
		"第1章 开始",
		"正文一",
		"第2章 继续",
		"正文二",
	}, "\n")
	if err := os.WriteFile(txtPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	book := &Book{
		Filename: txtPath,
		Tips:     false,
	}
	book.SetDefault()
	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
	if err := book.Parse(); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(book.SectionList) != 1 {
		t.Fatalf("expected one top-level volume, got %d", len(book.SectionList))
	}
	volume := book.SectionList[0]
	if !regexp.MustCompile(`^第一卷`).MatchString(volume.Title) {
		t.Fatalf("unexpected volume title: %q", volume.Title)
	}
	if len(volume.Sections) != 2 {
		t.Fatalf("expected 2 sections in volume, got %d", len(volume.Sections))
	}
}

func TestParseKeepsRankedListAsContent(t *testing.T) {
	dir := t.TempDir()
	txtPath := filepath.Join(dir, "novel.txt")
	content := strings.Join([]string{
		"第70章 六十九章「他真的会那么狭隘吗？」",
		"【世界排行榜（综合评定）：",
		"1、苏明安（战力：550）（第一玩家）（完美通关*2）/职业：白审",
		"2、诺尔（战力：650）（完美通关*1）/职业：傀儡师",
		"……】",
		"苏明安看着排行榜，刚走到楼下。",
		"第71章 下一章",
		"正文第二章",
	}, "\n")
	if err := os.WriteFile(txtPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	book := &Book{
		Filename: txtPath,
		Tips:     false,
	}
	book.SetDefault()
	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
	if err := book.Parse(); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(book.SectionList) != 2 {
		t.Fatalf("expected 2 real chapters, got %d", len(book.SectionList))
	}
	if !strings.Contains(book.SectionList[0].Content, "1、苏明安（战力：550）") {
		t.Fatalf("ranked list line should stay in content: %q", book.SectionList[0].Content)
	}
	if book.SectionList[1].Title != "第71章 下一章" {
		t.Fatalf("unexpected second chapter title: %q", book.SectionList[1].Title)
	}
}

func TestParseKeepsRoundLabelsAsContent(t *testing.T) {
	dir := t.TempDir()
	txtPath := filepath.Join(dir, "novel.txt")
	content := strings.Join([]string{
		"第1章 开始",
		"正文第一段",
		"第二回合，苏明安开始拉高信仰值。",
		"【第二回合结束。】",
		"第三回合，他感觉应该是七百年前左右的时间段。",
		"【双方行棋，第三回合。】",
		"第2章 继续",
		"正文第二章",
	}, "\n")
	if err := os.WriteFile(txtPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	book := &Book{
		Filename: txtPath,
		Tips:     false,
	}
	book.SetDefault()
	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
	if err := book.Parse(); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(book.SectionList) != 2 {
		t.Fatalf("expected 2 real chapters, got %d", len(book.SectionList))
	}
	if !strings.Contains(book.SectionList[0].Content, "第二回合，苏明安开始拉高信仰值。") {
		t.Fatalf("round label line should stay in content: %q", book.SectionList[0].Content)
	}
	if !strings.Contains(book.SectionList[0].Content, "【双方行棋，第三回合。】") {
		t.Fatalf("chess round label should stay in content: %q", book.SectionList[0].Content)
	}
}

func TestParseSkipsInlineChapterLineAsBookmark(t *testing.T) {
	dir := t.TempDir()
	txtPath := filepath.Join(dir, "novel.txt")
	content := strings.Join([]string{
		"第403章 四百零一章“没有战争的世界。”",
		"正文第一章",
		"第29章 二十八章·“第一玩家？”浩瀚的星海下，是巨大的黑白棋盘。",
		"几乎聚集了全世界目光的艾尼，立于黑白棋盘之上。",
		"第404章 下一章",
		"正文第二章",
	}, "\n")
	if err := os.WriteFile(txtPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	book := &Book{
		Filename: txtPath,
		Tips:     false,
	}
	book.SetDefault()
	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
	if err := book.Parse(); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(book.SectionList) != 2 {
		t.Fatalf("expected 2 real chapters, got %d", len(book.SectionList))
	}
	if strings.Contains(book.SectionList[0].Title, "第一玩家") {
		t.Fatalf("inline chapter body line should not become its own bookmark: %q", book.SectionList[0].Title)
	}
	if !strings.Contains(book.SectionList[0].Content, "第29章 二十八章·“第一玩家？”浩瀚的星海下") {
		t.Fatalf("inline chapter line should stay in content: %q", book.SectionList[0].Content)
	}
}

func TestParseSupportsChineseOrdinalChapterDotTitles(t *testing.T) {
	dir := t.TempDir()
	txtPath := filepath.Join(dir, "novel.txt")
	content := strings.Join([]string{
		"一章·“你拥有光辉明亮的未来。”",
		"正文第一章",
		"二章·新手副本TE·消失的她",
		"正文第二章",
		"三十九章·最终推论环节",
		"正文第三章",
	}, "\n")
	if err := os.WriteFile(txtPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	book := &Book{
		Filename: txtPath,
		Tips:     false,
	}
	book.SetDefault()
	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
	if err := book.Parse(); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(book.SectionList) != 3 {
		t.Fatalf("expected 3 chapters, got %d", len(book.SectionList))
	}
	if book.SectionList[0].Title != "一章·“你拥有光辉明亮的未来。”" {
		t.Fatalf("unexpected first title: %q", book.SectionList[0].Title)
	}
	if book.SectionList[2].Title != "三十九章·最终推论环节" {
		t.Fatalf("unexpected third title: %q", book.SectionList[2].Title)
	}
}

func TestLooksLikeInlineChapterLineKeepsNormalQuotedTitles(t *testing.T) {
	for _, line := range []string{
		"第29章 二十八章「第一玩家？」",
		"第403章 四百零一章“没有战争的世界。”",
		"第67章 六十六章“她是完美的诺丽雅。”",
		"第五十九章：那都是演技？对，演技",
		"第六十九章：大荒囚天指！（5700字",
		"第七十二章：北海魔女永不团灭！（5000字）",
		"第一百三十八章：以多欺少？坚毅不倒！（五千求月票",
	} {
		if looksLikeInlineChapterLine(line) {
			t.Fatalf("normal quoted title should not be treated as inline body: %q", line)
		}
	}
	if !looksLikeInlineChapterLine("第29章 二十八章·“第一玩家？”浩瀚的星海下，是巨大的黑白棋盘。") {
		t.Fatal("title glued to body text should be treated as inline body")
	}
}

func TestIsFalsePositiveOrdinalTitle(t *testing.T) {
	for _, line := range []string{
		"第一节课是数学课。",
		"第一章·白莲灭城",
		"第二章0%",
		"第一章87%",
	} {
		if !isFalsePositiveOrdinalTitle(line) {
			t.Fatalf("expected false positive ordinal title: %q", line)
		}
	}
	for _, line := range []string{
		"第29章 二十八章「第一玩家？」",
		"第403章 四百零一章“没有战争的世界。”",
		"第1313章 一千三百零八章【叙事错误（下）】",
		"第一章：我等这一刻，望眼欲穿",
		"第一章: 望眼欲穿",
		"第一章、望眼欲穿",
		"第一章 望眼欲穿",
		"第一章-望眼欲穿",
		"第一章——望眼欲穿",
		"第013章",
		"第二百四十二章 代天巡狩道 (五千四)",
	} {
		if isFalsePositiveOrdinalTitle(line) {
			t.Fatalf("real chapter should not be blocked: %q", line)
		}
	}
}

func TestParseExampleBookDedupsRepeatedChapters(t *testing.T) {
	txtPath := filepath.Join("..", "..", "examples", "《第一玩家》作者：流泪猫安头.txt")
	if _, err := os.Stat(txtPath); err != nil {
		t.Skipf("example book not available: %v", err)
	}

	book := &Book{
		Filename:   txtPath,
		DedupTitle: true,
		Tips:       false,
	}
	book.SetDefault()
	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
	if err := book.Parse(); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	legacyTitles := []string{
		"第29章 二十八章",
		"第67章 六十六章",
		"第331章 三百二十九章",
		"第335章 三百三十三章",
		"第337章 三百三十五章",
		"第344章 三百四十二章",
		"第403章 四百零一章",
	}
	newSourceTitles := []string{
		"一章·“你拥有光辉明亮的未来。”",
		"二章·新手副本TE·消失的她",
		"三十九章·最终推论环节",
	}

	if count, _ := countSectionTitlesContaining(book.SectionList, "第29章 二十八章"); count > 0 {
		for _, titlePart := range legacyTitles {
			count, matches := countSectionTitlesContaining(book.SectionList, titlePart)
			if count != 1 {
				t.Fatalf("expected %q to appear once after dedup, got %d, matches=%v details=%v", titlePart, count, matches, collectSectionDebugContaining(book.SectionList, titlePart))
			}
		}
	} else {
		for _, titlePart := range newSourceTitles {
			count, matches := countSectionTitlesContaining(book.SectionList, titlePart)
			if count == 0 {
				t.Fatalf("expected new-source title %q to be recognized, matches=%v", titlePart, matches)
			}
		}
	}

	for _, bad := range []string{"第二回合", "第三回合", "1、苏明安"} {
		count, matches := countSectionTitlesContaining(book.SectionList, bad)
		if count != 0 {
			t.Fatalf("expected %q to stay out of bookmarks, got %d matches=%v", bad, count, matches)
		}
	}

	for _, bad := range []string{"第一节课是数学课", "开启下一章", "第一章87%", "第一章86%", "只有第一章"} {
		count, matches := countSectionTitlesContaining(book.SectionList, bad)
		if count != 0 {
			t.Fatalf("expected %q to stay out of bookmarks, got %d matches=%v", bad, count, matches)
		}
	}
}

func TestParseExampleBookSkipsOrdinalNarrationBookmarks(t *testing.T) {
	txtPath := filepath.Join("..", "..", "examples", "《第一玩家》作者：流泪猫安头.txt")
	if _, err := os.Stat(txtPath); err != nil {
		t.Skipf("example book not available: %v", err)
	}

	book := &Book{
		Filename:   txtPath,
		DedupTitle: true,
		Tips:       false,
	}
	book.SetDefault()
	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
	if err := book.Parse(); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	for _, bad := range []string{
		"第一节课是数学课",
		"开启下一章",
		"第一章87%",
		"第一章86%",
		"只有第一章",
		"章节怎么断在这里",
	} {
		count, matches := countSectionTitlesContaining(book.SectionList, bad)
		if count != 0 {
			t.Fatalf("expected %q to stay out of bookmarks, got %d matches=%v", bad, count, matches)
		}
	}
}

func TestParseExampleBookSupportsChineseOrdinalChapterDotTitles(t *testing.T) {
	txtPath := filepath.Join("..", "..", "examples", "《第一玩家》作者：流泪猫安头.txt")
	if _, err := os.Stat(txtPath); err != nil {
		t.Skipf("example book not available: %v", err)
	}

	book := &Book{
		Filename:   txtPath,
		DedupTitle: true,
		Tips:       false,
	}
	book.SetDefault()
	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
	if err := book.Parse(); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	for _, title := range []string{
		"一章·“你拥有光辉明亮的未来。”",
		"二章·新手副本TE·消失的她",
		"三十九章·最终推论环节",
		"一百二十章·",
	} {
		count, matches := countSectionTitlesContaining(book.SectionList, title)
		if count == 0 {
			t.Fatalf("expected chapter title %q to be recognized, matches=%v", title, matches)
		}
	}
}

func TestParseExampleBookDedupsChineseOrdinalMirrorChapters(t *testing.T) {
	txtPath := filepath.Join("..", "..", "examples", "《第一玩家》作者：流泪猫安头.txt")
	if _, err := os.Stat(txtPath); err != nil {
		t.Skipf("example book not available: %v", err)
	}

	book := &Book{
		Filename:   txtPath,
		DedupTitle: true,
		Tips:       false,
	}
	book.SetDefault()
	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
	if err := book.Parse(); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	for _, title := range []string{
		"一百二十章·第二天·白天",
		"一百二十四章·“我顺光而行”",
		"一百二十五章·“我老了，也快要疯了。”",
		"一百二十六章·“我亲爱的孩子，泊里。”",
	} {
		count, matches := countSectionTitlesContaining(book.SectionList, title)
		if count != 1 {
			t.Fatalf("expected chapter title %q to appear once after dedup, got %d matches=%v details=%v", title, count, matches, collectSectionDebugContaining(book.SectionList, title))
		}
	}

	count, matches := countSectionTitlesContaining(book.SectionList, "一千一百二十章")
	if count != 1 {
		t.Fatalf("expected mirrored chapter around 一千一百二十章 to collapse to one bookmark, got %d matches=%v details=%v", count, matches, collectSectionDebugContaining(book.SectionList, "一千一百二十章"))
	}
}

func countSectionTitlesWithPrefix(sections []Section, prefix string) (int, []string) {
	count := 0
	var matches []string
	for _, sec := range sections {
		if strings.HasPrefix(strings.TrimSpace(sec.Title), prefix) {
			count++
			matches = append(matches, sec.Title)
		}
		for _, child := range sec.Sections {
			if strings.HasPrefix(strings.TrimSpace(child.Title), prefix) {
				count++
				matches = append(matches, child.Title)
			}
		}
	}
	return count, matches
}

func countSectionTitlesContaining(sections []Section, sub string) (int, []string) {
	count := 0
	var matches []string
	for _, sec := range sections {
		if strings.Contains(sec.Title, sub) {
			count++
			matches = append(matches, sec.Title)
		}
		for _, child := range sec.Sections {
			if strings.Contains(child.Title, sub) {
				count++
				matches = append(matches, child.Title)
			}
		}
	}
	return count, matches
}

func collectSectionDebug(sections []Section, prefix string) []string {
	var details []string
	appendDetail := func(sec Section) {
		if !strings.HasPrefix(strings.TrimSpace(sec.Title), prefix) {
			return
		}
		contentKey := normalizeChapterContent(sec.Content)
		contentKey = stripLeadingTitleLine(sec.Title, contentKey)
		snippet := contentKey
		rs := []rune(snippet)
		if len(rs) > 40 {
			snippet = string(rs[:40])
		}
		details = append(details, sec.Title+" => "+snippet)
	}
	for _, sec := range sections {
		appendDetail(sec)
		for _, child := range sec.Sections {
			appendDetail(child)
		}
	}
	return details
}

func collectSectionDebugContaining(sections []Section, sub string) []string {
	var details []string
	appendDetail := func(sec Section) {
		if !strings.Contains(sec.Title, sub) {
			return
		}
		contentKey := normalizeChapterContent(sec.Content)
		contentKey = stripLeadingTitleLine(sec.Title, contentKey)
		snippet := contentKey
		rs := []rune(snippet)
		if len(rs) > 40 {
			snippet = string(rs[:40])
		}
		details = append(details, sec.Title+" => "+snippet)
	}
	for _, sec := range sections {
		appendDetail(sec)
		for _, child := range sec.Sections {
			appendDetail(child)
		}
	}
	return details
}

func TestCleanChapterTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "第二十七章：二阶段，血回满（5500字）",
			want:  "第二十七章：二阶段，血回满",
		},
		{
			input: "第三十一章：我不明白（三江pk 求追读求票）",
			want:  "第三十一章：我不明白",
		},
		{
			input: "第三十八章：最后的约会（5000",
			want:  "第三十八章：最后的约会",
		},
		{
			input: "第一百九十四章 归来的司魔屠（八千",
			want:  "第一百九十四章 归来的司魔屠",
		},
		{
			input: "第一百九十五章 大被同眠（六千七）",
			want:  "第一百九十五章 大被同眠",
		},
		{
			input: "第一百九十六章 天劫再临（六千求月票）",
			want:  "第一百九十六章 天劫再临",
		},
		{
			input: "第两百零一章 大成圣体！(五千七求月票 第二更",
			want:  "第两百零一章 大成圣体！",
		},
		{
			input: "第二百四十二章 代天巡狩道 (五千四)",
			want:  "第二百四十二章 代天巡狩道",
		},
		{
			input: "第二百一十四章 求金 (求月票)",
			want:  "第二百一十四章 求金",
		},
		{
			input: "第1313章 一千三百零八章【叙事错误（下）】",
			want:  "第1313章 一千三百零八章【叙事错误（下）】",
		},
		{
			input: "第一百章 决战（上）",
			want:  "第一百章 决战（上）",
		},
	}

	for _, tt := range tests {
		got := cleanChapterTitle(tt.input)
		if got != tt.want {
			t.Errorf("cleanChapterTitle(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsDividerLine(t *testing.T) {
	if !isDividerLine("------------") {
		t.Error("expected ------------ to be recognized as divider")
	}
	if !isDividerLine("------") {
		t.Error("expected ------ to be recognized as divider")
	}
	if !isDividerLine("***") {
		t.Error("expected *** to be recognized as divider")
	}
	if isDividerLine("--") {
		t.Error("expected -- not to be divider (< 3 chars)")
	}
	if isDividerLine("普通正文内容") {
		t.Error("expected normal text not to be divider")
	}
}

func TestParseMagicalGirlExampleBook(t *testing.T) {
	txtPath := filepath.Join("..", "..", "examples", "book", "《从魔法少女开始独断万古》作者：绿茶藨子.txt")
	if _, err := os.Stat(txtPath); err != nil {
		t.Skipf("example book not available: %v", err)
	}

	book := &Book{
		Filename:        txtPath,
		DedupTitle:      true,
		NormalizeQuotes: true,
		Tips:            false,
	}
	book.SetDefault()
	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
	if book.Bookname != "从魔法少女开始独断万古" {
		t.Fatalf("unexpected bookname: %q", book.Bookname)
	}
	if book.Author != "绿茶藨子" {
		t.Fatalf("unexpected author: %q", book.Author)
	}
	if err := book.Parse(); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(book.SectionList) < 370 {
		t.Fatalf("expected at least 370 chapters, got %d", len(book.SectionList))
	}

	// 验证第一章被正常识别且未被误报拦截
	if !strings.HasPrefix(book.SectionList[0].Title, "第一章：我等这一刻") {
		t.Fatalf("unexpected first chapter title: %q", book.SectionList[0].Title)
	}

	// 验证标题作话被清理
	for _, sec := range book.SectionList {
		if strings.Contains(sec.Title, "5500字") || strings.Contains(sec.Title, "求月票") {
			t.Errorf("found uncleaned noise in title: %q", sec.Title)
		}
	}
}

func TestMergeIsolatedDigitSections(t *testing.T) {
	sections := []Section{
		{Title: "第一章 觉醒", Content: "<p>第一段</p>"},
		{Title: "53", Content: "<p>榜单02: 灵梦</p>"},
		{Title: "第二章 进发", Content: "<p>第二段</p>"},
		{Title: "第三章 决战", Content: "<p>第三段</p>"},
		{Title: "第四章 终曲", Content: "<p>第四段</p>"},
	}
	merged := mergeIsolatedDigitSections(sections)
	if len(merged) != 4 {
		t.Fatalf("expected 4 sections after merging digit section, got %d", len(merged))
	}
	if merged[0].Title != "第一章 觉醒" {
		t.Errorf("unexpected first section title: %q", merged[0].Title)
	}
	if !strings.Contains(merged[0].Content, "53") || !strings.Contains(merged[0].Content, "榜单02: 灵梦") {
		t.Errorf("digit content should be merged into previous section, got %q", merged[0].Content)
	}
}

func TestParseMagicalGirlWithTipsNoDigitChapters(t *testing.T) {
	txtPath := filepath.Join("..", "..", "examples", "book", "《从魔法少女开始独断万古》作者：绿茶藨子.txt")
	if _, err := os.Stat(txtPath); err != nil {
		t.Skipf("example book not available: %v", err)
	}

	book := &Book{
		Filename:        txtPath,
		DedupTitle:      true,
		NormalizeQuotes: true,
		Tips:            true, // 模拟 GUI 默认开启 Tips 的环境
	}
	book.SetDefault()
	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
	if err := book.Parse(); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	// 验证解析出的所有章节中没有任何纯数字标题（如 53, 33, 97, 1）
	digitRe := regexp.MustCompile(`^\d{1,5}$`)
	for _, sec := range book.SectionList {
		if digitRe.MatchString(strings.TrimSpace(sec.Title)) {
			t.Errorf("found digit-only chapter title: %q", sec.Title)
		}
	}
}

func TestBuildMagicalGirlEpub(t *testing.T) {
	txtPath := filepath.Join("..", "..", "examples", "book", "《从魔法少女开始独断万古》作者：绿茶藨子.txt")
	if _, err := os.Stat(txtPath); err != nil {
		t.Skipf("example book not available: %v", err)
	}

	dir := t.TempDir()
	outPath := filepath.Join(dir, "out")

	book := &Book{
		Filename:        txtPath,
		Out:             outPath,
		Format:          "epub",
		DedupTitle:      true,
		NormalizeQuotes: true,
		Tips:            false,
		Cover:           "none",
	}
	book.SetDefault()
	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
	if err := book.Parse(); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if err := book.Convert(); err != nil {
		t.Fatalf("unexpected convert error: %v", err)
	}

	epubFile := outPath + ".epub"
	stat, err := os.Stat(epubFile)
	if err != nil {
		t.Fatalf("epub file not generated: %v", err)
	}
	if stat.Size() < 100*1024 {
		t.Fatalf("epub file too small: %d bytes", stat.Size())
	}
}

func TestParseAutoDetectsQuotesAndDedup(t *testing.T) {
	dir := t.TempDir()
	txtPath := filepath.Join(dir, "auto_test.txt")
	content := strings.Join([]string{
		"第1章 开启",
		"「你终于来了。」主角说道：「等你好久。」",
		"第2章 冒险",
		"正文内容第二章",
		"第2章 冒险",
		"正文内容第二章重复行",
		"53",
		"积分榜单：100分",
		"第3章 终章",
		"正文第三章",
	}, "\n")
	if err := os.WriteFile(txtPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	// 初始不显式开启 DedupTitle 与 NormalizeQuotes
	book := &Book{
		Filename:        txtPath,
		DedupTitle:      false,
		NormalizeQuotes: false,
		Tips:            false,
	}
	book.SetDefault()
	if err := book.Check("test-version"); err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
	if err := book.Parse(); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	// 验证自适应检测直角引号并规范化
	if !book.NormalizeQuotes {
		t.Error("expected NormalizeQuotes to be automatically enabled")
	}
	if !strings.Contains(book.SectionList[0].Content, "“你终于来了。”主角说道：“等你好久。”") {
		t.Errorf("quotes were not normalized in content: %q", book.SectionList[0].Content)
	}

	// 验证自适应检测到章节重复和孤立数字行并自动去重
	if !book.DedupTitle {
		t.Error("expected DedupTitle to be automatically enabled")
	}
	// 第2章重复与孤立数字53均被合并或清除，最终应为3章
	if len(book.SectionList) != 3 {
		t.Errorf("expected 3 chapters after auto dedup, got %d", len(book.SectionList))
	}
}
