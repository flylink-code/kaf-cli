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
