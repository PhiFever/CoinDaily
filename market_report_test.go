package main

import (
	"fmt"
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

	embed := gen.GenerateMarketDiscordEmbeds(report)[0]
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
	embed := NewReportGenerator().GenerateMarketDiscordEmbeds(report)[0]
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

func TestCanaryReportRendering(t *testing.T) {
	report := largeDiscordReport()
	report.Crypto = CryptoSection{}
	report.Stocks = StockSection{}
	report.Perpetuals = PerpetualSection{}
	gen := NewReportGenerator()
	html := gen.GenerateMarketHTML(report)
	embed := gen.GenerateMarketDiscordEmbeds(report)[0]
	var fields strings.Builder
	for _, field := range embed.Fields {
		if field.Inline {
			t.Errorf("unexpected inline field %q", field.Name)
		}
		fields.WriteString(field.Name + " " + field.Value + "\n")
	}
	for _, output := range []string{html, fields.String()} {
		for _, want := range []string{"流动性金丝雀", "USDT 供应", "TLT 长债", "SOFR (%)", "SOFR − IORB", "7d", "30d", "▁█", "+2 bp", "11-18"} {
			if !strings.Contains(output, want) {
				t.Errorf("missing %q", want)
			}
		}
		if strings.Contains(output, "资金面") {
			t.Error("obsolete FRED section")
		}
	}
	if embed.Footer == nil || embed.Footer.Text != "CoinGecko · Alpaca · FRED | CoinDaily" {
		t.Fatalf("footer=%+v", embed.Footer)
	}
}

func largeDiscordReport() *MarketReport {
	date := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
	report := &MarketReport{
		GeneratedAt: date,
		Crypto:      CryptoSection{Configured: true},
		Stocks:      StockSection{Configured: true, Feed: "delayed_sip"},
		Perpetuals:  PerpetualSection{Configured: true, Items: []PerpetualPrice{{Symbol: "xyz:ZHIPU", ObservedAt: date}}},
		Canary:      CanarySection{Configured: true, sources: map[string]bool{"CoinGecko": true, "Alpaca": true, "FRED": true}},
	}
	for i := range 10 {
		report.Crypto.Items = append(report.Crypto.Items, CoinPrice{Name: fmt.Sprintf("Coin %d", i), Symbol: fmt.Sprintf("c%d", i)})
	}
	for i := range 7 {
		report.Stocks.Items = append(report.Stocks.Items, StockPrice{Symbol: fmt.Sprintf("S%d", i), PriceTime: date})
	}
	for _, id := range []string{"tether", "usd-coin", "TLT", "GLD", "UUP", "HYG"} {
		group := etfGroup
		if id == "tether" || id == "usd-coin" {
			group = stablecoinGroup
		}
		report.Canary.Items = append(report.Canary.Items, marketCanary(group, id, []CanaryPoint{{Date: date, Value: 100}}))
	}
	var fred []FredSeries
	for _, id := range []string{"BAMLH0A0HYM2", "VIXCLS", "SOFR", "IORB"} {
		fred = append(fred, FredSeries{ID: id, Observations: []FredObservation{{Date: date, Value: 3.12}, {Date: date.AddDate(0, 0, -1), Value: 3.10}}})
	}
	report.Canary.Items = append(report.Canary.Items, fredCanaries(fred)...)
	return report
}

func TestLargeMarketDiscordEmbeds(t *testing.T) {
	report := largeDiscordReport()
	embeds := NewReportGenerator().GenerateMarketDiscordEmbeds(report)
	if len(embeds) != 2 || len(embeds[0].Fields) != 23 || len(embeds[1].Fields) != 12 {
		t.Fatalf("expected the canary section on page two")
	}
	first, last := embeds[0], embeds[1]
	if first.Description == "" || first.Footer != nil || first.Timestamp != "" ||
		last.Title != "📈 市场行情报表（续 2）" || last.Description != "" || last.Footer == nil || last.Timestamp != report.GeneratedAt.Format(time.RFC3339) {
		t.Fatal("unexpected metadata on split report")
	}
	seen := map[string]int{}
	total := 0
	for i, embed := range embeds {
		if len(embed.Fields) > maxEmbedFields || calculateEmbedLength(embed) > maxEmbedTotalLength {
			t.Fatalf("embed %d exceeds limits", i+1)
		}
		for j, field := range embed.Fields {
			total++
			seen[field.Name]++
			if strings.Contains(field.Value, "更多标的已省略") {
				t.Errorf("embed %d truncated %s", i+1, field.Name)
			}
			if field.Name == "📈 xyz:ZHIPU · 行情" && (j+2 >= len(embed.Fields) || embed.Fields[j+1].Name != "💰 持仓 / 资金" || embed.Fields[j+2].Name != "🔥 成交活跃度") {
				t.Error("split perpetual fields")
			}
		}
	}
	if total != 35 {
		t.Errorf("got %d fields, want 35", total)
	}
	for _, name := range []string{"USDT 供应", "USDC 供应", "TLT 长债", "GLD 黄金", "UUP 美元", "HYG 高收益债", "高收益债利差 OAS (%)", "VIX", "SOFR (%)", "IORB (%)", "SOFR − IORB", "📈 xyz:ZHIPU · 行情"} {
		if seen[name] != 1 {
			t.Errorf("field %s occurred %d times", name, seen[name])
		}
	}
}
