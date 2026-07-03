//go:build windows

package kafcli

import (
	"encoding/base64"
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

// DPAPI（Data Protection API）是 Windows 提供的对称加密接口，
// 密钥由当前用户账户的登录凭据派生，因此：
//   - 只有同一个 Windows 账户能解密
//   - 把配置文件拷到别的机器 / 别的账户，密文无法还原，需重填
// 这正是存储 API Key 等本地敏感凭据的标准做法（VS Code、git credential 等同此）。
//
// 本文件通过 syscall 直接调用 crypt32.dll，不引入任何第三方依赖。

var (
	modCrypt32          = syscall.NewLazyDLL("crypt32.dll")
	procCryptProtect    = modCrypt32.NewProc("CryptProtectData")
	procCryptUnprotect  = modCrypt32.NewProc("CryptUnprotectData")
)

// DATA_BLOB 是 DPAPI 的输入/输出结构。
type dataBlob struct {
	Size uint32
	Data uintptr
}

const (
	dpapiDescription = "kaf-cli api key" // DPAPI 附加的描述字符串，便于辨识
	dpapiEntropy     = "kafcli-ai-v1"    // 应用级熵，进一步限定解密者范围
)

// Protect 用 DPAPI 加密明文，返回 base64 编码的密文（便于写入 JSON）。
// 出错时返回 error，调用方可回退到原文（不致中断使用）。
func Protect(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	plainBytes := []byte(plain)
	inBlob, err := bytesToBlob(plainBytes)
	if err != nil {
		return "", err
	}
	defer freeBlob(inBlob)

	descPtr, err := syscall.UTF16PtrFromString(dpapiDescription)
	if err != nil {
		return "", err
	}
	entropyBlob, err := bytesToBlob([]byte(dpapiEntropy))
	if err != nil {
		return "", err
	}
	defer freeBlob(entropyBlob)

	var out dataBlob
	// CryptProtectData(DATA_BLOB* dataIn, LPCWSTR szDataDescr, DATA_BLOB* optionalEntropy, ...)
	ret, _, callErr := procCryptProtect.Call(
		uintptr(unsafe.Pointer(inBlob)),
		uintptr(unsafe.Pointer(descPtr)),
		uintptr(unsafe.Pointer(entropyBlob)),
		0, 0, 0, // reserved / prompt / flags
		uintptr(unsafe.Pointer(&out)),
	)
	if ret == 0 {
		return "", fmt.Errorf("DPAPI 加密失败: %w", callErr)
	}
	defer freeBlob(&out)
	encrypted := blobToBytes(&out)
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// Unprotect 解密 DPAPI 密文。出错时返回 error（如换机器/换账户），调用方应回退要求重填。
func Unprotect(cipherB64 string) (string, error) {
	if cipherB64 == "" {
		return "", nil
	}
	cipherBytes, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return "", fmt.Errorf("密文不是合法 base64: %w", err)
	}
	inBlob, err := bytesToBlob(cipherBytes)
	if err != nil {
		return "", err
	}
	defer freeBlob(inBlob)

	entropyBlob, err := bytesToBlob([]byte(dpapiEntropy))
	if err != nil {
		return "", err
	}
	defer freeBlob(entropyBlob)

	var out dataBlob
	ret, _, callErr := procCryptUnprotect.Call(
		uintptr(unsafe.Pointer(inBlob)),
		0, // 不取描述
		uintptr(unsafe.Pointer(entropyBlob)),
		0, 0, 0,
		uintptr(unsafe.Pointer(&out)),
	)
	if ret == 0 {
		return "", fmt.Errorf("DPAPI 解密失败（可能换机器/账户）: %w", callErr)
	}
	defer freeBlob(&out)
	return string(blobToBytes(&out)), nil
}

// bytesToBlob 分配一块内存并构造 dataBlob。调用方负责 freeBlob。
func bytesToBlob(b []byte) (*dataBlob, error) {
	if len(b) == 0 {
		return &dataBlob{}, nil
	}
	// LocalAlloc(LPTR, size) 分配可释放内存
	modKernel32 := syscall.NewLazyDLL("kernel32.dll")
	procLocalAlloc := modKernel32.NewProc("LocalAlloc")
	const LPTR = 0x0040
	ptr, _, _ := procLocalAlloc.Call(LPTR, uintptr(len(b)))
	if ptr == 0 {
		return nil, fmt.Errorf("LocalAlloc 失败")
	}
	copy((*[1 << 30]byte)(unsafe.Pointer(ptr))[:len(b)], b)
	return &dataBlob{Size: uint32(len(b)), Data: ptr}, nil
}

func blobToBytes(b *dataBlob) []byte {
	if b == nil || b.Size == 0 {
		return nil
	}
	return (*[1 << 30]byte)(unsafe.Pointer(b.Data))[:b.Size:b.Size]
}

func freeBlob(b *dataBlob) {
	if b == nil || b.Data == 0 {
		return
	}
	modKernel32 := syscall.NewLazyDLL("kernel32.dll")
	procLocalFree := modKernel32.NewProc("LocalFree")
	procLocalFree.Call(b.Data)
	b.Data = 0
	b.Size = 0
}

// MaskSecret 把疑似 API Key 的片段打码，用于日志/错误消息脱敏。
// 规则：sk- 开头或长度>=20 的连续十六进制/Base64 串，保留前3后2字符。
func MaskSecret(s string) string {
	if s == "" {
		return s
	}
	// sk-xxxxxxxx... 形态
	if len(s) > 3 && (s[:3] == "sk-" || s[:3] == "sk_") {
		return maskMiddle(s)
	}
	// 长密钥形态：>=20 个可见字符且无明显空格
	if len(s) >= 20 && !containsSpace(s) {
		return maskMiddle(s)
	}
	return s
}

func maskMiddle(s string) string {
	if len(s) <= 6 {
		return strings.Repeat("*", len(s))
	}
	return s[:3] + strings.Repeat("*", min(8, len(s)-5)) + s[len(s)-2:]
}

func containsSpace(s string) bool {
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
