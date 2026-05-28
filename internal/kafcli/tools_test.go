package kafcli

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

func TestLook(t *testing.T) {
	kindlegen, err := exec.LookPath("kindlegen")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(kindlegen)
}

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

func TestExe(t *testing.T) {
	fmt.Println(os.Args)
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(path)
}
