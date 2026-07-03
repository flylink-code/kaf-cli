// Package ai 提供基于 OpenAI 兼容协议的章节后处理能力。
// 作为 kafcli 规则引擎的可选增强层，默认关闭、离线优先，
// 任一调用失败均静默降级到原始结果，绝不阻断转换流程。
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Client 是 OpenAI 兼容 Chat Completions 客户端。
// 覆盖 OpenAI / DeepSeek / Qwen / Moonshot / 本地 vLLM / Ollama 等遵循
// {baseURL}/chat/completions 协议的服务。
type Client struct {
	baseURL   string
	apiKey    string
	model     string
	http      *http.Client
	maxTokens int
}

// ClientConfig 配置一个 Client。
type ClientConfig struct {
	BaseURL   string        // 例 https://api.deepseek.com/v1；尾斜杠会被去掉
	APIKey    string        // Bearer token
	Model     string        // 例 deepseek-chat
	Timeout   time.Duration // 默认 120s
	MaxTokens int           // 单次响应上限，默认 2048
}

// NewClient 构造客户端。BaseURL 为空时回退到 OpenAI 官方地址。
func NewClient(cfg ClientConfig) *Client {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 2048
	}
	return &Client{
		baseURL: baseURL,
		apiKey:  strings.TrimSpace(cfg.APIKey),
		model:   strings.TrimSpace(cfg.Model),
		http: &http.Client{
			Timeout: timeout,
		},
		maxTokens: maxTokens,
	}
}

// Ready 表示客户端是否具备发起请求的最小条件。
func (c *Client) Ready() bool {
	return c != nil && c.apiKey != "" && c.model != ""
}

type chatMessage struct {
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content"` // DeepSeek 思考/推理模型
}

type chatChoice struct {
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	Temperature    float64         `json:"temperature"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   *apiError    `json:"error,omitempty"`
}

const chatMaxAttempts = 3 // JSON 模式下 DeepSeek 偶发空 content，官方建议重试

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// Chat 发送一次对话请求，返回助手回复内容。
// 当 wantJSON 为 true 时设置 response_format=json_object，
// 兼容 DeepSeek/Qwen/OpenAI；不支持该字段的服务端会忽略它。
// DeepSeek 在 json_object 模式下偶发空 content，会自动重试最多 chatMaxAttempts 次。
func (c *Client) Chat(ctx context.Context, system, user string, wantJSON bool) (string, error) {
	if c == nil || !c.Ready() {
		return "", fmt.Errorf("AI 客户端未配置 api_key 或 model")
	}
	reqBody := chatRequest{
		Model:       c.model,
		Temperature: 0.2,
		MaxTokens:   c.maxTokens,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}
	if wantJSON {
		reqBody.ResponseFormat = &responseFormat{Type: "json_object"}
	}

	raw, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("构造请求失败: %w", err)
	}

	attempts := 1
	if wantJSON {
		attempts = chatMaxAttempts
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(i) * 400 * time.Millisecond):
			}
		}
		content, finishReason, body, err := c.doChat(ctx, raw)
		if err != nil {
			return "", err
		}
		if content != "" {
			return content, nil
		}
		lastErr = fmt.Errorf("AI 服务返回空内容 (finish_reason=%s)，原始响应: %s",
			finishReason, maskKeyInString(truncateBody(body, 500)))
	}
	return "", lastErr
}

func (c *Client) doChat(ctx context.Context, raw []byte) (content, finishReason string, body []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", "", nil, fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", "", nil, fmt.Errorf("请求 AI 服务失败: %w", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", "", nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", body, fmt.Errorf("AI 服务返回异常状态 %s: %s", resp.Status, maskKeyInString(truncateBody(body, 300)))
	}

	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", body, fmt.Errorf("解析响应失败: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", "", body, fmt.Errorf("AI 服务报错: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", "", body, fmt.Errorf("AI 服务未返回任何结果")
	}
	choice := parsed.Choices[0]
	finishReason = choice.FinishReason
	content = extractAssistantContent(choice.Message)
	return content, finishReason, body, nil
}

// extractAssistantContent 取助手正文。思考模型常在 content 放说明、在 reasoning_content 放 json。
func extractAssistantContent(msg chatMessage) string {
	content := strings.TrimSpace(msg.Content)
	reasoning := strings.TrimSpace(msg.ReasoningContent)
	if strings.Contains(content, "{") {
		return content
	}
	if strings.Contains(reasoning, "{") {
		return reasoning
	}
	if content != "" {
		return content
	}
	return reasoning
}

func truncateBody(body []byte, limit int) string {
	s := string(body)
	if len(s) > limit {
		return s[:limit] + "..."
	}
	return s
}

// maskKeyInString 把文本中疑似 API Key 的片段打码，避免 key 出现在日志/错误中。
// 覆盖两种情形：本 client 持有的 apiKey、以及任意 sk-/sk_ 开头的串。
func (c *Client) maskKeyInString(s string) string {
	if s == "" {
		return s
	}
	out := s
	// 先遮蔽本 client 的 key（整体替换为打码形式）
	if c != nil && c.apiKey != "" && len(c.apiKey) >= 8 {
		masked := maskKey(c.apiKey)
		out = strings.ReplaceAll(out, c.apiKey, masked)
	}
	// 再扫描 sk- / sk_ 开头的残留片段
	out = maskKeyPatterns(out)
	return out
}

// maskKey 把单个 key 打码，保留前3后2。
func maskKey(k string) string {
	if len(k) <= 6 {
		return strings.Repeat("*", len(k))
	}
	stars := 8
	if len(k)-5 < stars {
		stars = len(k) - 5
	}
	return k[:3] + strings.Repeat("*", stars) + k[len(k)-2:]
}

// maskKeyPatterns 用正则匹配 sk-xxx / sk_xxx 形态并打码。
var skPattern = regexp.MustCompile(`sk[-_][A-Za-z0-9_]{6,}`)

func maskKeyPatterns(s string) string {
	return skPattern.ReplaceAllStringFunc(s, func(m string) string {
		return maskKey(m)
	})
}

// maskKeyInString 是包级便捷函数，供无 client 引用处使用。
func maskKeyInString(s string) string {
	return maskKeyPatterns(s)
}
