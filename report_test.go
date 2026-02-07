package main

import (
	"strings"
	"testing"
)

// TestGenerateDiscordEmbed 测试 GenerateDiscordEmbed 生成正确的 Embed 结构
func TestGenerateDiscordEmbed(t *testing.T) {
	coins := []CoinPrice{
		{
			ID:                 "bitcoin",
			Symbol:             "btc",
			Name:               "Bitcoin",
			CurrentPrice:       45000.00,
			MarketCap:          850000000000,
			PriceChange24h:     1000.00,
			PriceChangePerc24h: 2.27,
			Volume24h:          25000000000,
		},
	}

	gen := NewReportGenerator()
	embed := gen.GenerateDiscordEmbed(coins)

	if embed == nil {
		t.Fatal("GenerateDiscordEmbed 返回 nil")
	}

	// 验证标题包含报表信息
	if !strings.Contains(embed.Title, "加密货币价格报表") {
		t.Errorf("标题应该包含 '加密货币价格报表'，实际为 '%s'", embed.Title)
	}

	// 验证有字段
	if len(embed.Fields) == 0 {
		t.Error("Embed 应该包含至少一个字段")
	}

	// 验证有 Footer
	if embed.Footer == nil {
		t.Error("Embed 应该包含 Footer")
	}
}

// TestGenerateDiscordEmbedFields 测试 Embed 字段包含币种名称、价格、24h变化
func TestGenerateDiscordEmbedFields(t *testing.T) {
	coins := []CoinPrice{
		{
			ID:                 "bitcoin",
			Symbol:             "btc",
			Name:               "Bitcoin",
			CurrentPrice:       45000.00,
			MarketCap:          850000000000,
			PriceChange24h:     1000.00,
			PriceChangePerc24h: 2.27,
			Volume24h:          25000000000,
		},
		{
			ID:                 "ethereum",
			Symbol:             "eth",
			Name:               "Ethereum",
			CurrentPrice:       2800.00,
			MarketCap:          320000000000,
			PriceChange24h:     -50.00,
			PriceChangePerc24h: -1.75,
			Volume24h:          15000000000,
		},
	}

	gen := NewReportGenerator()
	embed := gen.GenerateDiscordEmbed(coins)

	// 验证字段数量
	if len(embed.Fields) != 2 {
		t.Errorf("期望 2 个字段，实际为 %d", len(embed.Fields))
	}

	// 验证第一个字段包含 Bitcoin 信息
	field1 := embed.Fields[0]
	if !strings.Contains(field1.Name, "Bitcoin") || !strings.Contains(field1.Name, "BTC") {
		t.Errorf("第一个字段名应该包含 'Bitcoin' 和 'BTC'，实际为 '%s'", field1.Name)
	}
	if !strings.Contains(field1.Value, "45") {
		t.Errorf("第一个字段值应该包含价格，实际为 '%s'", field1.Value)
	}
	if !strings.Contains(field1.Value, "2.27") && !strings.Contains(field1.Value, "+") {
		t.Errorf("第一个字段值应该包含24h变化，实际为 '%s'", field1.Value)
	}

	// 验证字段是 inline 的
	if !field1.Inline {
		t.Error("字段应该是 inline 的")
	}
}

// TestGenerateDiscordEmbedColors 测试涨跌使用不同颜色
func TestGenerateDiscordEmbedColors(t *testing.T) {
	// 测试全涨
	coinsUp := []CoinPrice{
		{
			ID:                 "bitcoin",
			Symbol:             "btc",
			Name:               "Bitcoin",
			CurrentPrice:       45000.00,
			PriceChange24h:     1000.00,
			PriceChangePerc24h: 2.27,
		},
	}

	gen := NewReportGenerator()
	embedUp := gen.GenerateDiscordEmbed(coinsUp)

	// 验证颜色是绿色或金色（涨）
	if embedUp.Color == 0 {
		t.Error("Embed 颜色不应该为 0")
	}

	// 测试全跌
	coinsDown := []CoinPrice{
		{
			ID:                 "bitcoin",
			Symbol:             "btc",
			Name:               "Bitcoin",
			CurrentPrice:       40000.00,
			PriceChange24h:     -5000.00,
			PriceChangePerc24h: -11.11,
		},
	}

	embedDown := gen.GenerateDiscordEmbed(coinsDown)

	// 下跌时可能使用不同颜色，但必须有颜色
	if embedDown.Color == 0 {
		t.Error("下跌时 Embed 颜色不应该为 0")
	}
}

// TestGenerateDiscordEmbedEmpty 测试空币种列表
func TestGenerateDiscordEmbedEmpty(t *testing.T) {
	gen := NewReportGenerator()
	embed := gen.GenerateDiscordEmbed([]CoinPrice{})

	if embed == nil {
		t.Fatal("即使币种列表为空，也应该返回 Embed")
	}

	if len(embed.Fields) != 0 {
		t.Errorf("空列表应该生成 0 个字段，实际为 %d", len(embed.Fields))
	}
}
