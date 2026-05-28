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

func TestLooksLikeInlineChapterLineKeepsNormalQuotedTitles(t *testing.T) {
	for _, line := range []string{
		"第29章 二十八章「第一玩家？」",
		"第403章 四百零一章“没有战争的世界。”",
		"第67章 六十六章“她是完美的诺丽雅。”",
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

	for _, titlePart := range []string{
		"第29章 二十八章",
		"第67章 六十六章",
		"第331章 三百二十九章",
		"第335章 三百三十三章",
		"第337章 三百三十五章",
		"第344章 三百四十二章",
		"第403章 四百零一章",
	} {
		count, matches := countSectionTitlesContaining(book.SectionList, titlePart)
		if count != 1 {
			t.Fatalf("expected %q to appear once after dedup, got %d, matches=%v details=%v", titlePart, count, matches, collectSectionDebugContaining(book.SectionList, titlePart))
		}
	}

	for _, bad := range []string{"第二回合", "第三回合", "1、苏明安"} {
		count, matches := countSectionTitlesContaining(book.SectionList, bad)
		if count != 0 {
			t.Fatalf("expected %q to stay out of bookmarks, got %d matches=%v", bad, count, matches)
		}
	}

	for _, bad := range []string{"第一节课是数学课", "第一章·白莲灭城", "开启下一章", "第一章87%", "第一章86%", "只有第一章"} {
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
		"第一章·白莲灭城",
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
