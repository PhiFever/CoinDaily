package main

import (
	"testing"
)

// TestSchedulerInitWithBothChannels 测试 Scheduler 同时初始化 EmailSender 和 DiscordSender
func TestSchedulerInitWithBothChannels(t *testing.T) {
	configPath := createTempConfigFile(t, baseConfigWithBoth())

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	scheduler := NewScheduler(config)
	if scheduler == nil {
		t.Fatal("NewScheduler 返回 nil")
	}

	// 验证 emailSender 已初始化
	if scheduler.emailSender == nil {
		t.Error("emailSender 应该被初始化")
	}

	// 验证 discordSender 已初始化
	if scheduler.discordSender == nil {
		t.Error("discordSender 应该被初始化")
	}

	// 验证 Discord 配置正确
	if !scheduler.discordSender.IsConfigured() {
		t.Error("discordSender 应该已配置")
	}
}

// TestSchedulerInitDiscordOnly 测试仅 Discord 配置时 Scheduler 正常初始化
func TestSchedulerInitDiscordOnly(t *testing.T) {
	configPath := createTempConfigFile(t, baseConfigWithDiscord())

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	scheduler := NewScheduler(config)
	if scheduler == nil {
		t.Fatal("NewScheduler 返回 nil")
	}

	// emailSender 可能为 nil 或未配置
	if scheduler.emailSender != nil && scheduler.emailSender.IsConfigured() {
		t.Error("仅配置 Discord 时，emailSender 不应该被配置")
	}

	// 验证 discordSender 已初始化
	if scheduler.discordSender == nil {
		t.Error("discordSender 应该被初始化")
	}
}

// TestSchedulerInitEmailOnly 测试仅邮件配置时 Scheduler 正常初始化
func TestSchedulerInitEmailOnly(t *testing.T) {
	configPath := createTempConfigFile(t, baseConfigWithEmail())

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	scheduler := NewScheduler(config)
	if scheduler == nil {
		t.Fatal("NewScheduler 返回 nil")
	}

	// emailSender 应该被初始化
	if scheduler.emailSender == nil {
		t.Error("emailSender 应该被初始化")
	}

	// discordSender 可能为 nil 或未配置
	if scheduler.discordSender != nil && scheduler.discordSender.IsConfigured() {
		t.Error("仅配置邮件时，discordSender 不应该被配置")
	}
}

// EmailSenderInterface 用于测试的邮件发送器接口
type EmailSenderInterface interface {
	SendReport(subject, htmlContent string) error
	IsConfigured() bool
}
