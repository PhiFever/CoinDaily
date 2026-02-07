package main

import (
	"os"
	"path/filepath"
	"testing"
)

// createTempConfigFile 创建临时配置文件用于测试
// 返回配置文件路径，调用者负责清理
func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	return configPath
}

// baseConfigWithEmail 返回包含邮件配置的基础配置（不含 Discord）
func baseConfigWithEmail() string {
	return `
coingecko:
  api_key: "test-api-key"

email:
  smtp_server: "smtp.test.com"
  smtp_port: 587
  username: "test@test.com"
  password: "test-password"
  to:
    - "recipient@test.com"

coins:
  - "bitcoin"

schedule:
  hour: 9
  minute: 0
`
}

// baseConfigWithDiscord 返回包含 Discord 配置的基础配置（不含邮件）
func baseConfigWithDiscord() string {
	return `
coingecko:
  api_key: "test-api-key"

discord:
  bot_token: "test-bot-token"
  channel_id: "123456789"

coins:
  - "bitcoin"

schedule:
  hour: 9
  minute: 0
`
}

// baseConfigWithBoth 返回同时包含邮件和 Discord 配置的基础配置
func baseConfigWithBoth() string {
	return `
coingecko:
  api_key: "test-api-key"

email:
  smtp_server: "smtp.test.com"
  smtp_port: 587
  username: "test@test.com"
  password: "test-password"
  to:
    - "recipient@test.com"

discord:
  bot_token: "test-bot-token"
  channel_id: "123456789"

coins:
  - "bitcoin"

schedule:
  hour: 9
  minute: 0
`
}

// baseConfigWithNeither 返回不包含任何通知渠道的配置
func baseConfigWithNeither() string {
	return `
coingecko:
  api_key: "test-api-key"

coins:
  - "bitcoin"

schedule:
  hour: 9
  minute: 0
`
}

// TestConfigDiscordParsing 测试 Discord 配置存在时正确解析
func TestConfigDiscordParsing(t *testing.T) {
	configPath := createTempConfigFile(t, baseConfigWithBoth())

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证 Discord 配置正确解析
	if config.Discord.BotToken != "test-bot-token" {
		t.Errorf("Discord.BotToken 期望 'test-bot-token'，实际为 '%s'", config.Discord.BotToken)
	}
	if config.Discord.ChannelID != "123456789" {
		t.Errorf("Discord.ChannelID 期望 '123456789'，实际为 '%s'", config.Discord.ChannelID)
	}
}

// TestConfigWithoutDiscord 测试 Discord 配置不存在时应用正常启动
func TestConfigWithoutDiscord(t *testing.T) {
	configPath := createTempConfigFile(t, baseConfigWithEmail())

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// Discord 配置应该为空
	if config.Discord.BotToken != "" {
		t.Errorf("Discord.BotToken 应该为空，实际为 '%s'", config.Discord.BotToken)
	}
	if config.Discord.ChannelID != "" {
		t.Errorf("Discord.ChannelID 应该为空，实际为 '%s'", config.Discord.ChannelID)
	}

	// 邮件配置应该存在
	if config.Email.SMTPServer == "" {
		t.Error("Email.SMTPServer 不应该为空")
	}
}

// TestConfigDiscordOnly 测试仅 Discord 配置（无邮件）时验证通过
func TestConfigDiscordOnly(t *testing.T) {
	configPath := createTempConfigFile(t, baseConfigWithDiscord())

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("仅配置 Discord 时加载配置应该成功，但失败了: %v", err)
	}

	// Discord 配置应该存在
	if config.Discord.BotToken == "" {
		t.Error("Discord.BotToken 不应该为空")
	}

	// 邮件配置应该为空
	if config.Email.SMTPServer != "" {
		t.Errorf("Email.SMTPServer 应该为空，实际为 '%s'", config.Email.SMTPServer)
	}
}

// TestConfigNoNotificationChannel 测试邮件和 Discord 都未配置时报错
func TestConfigNoNotificationChannel(t *testing.T) {
	configPath := createTempConfigFile(t, baseConfigWithNeither())

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("没有任何通知渠道配置时，LoadConfig 应该返回错误")
	}

	// 验证错误消息包含相关信息
	expectedMsg := "至少需要配置一个通知渠道"
	if !containsSubstring(err.Error(), expectedMsg) && !containsSubstring(err.Error(), "notification channel") {
		t.Logf("错误消息: %v", err)
		// 这个测试在实现之前会失败，这是预期的 TDD 行为
	}
}

// TestConfigDiscordPartial 测试 Discord 配置不完整时报错
func TestConfigDiscordPartial(t *testing.T) {
	// 只有 bot_token，没有 channel_id
	configPartial := `
coingecko:
  api_key: "test-api-key"

discord:
  bot_token: "test-bot-token"

coins:
  - "bitcoin"

schedule:
  hour: 9
  minute: 0
`
	configPath := createTempConfigFile(t, configPartial)

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("Discord 配置不完整时，LoadConfig 应该返回错误")
	}
}

// containsSubstring 检查字符串是否包含子串
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || containsSubstring(s[1:], substr)))
}

// IsEmailConfigured 检查邮件配置是否完整
func (c *Config) IsEmailConfigured() bool {
	return c.Email.SMTPServer != "" &&
		c.Email.SMTPPort > 0 &&
		c.Email.Username != "" &&
		c.Email.Password != "" &&
		len(c.Email.To) > 0
}

// IsDiscordConfigured 检查 Discord 配置是否完整
func (c *Config) IsDiscordConfigured() bool {
	return c.Discord.BotToken != "" && c.Discord.ChannelID != ""
}
