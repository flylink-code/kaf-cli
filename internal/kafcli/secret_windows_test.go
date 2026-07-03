//go:build windows

package kafcli

import (
	"strings"
	"testing"
)

func TestProtectUnprotectRoundTrip(t *testing.T) {
	cases := []string{
		"sk-abc1234567890def",
		"sk_test_1234567890abcdefgHIJKL",
		"a",
		"普通的中文密钥测试内容",
	}
	for _, plain := range cases {
		cipher, err := Protect(plain)
		if err != nil {
			t.Fatalf("Protect(%q) error: %v", plain, err)
		}
		if plain != "" && cipher == plain {
			t.Fatalf("Protect(%q) did not encrypt: cipher==plain", plain)
		}
		got, err := Unprotect(cipher)
		if err != nil {
			t.Fatalf("Unprotect error: %v", err)
		}
		if got != plain {
			t.Fatalf("round-trip mismatch: got %q want %q", got, plain)
		}
	}
}

func TestProtectEmptyReturnsEmpty(t *testing.T) {
	c, err := Protect("")
	if err != nil || c != "" {
		t.Fatalf("Protect(\"\") = (%q,%v), want (\"\",nil)", c, err)
	}
	g, err := Unprotect("")
	if err != nil || g != "" {
		t.Fatalf("Unprotect(\"\") = (%q,%v), want (\"\",nil)", g, err)
	}
}

func TestUnprotectGarbageFails(t *testing.T) {
	_, err := Unprotect("这不是合法密文!!!")
	if err == nil {
		t.Fatal("expected error for garbage input")
	}
}

func TestMaskSecret(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"sk-abcdef1234567890", "sk-" + strings.Repeat("*", 8) + "90"},
		{"sk_abcdef1234567890", "sk_" + strings.Repeat("*", 8) + "90"},
		{"abcdef0123456789abcdef0123456789", "abc" + strings.Repeat("*", 8) + "89"},
		{"short", "short"},  // 5字符，不匹配 sk- 且长度<20，不打码
		{"ab", "ab"},
		{"", ""},
	}
	for _, c := range cases {
		got := MaskSecret(c.in)
		if got != c.want {
			t.Errorf("MaskSecret(%q) = %q, want %q", c.in, got, c.want)
		}
		// 确保打码后不再包含完整原文（长度足够时）
		if len(c.in) >= 20 && strings.Contains(got, c.in) {
			t.Errorf("MaskSecret(%q) leaked full secret: %q", c.in, got)
		}
	}
}

func TestMaskSecretPreservesNonSecrets(t *testing.T) {
	// 普通短文本（如模型名、错误说明）不应被误打码
	for _, in := range []string{"deepseek-chat", "连接成功", "请求超时"} {
		if MaskSecret(in) != in {
			t.Errorf("MaskSecret(%q) should be unchanged", in)
		}
	}
}
