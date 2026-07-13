package ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"dida/models"
)

// DeepSeekClient DeepSeek API 客户端
type DeepSeekClient struct {
	endpoint string
	apiKey   string
	model    string
	client   *http.Client
}

// NewDeepSeekClient 创建 DeepSeek API 客户端
func NewDeepSeekClient(endpoint, apiKey, model string) *DeepSeekClient {
	if endpoint == "" {
		endpoint = "https://api.deepseek.com"
	}
	if model == "" {
		model = "deepseek-chat"
	}
	return &DeepSeekClient{
		endpoint: endpoint,
		apiKey:   apiKey,
		model:    model,
		client: &http.Client{
			Timeout: 300 * time.Second,
		},
	}
}

// apiURL 根据 endpoint 自动推断兼容的 API 路径
func (c *DeepSeekClient) apiURL() string {
	e := strings.TrimRight(c.endpoint, "/")
	if strings.Contains(e, "/chat/completions") {
		return e
	}
	if strings.Contains(e, "openrouter.ai") {
		return e + "/api/v1/chat/completions"
	}
	return e + "/v1/chat/completions"
}

// Chat 发送聊天请求并获取完整响应
func (c *DeepSeekClient) Chat(messages []models.Message, temperature *float32) (string, error) {
	req := models.AIRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	}
	if temperature != nil {
		req.Temperature = temperature
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	msgCount := len(messages)
	totalChars := 0
	for _, m := range messages {
		totalChars += len([]rune(m.Content))
	}
	estimatedTokens := totalChars / 2
	log.Printf("[AI] 请求 model=%s | endpoint=%s | 消息数=%d | 输入字符数=%d | 估算token≈%d",
		c.model, c.endpoint, msgCount, totalChars, estimatedTokens)

	apiURL := c.apiURL()
	log.Printf("[AI] 发送请求 → %s", apiURL)

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	start := time.Now()
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API返回错误(%d): %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	var aiResp models.AIResponse
	if err := json.Unmarshal(respBody, &aiResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if len(aiResp.Choices) == 0 {
		return "", fmt.Errorf("API响应中没有choices")
	}

	responseContent := aiResp.Choices[0].Message.Content
	log.Printf("[AI] 响应完成 | 输出字符数=%d | 耗时=%v",
		len([]rune(responseContent)), time.Since(start))
	return responseContent, nil
}

// StreamChat 发送流式聊天请求，通过 onChunk 回调逐块返回内容增量
func (c *DeepSeekClient) StreamChat(messages []models.Message, temperature *float32, onChunk func(string)) (string, error) {
	req := models.AIRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   true,
	}
	if temperature != nil {
		req.Temperature = temperature
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	msgCount := len(messages)
	totalChars := 0
	for _, m := range messages {
		totalChars += len([]rune(m.Content))
	}
	log.Printf("[AI] 流式请求 model=%s | endpoint=%s | 消息数=%d | 输入字符数=%d",
		c.model, c.endpoint, msgCount, totalChars)

	apiURL := c.apiURL()
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	start := time.Now()
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API返回错误(%d): %s", resp.StatusCode, string(respBody))
	}

	var fullText strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		data = strings.TrimSpace(data)
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				fullText.WriteString(content)
				onChunk(content)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fullText.String(), fmt.Errorf("读取流失败: %w", err)
	}

	log.Printf("[AI] 流式响应完成 | 输出字符数=%d | 耗时=%v",
		len([]rune(fullText.String())), time.Since(start))
	return fullText.String(), nil
}

// CheckConnection 检查API连接是否正常
func (c *DeepSeekClient) CheckConnection() error {
	messages := []models.Message{
		{Role: "user", Content: "Hi"},
	}
	_, err := c.Chat(messages, nil)
	return err
}
