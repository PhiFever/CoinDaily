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
	if embed.Title != "📈 市场行情报表" || len(embed.Fields) != 9 {
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
	for index, wants := range map[int][]string{
		1: {"> CoinGecko 返回了部分数据"},
		2: {"**$60000.00**", "🟢 **+1.20%**", "市值　", "成交　", "市值 Δ　", "<t:"},
		4: {"**$500.00**", "🟢 **+2.50%**", "1h 成交　", "VWAP　", "方向　🔴 下跌", "· 延迟"},
		6: {"**Mark $100.00**", "Oracle　", "溢折价　▲", "🟢 **+11.11%**", "<t:"},
		7: {"资金费率　▲", "名义 OI　", "OI Δ　▲"},
		8: {"**1h ", "24h　", "Candle　"},
	} {
		for _, want := range wants {
			if !strings.Contains(embed.Fields[index].Value, want) {
				t.Errorf("Discord field %d missing compact fragment %q:\n%s", index, want, embed.Fields[index].Value)
			}
		}
	}
	for _, index := range []int{0, 1, 3, 5} {
		if embed.Fields[index].Inline {
			t.Errorf("section/warning field %d should force a row break", index)
		}
	}
	for _, index := range []int{2, 4, 6, 7, 8} {
		if !embed.Fields[index].Inline {
			t.Errorf("market card %d should render inline", index)
		}
	}
}

func TestDiscordEmbedUsesHorizontalAssetCards(t *testing.T) {
	now := time.Now()
	report := &MarketReport{
		GeneratedAt: now,
		Crypto: CryptoSection{Configured: true, Items: []CoinPrice{
			{Name: "Bitcoin", Symbol: "btc", CurrentPrice: 1},
			{Name: "Ethereum", Symbol: "eth", CurrentPrice: 1},
			{Name: "Solana", Symbol: "sol", CurrentPrice: 1},
		}},
		Stocks: StockSection{Configured: true, Feed: "delayed_sip", Items: []StockPrice{
			{Symbol: "QQQ", Price: 1, PriceTime: now},
			{Symbol: "SPCX", Price: 1, PriceTime: now},
		}},
		Perpetuals: PerpetualSection{Configured: true, Items: []PerpetualPrice{{Symbol: "xyz:ZHIPU", MarkPrice: 1, ObservedAt: now}}},
	}
	embed := NewReportGenerator().GenerateMarketDiscordEmbed(report)
	if len(embed.Fields) != 11 {
		t.Fatalf("fields=%d, want 11", len(embed.Fields))
	}
	for _, index := range []int{0, 4, 7} {
		if embed.Fields[index].Inline {
			t.Errorf("section header %d should not be inline", index)
		}
	}
	for _, index := range []int{1, 2, 3, 5, 6, 8, 9, 10} {
		if !embed.Fields[index].Inline {
			t.Errorf("asset/detail card %d should be inline", index)
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
