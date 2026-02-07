package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNewDiscordSender 测试 DiscordSender 客户端创建（带代理配置）
func TestNewDiscordSender(t *testing.T) {
	// 测试不带代理创建
	sender := NewDiscordSender("test-token", "123456789", false, "")
	if sender == nil {
		t.Fatal("NewDiscordSender 返回 nil")
	}
	if sender.botToken != "test-token" {
		t.Errorf("botToken 期望 'test-token'，实际为 '%s'", sender.botToken)
	}
	if sender.channelID != "123456789" {
		t.Errorf("channelID 期望 '123456789'，实际为 '%s'", sender.channelID)
	}

	// 测试带代理创建
	senderWithProxy := NewDiscordSender("test-token", "123456789", true, "http://127.0.0.1:8080")
	if senderWithProxy == nil {
		t.Fatal("带代理的 NewDiscordSender 返回 nil")
	}
}

// TestDiscordSenderIsConfigured 测试 IsConfigured 方法正确判断配置状态
func TestDiscordSenderIsConfigured(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		channelID string
		expected  bool
	}{
		{"完整配置", "token", "123", true},
		{"缺少 token", "", "123", false},
		{"缺少 channelID", "token", "", false},
		{"都为空", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := NewDiscordSender(tt.token, tt.channelID, false, "")
			if sender.IsConfigured() != tt.expected {
				t.Errorf("IsConfigured() = %v，期望 %v", sender.IsConfigured(), tt.expected)
			}
		})
	}
}

// TestDiscordSenderSendEmbed 测试 SendEmbed 成功发送消息
func TestDiscordSenderSendEmbed(t *testing.T) {
	// 创建 mock HTTP 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法
		if r.Method != "POST" {
			t.Errorf("期望 POST 方法，实际为 %s", r.Method)
		}

		// 验证 Authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bot test-token" {
			t.Errorf("Authorization header 错误: %s", auth)
		}

		// 验证 Content-Type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type 错误: %s", contentType)
		}

		// 返回成功响应
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": "123", "channel_id": "456"}`))
	}))
	defer server.Close()

	// 创建测试用的 embed
	embed := &DiscordEmbed{
		Title:       "测试报表",
		Description: "2026年02月07日",
		Color:       0xFFD700, // 金色
		Fields: []EmbedField{
			{Name: "Bitcoin (BTC)", Value: "**$45,000**\n24h: +2.5%", Inline: true},
		},
		Footer: &EmbedFooter{Text: "数据来源: CoinGecko API"},
	}

	sender := NewDiscordSender("test-token", "123456789", false, "")
	// 替换 API 地址为 mock 服务器
	sender.apiBaseURL = server.URL

	err := sender.SendEmbed(embed)
	if err != nil {
		t.Errorf("SendEmbed 失败: %v", err)
	}
}

// TestDiscordSenderSendEmbedUnauthorized 测试认证失败场景
func TestDiscordSenderSendEmbedUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message": "401: Unauthorized"}`))
	}))
	defer server.Close()

	embed := &DiscordEmbed{Title: "测试"}
	sender := NewDiscordSender("invalid-token", "123456789", false, "")
	sender.apiBaseURL = server.URL

	err := sender.SendEmbed(embed)
	if err == nil {
		t.Error("认证失败时应该返回错误")
	}
}

// TestDiscordSenderSendEmbedForbidden 测试权限不足场景
func TestDiscordSenderSendEmbedForbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message": "Missing Permissions"}`))
	}))
	defer server.Close()

	embed := &DiscordEmbed{Title: "测试"}
	sender := NewDiscordSender("test-token", "123456789", false, "")
	sender.apiBaseURL = server.URL

	err := sender.SendEmbed(embed)
	if err == nil {
		t.Error("权限不足时应该返回错误")
	}
}
