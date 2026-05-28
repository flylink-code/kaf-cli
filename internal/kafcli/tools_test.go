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
