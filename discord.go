package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// Discord Embed 相关结构体

// DiscordEmbed 表示 Discord 消息的嵌入式内容
type DiscordEmbed struct {
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Color       int          `json:"color"`
	Fields      []EmbedField `json:"fields"`
	Footer      *EmbedFooter `json:"footer,omitempty"`
	Timestamp   string       `json:"timestamp,omitempty"`
}

// EmbedField 表示 Embed 中的一个字段
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

// EmbedFooter 表示 Embed 的页脚
type EmbedFooter struct {
	Text string `json:"text"`
}

// discordMessage 表示发送到 Discord 的消息结构
type discordMessage struct {
	Embeds []DiscordEmbed `json:"embeds"`
}

// DiscordSender 负责发送 Discord 消息
type DiscordSender struct {
	botToken   string
	channelID  string
	client     *http.Client
	apiBaseURL string
}

// 重试配置
const (
	discordMaxRetries    = 3
	discordRetryInterval = 10 * time.Second
)

// NewDiscordSender 创建新的 Discord 发送器
func NewDiscordSender(botToken, channelID string, proxyEnabled bool, proxyURL string) *DiscordSender {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 如果启用了代理，配置 HTTP Transport
	if proxyEnabled && proxyURL != "" {
		parsedProxyURL, err := url.Parse(proxyURL)
		if err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(parsedProxyURL),
			}
		}
	}

	return &DiscordSender{
		botToken:   botToken,
		channelID:  channelID,
		client:     client,
		apiBaseURL: "https://discord.com/api/v10",
	}
}

// IsConfigured 检查 Discord 是否已正确配置
func (d *DiscordSender) IsConfigured() bool {
	return d.botToken != "" && d.channelID != ""
}

// SendEmbed 发送 Discord Embed 消息
func (d *DiscordSender) SendEmbed(embed *DiscordEmbed) error {
	if !d.IsConfigured() {
		return fmt.Errorf("Discord 未配置")
	}

	var lastErr error
	for attempt := 1; attempt <= discordMaxRetries; attempt++ {
		err := d.doSendEmbed(embed)
		if err == nil {
			return nil
		}
		lastErr = err

		// 如果是认证或权限错误，不重试
		if isDiscordAuthError(err) || isDiscordPermissionError(err) {
			return err
		}

		if attempt < discordMaxRetries {
			log.Printf("Discord 消息发送失败 (尝试 %d/%d): %v，%v 后重试...",
				attempt, discordMaxRetries, err, discordRetryInterval)
			time.Sleep(discordRetryInterval)
		}
	}

	return fmt.Errorf("Discord 消息发送失败，已重试 %d 次: %w", discordMaxRetries, lastErr)
}

// doSendEmbed 执行实际的发送操作
func (d *DiscordSender) doSendEmbed(embed *DiscordEmbed) error {
	// 构建消息
	message := discordMessage{
		Embeds: []DiscordEmbed{*embed},
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 构建请求 URL
	url := fmt.Sprintf("%s/channels/%s/messages", d.apiBaseURL, d.channelID)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bot "+d.botToken)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return &DiscordAPIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return nil
}

// DiscordAPIError 表示 Discord API 错误
type DiscordAPIError struct {
	StatusCode int
	Message    string
}

func (e *DiscordAPIError) Error() string {
	switch e.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Sprintf("Discord 认证失败 (401): Bot Token 无效或已过期")
	case http.StatusForbidden:
		return fmt.Sprintf("Discord 权限不足 (403): Bot 没有在该频道发送消息的权限")
	case http.StatusNotFound:
		return fmt.Sprintf("Discord 频道不存在 (404): 请检查 channel_id 是否正确")
	case http.StatusTooManyRequests:
		return fmt.Sprintf("Discord API 请求过于频繁 (429): 请稍后重试")
	default:
		return fmt.Sprintf("Discord API 错误 (%d): %s", e.StatusCode, e.Message)
	}
}

// isDiscordAuthError 检查是否是认证错误
func isDiscordAuthError(err error) bool {
	if apiErr, ok := err.(*DiscordAPIError); ok {
		return apiErr.StatusCode == http.StatusUnauthorized
	}
	return false
}

// isDiscordPermissionError 检查是否是权限错误
func isDiscordPermissionError(err error) bool {
	if apiErr, ok := err.(*DiscordAPIError); ok {
		return apiErr.StatusCode == http.StatusForbidden
	}
	return false
}

// SendReport 发送加密货币价格报表到 Discord
func (d *DiscordSender) SendReport(coins []CoinPrice) error {
	if !d.IsConfigured() {
		return nil // 未配置时静默跳过
	}

	gen := NewReportGenerator()
	embed := gen.GenerateDiscordEmbed(coins)

	// 检查 Embed 长度限制（Discord 限制为 6000 字符）
	embed = truncateEmbedIfNeeded(embed)

	return d.SendEmbed(embed)
}

// Discord Embed 字符限制
const (
	maxEmbedTotalLength = 6000
	maxFieldValueLength = 1024
	maxFieldNameLength  = 256
	maxTitleLength      = 256
	maxDescLength       = 4096
)

// truncateEmbedIfNeeded 如果 Embed 超过长度限制，进行截断
func truncateEmbedIfNeeded(embed *DiscordEmbed) *DiscordEmbed {
	// 截断标题
	if len(embed.Title) > maxTitleLength {
		embed.Title = embed.Title[:maxTitleLength-3] + "..."
	}

	// 截断描述
	if len(embed.Description) > maxDescLength {
		embed.Description = embed.Description[:maxDescLength-3] + "..."
	}

	// 截断字段
	for i := range embed.Fields {
		if len(embed.Fields[i].Name) > maxFieldNameLength {
			embed.Fields[i].Name = embed.Fields[i].Name[:maxFieldNameLength-3] + "..."
		}
		if len(embed.Fields[i].Value) > maxFieldValueLength {
			embed.Fields[i].Value = embed.Fields[i].Value[:maxFieldValueLength-3] + "..."
		}
	}

	// 如果总长度仍然超过限制，移除一些字段
	for calculateEmbedLength(embed) > maxEmbedTotalLength && len(embed.Fields) > 0 {
		embed.Fields = embed.Fields[:len(embed.Fields)-1]
		if len(embed.Fields) > 0 {
			embed.Fields[len(embed.Fields)-1].Value += "\n... (更多币种已省略)"
		}
	}

	return embed
}

// calculateEmbedLength 计算 Embed 的总字符数
func calculateEmbedLength(embed *DiscordEmbed) int {
	length := len(embed.Title) + len(embed.Description)
	for _, field := range embed.Fields {
		length += len(field.Name) + len(field.Value)
	}
	if embed.Footer != nil {
		length += len(embed.Footer.Text)
	}
	return length
}
