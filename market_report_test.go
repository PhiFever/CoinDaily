package main

import (
	"strings"
	"testing"
	"time"
)

func TestMarketReportRendersThreeSemanticSections(t *testing.T) {
	now := time.Date(2026, 8, 25, 20, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	marketCapChange := -2_000_000.0
	daily := 2.5
	turnover := 500_000.0
	vwap := 499.0
	direction := -1.0
	oracle := 99.0
	spread := 1.0101
	spreadAbsolute := 1.0
	price24h := 11.11
	funding := 0.0001
	oi := 500.0
	oiDelta := 30.0
	report := &MarketReport{
		GeneratedAt: now,
		Crypto: CryptoSection{Configured: true, Warning: "CoinGecko 返回了部分数据", Items: []CoinPrice{{
			Name: "Bitcoin", Symbol: "btc", CurrentPrice: 60_000, PriceChangePerc24h: 1.2, MarketCap: 1e12, Volume24h: 1e9, MarketCapChange24h: &marketCapChange, LastUpdated: "2026-08-25T11:59:00Z",
		}}},
		Stocks: StockSection{Configured: true, Feed: "delayed_sip", Items: []StockPrice{{
			Symbol: "QQQ", Price: 500, PriceTime: now.Add(-15 * time.Minute), DailyChangePercent: &daily, HourlyTurnover: &turnover, HourlyVWAP: &vwap, HourlyDirection: &direction,
		}}},
		Perpetuals: PerpetualSection{Configured: true, Items: []PerpetualPrice{{
			Symbol: "xyz:ZHIPU", MarkPrice: 100, OraclePrice: &oracle, MarkOracleSpread: &spreadAbsolute, MarkOracleSpreadPct: &spread, PriceChange24hPct: &price24h, FundingRate: &funding, NotionalOpenInterest: &oi, OpenInterestChange: &oiDelta, ObservedAt: now,
		}}},
	}
	gen := NewReportGenerator()
	html := gen.GenerateMarketHTML(report)
	for _, want := range []string{"市场行情报表", "加密货币", "美股 / ETF", "15 分钟延迟 SIP", "Hyperliquid 永续合约", "资金动向", "增仓", "CoinGecko 返回了部分数据"} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML missing %q", want)
		}
	}
	if strings.Contains(html, "净流入") {
		t.Fatal("report must not claim net inflow")
	}

	embed := gen.GenerateMarketDiscordEmbed(report)
	if embed.Title != "📈 市场行情报表" || len(embed.Fields) != 4 {
		t.Fatalf("embed=%#v", embed)
	}
	serialized := embed.Title + embed.Description
	for _, field := range embed.Fields {
		serialized += field.Name + field.Value
	}
	for _, want := range []string{"Bitcoin", "QQQ", "xyz:ZHIPU", "资金动向", "延迟 SIP"} {
		if !strings.Contains(serialized, want) {
			t.Errorf("Discord embed missing %q", want)
		}
	}
}

func TestMarketReportRendersUnavailableValuesAsNA(t *testing.T) {
	report := &MarketReport{
		GeneratedAt: time.Now(),
		Stocks:      StockSection{Configured: true, Feed: "delayed_sip", Items: []StockPrice{{Symbol: "SPCX", Price: 20, PriceTime: time.Now()}}},
		Perpetuals:  PerpetualSection{Configured: true, Items: []PerpetualPrice{{Symbol: "xyz:ZHIPU", MarkPrice: 100, ObservedAt: time.Now()}}},
	}
	html := NewReportGenerator().GenerateMarketHTML(report)
	if !strings.Contains(html, "N/A") || !strings.Contains(html, "较上次 N/A") {
		t.Fatalf("missing N/A rendering: %s", html)
	}
}
