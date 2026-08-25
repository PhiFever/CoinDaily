package main

import (
	"fmt"
	"html"
	"math"
	"strings"
	"time"
)

func (r *ReportGenerator) GenerateMarketHTML(report *MarketReport) string {
	var body strings.Builder
	generated := report.GeneratedAt.Format("2006年01月02日 15:04 MST")
	body.WriteString(`<!DOCTYPE html><html><head><meta charset="UTF-8"><title>市场行情报表</title><style>
body{font-family:Arial,sans-serif;margin:20px;background:#f5f5f5;color:#2c3e50}.header,.section,.footer{background:#fff;border-radius:8px;padding:18px;margin-bottom:18px;box-shadow:0 2px 10px rgba(0,0,0,.08)}.header{text-align:center}table{width:100%;border-collapse:collapse}th,td{padding:10px;text-align:left;border-bottom:1px solid #ecf0f1}th{background:#34495e;color:#fff}.positive{color:#18864b;font-weight:bold}.negative{color:#c0392b;font-weight:bold}.warning{padding:10px;background:#fff3cd;color:#765c00;border-radius:5px}.meta{color:#6c757d;font-size:13px}.footer{text-align:center;color:#6c757d;font-size:13px}</style></head><body>`)
	fmt.Fprintf(&body, `<div class="header"><h1>📈 市场行情报表</h1><div class="meta">生成时间：%s</div></div>`, html.EscapeString(generated))

	if report.Crypto.Configured {
		body.WriteString(`<div class="section"><h2>加密货币</h2><div class="meta">数据来源：CoinGecko API</div>`)
		writeHTMLWarning(&body, report.Crypto.Warning)
		if len(report.Crypto.Items) > 0 {
			body.WriteString(`<table><thead><tr><th>币种</th><th>价格</th><th>24h</th><th>市值</th><th>资金动向：24h成交量 / 市值变化</th><th>更新时间</th></tr></thead><tbody>`)
			for _, coin := range report.Crypto.Items {
				fmt.Fprintf(&body, `<tr><td><strong>%s</strong> (%s)</td><td>$%s</td><td class="%s">%s</td><td>$%s</td><td>$%s / <span class="%s">%s</span></td><td>%s</td></tr>`,
					html.EscapeString(coin.Name), html.EscapeString(strings.ToUpper(coin.Symbol)), formatMarketNumber(coin.CurrentPrice), valueClass(coin.PriceChangePerc24h), formatSignedPercent(coin.PriceChangePerc24h), formatMarketLarge(coin.MarketCap), formatMarketLarge(coin.Volume24h), optionalClass(coin.MarketCapChange24h), formatOptionalSignedMoney(coin.MarketCapChange24h), html.EscapeString(formatSourceTime(coin.LastUpdated, report.GeneratedAt.Location())))
			}
			body.WriteString(`</tbody></table>`)
		}
		body.WriteString(`</div>`)
	}

	if report.Stocks.Configured {
		body.WriteString(`<div class="section"><h2>美股 / ETF</h2>`)
		fmt.Fprintf(&body, `<div class="meta">数据来源：Alpaca · %s（15 分钟延迟 SIP，非实时可成交报价）</div>`, html.EscapeString(report.Stocks.Feed))
		writeHTMLWarning(&body, report.Stocks.Warning)
		if len(report.Stocks.Items) > 0 {
			body.WriteString(`<table><thead><tr><th>标的</th><th>最新价格</th><th>相对前收</th><th>资金动向：完整1h估算成交额</th><th>1h VWAP / 方向</th><th>行情时间</th></tr></thead><tbody>`)
			for _, stock := range report.Stocks.Items {
				fmt.Fprintf(&body, `<tr><td><strong>%s</strong></td><td>$%s</td><td class="%s">%s</td><td>%s<br><span class="meta">bar: %s</span></td><td>%s / %s</td><td>%s</td></tr>`,
					html.EscapeString(stock.Symbol), formatMarketNumber(stock.Price), optionalClass(stock.DailyChangePercent), formatOptionalPercent(stock.DailyChangePercent), formatOptionalMoney(stock.HourlyTurnover), formatOptionalTime(stock.HourlyBarStart, report.GeneratedAt.Location()), formatOptionalPrice(stock.HourlyVWAP), formatDirection(stock.HourlyDirection), html.EscapeString(stock.PriceTime.In(report.GeneratedAt.Location()).Format("01-02 15:04:05 MST")))
			}
			body.WriteString(`</tbody></table>`)
		}
		body.WriteString(`</div>`)
	}

	if report.Perpetuals.Configured {
		body.WriteString(`<div class="section"><h2>Hyperliquid 永续合约</h2><div class="meta">数据来源：Hyperliquid 公开 Info API · 标记价格为主</div>`)
		writeHTMLWarning(&body, report.Perpetuals.Warning)
		if len(report.Perpetuals.Items) > 0 {
			body.WriteString(`<table><thead><tr><th>合约</th><th>标记 / 预言机</th><th>溢折价 / 24h</th><th>每小时资金费率</th><th>资金动向：名义OI / 较上次</th><th>完整1h估算成交额 / 24h成交额</th><th>观测时间</th></tr></thead><tbody>`)
			for _, perp := range report.Perpetuals.Items {
				fmt.Fprintf(&body, `<tr><td><strong>%s</strong></td><td>$%s / %s</td><td class="%s">%s / %s</td><td class="%s">%s</td><td>%s / %s</td><td>%s / %s<br><span class="meta">bar: %s</span></td><td>%s</td></tr>`,
					html.EscapeString(perp.Symbol), formatMarketNumber(perp.MarkPrice), formatOptionalPrice(perp.OraclePrice), optionalClass(perp.MarkOracleSpreadPct), formatSpread(perp.MarkOracleSpread, perp.MarkOracleSpreadPct), formatOptionalPercent(perp.PriceChange24hPct), optionalClass(perp.FundingRate), formatFunding(perp.FundingRate), formatOptionalMoney(perp.NotionalOpenInterest), formatOIDelta(perp.OpenInterestChange), formatOptionalMoney(perp.HourlyTurnover), formatOptionalMoney(perp.DayTurnover), formatOptionalTime(perp.HourlyCandleStart, report.GeneratedAt.Location()), html.EscapeString(perp.ObservedAt.In(report.GeneratedAt.Location()).Format("01-02 15:04:05 MST")))
			}
			body.WriteString(`</tbody></table>`)
		}
		body.WriteString(`</div>`)
	}

	body.WriteString(`<div class="footer">“资金动向”是成交、市值、持仓与资金费率代理指标，不代表经审计的资金流。<br>此报表由 CoinDaily 自动生成</div></body></html>`)
	return body.String()
}

func (r *ReportGenerator) GenerateMarketDiscordEmbed(report *MarketReport) *DiscordEmbed {
	fields := make([]EmbedField, 0, len(report.Crypto.Items)+len(report.Stocks.Items)+len(report.Perpetuals.Items)+3)
	if report.Crypto.Configured {
		if report.Crypto.Warning != "" {
			fields = append(fields, warningField("加密货币", report.Crypto.Warning))
		}
		for _, coin := range report.Crypto.Items {
			fields = append(fields, EmbedField{
				Name: fmt.Sprintf("加密货币 · %s (%s)", coin.Name, strings.ToUpper(coin.Symbol)),
				Value: fmt.Sprintf("**$%s** · 24h %s\n资金动向：成交量 $%s · 市值变化 %s\n更新：%s",
					formatMarketNumber(coin.CurrentPrice), formatSignedPercent(coin.PriceChangePerc24h), formatMarketLarge(coin.Volume24h), formatOptionalSignedMoney(coin.MarketCapChange24h), formatSourceTime(coin.LastUpdated, report.GeneratedAt.Location())),
				Inline: false,
			})
		}
	}
	if report.Stocks.Configured {
		if report.Stocks.Warning != "" {
			fields = append(fields, warningField("美股 / ETF", report.Stocks.Warning))
		}
		for _, stock := range report.Stocks.Items {
			fields = append(fields, EmbedField{
				Name: fmt.Sprintf("美股 / ETF · %s", stock.Symbol),
				Value: fmt.Sprintf("**$%s** · 相对前收 %s\n资金动向：完整1h成交额 %s · VWAP %s · %s\n延迟 SIP 行情：%s",
					formatMarketNumber(stock.Price), formatOptionalPercent(stock.DailyChangePercent), formatOptionalMoney(stock.HourlyTurnover), formatOptionalPrice(stock.HourlyVWAP), formatDirection(stock.HourlyDirection)+" · bar "+formatOptionalTime(stock.HourlyBarStart, report.GeneratedAt.Location()), stock.PriceTime.In(report.GeneratedAt.Location()).Format("01-02 15:04:05 MST")),
				Inline: false,
			})
		}
	}
	if report.Perpetuals.Configured {
		if report.Perpetuals.Warning != "" {
			fields = append(fields, warningField("Hyperliquid 永续", report.Perpetuals.Warning))
		}
		for _, perp := range report.Perpetuals.Items {
			fields = append(fields, EmbedField{
				Name: fmt.Sprintf("Hyperliquid 永续 · %s", perp.Symbol),
				Value: fmt.Sprintf("**Mark $%s** · Oracle %s · %s\n24h %s · 每小时资金费率 %s\n资金动向：名义 OI %s · %s · 完整1h成交额 %s\n观测：%s",
					formatMarketNumber(perp.MarkPrice), formatOptionalPrice(perp.OraclePrice), formatSpread(perp.MarkOracleSpread, perp.MarkOracleSpreadPct), formatOptionalPercent(perp.PriceChange24hPct), formatFunding(perp.FundingRate), formatOptionalMoney(perp.NotionalOpenInterest), formatOIDelta(perp.OpenInterestChange), formatOptionalMoney(perp.HourlyTurnover)+" · 24h "+formatOptionalMoney(perp.DayTurnover)+" · bar "+formatOptionalTime(perp.HourlyCandleStart, report.GeneratedAt.Location()), perp.ObservedAt.In(report.GeneratedAt.Location()).Format("01-02 15:04:05 MST")),
				Inline: false,
			})
		}
	}

	embed := &DiscordEmbed{
		Title:       "📈 市场行情报表",
		Description: report.GeneratedAt.Format("2006年01月02日 15:04 MST") + "\n资金动向为成交/市值/持仓代理指标。",
		Color:       marketReportColor(report),
		Fields:      fields,
		Footer:      &EmbedFooter{Text: strings.Join(configuredSourceNames(report), " · ") + " | CoinDaily"},
		Timestamp:   report.GeneratedAt.Format(time.RFC3339),
	}
	return truncateEmbedIfNeeded(embed)
}

func writeHTMLWarning(body *strings.Builder, warning string) {
	if warning != "" {
		fmt.Fprintf(body, `<p class="warning">⚠ %s</p>`, html.EscapeString(warning))
	}
}

func warningField(section, warning string) EmbedField {
	return EmbedField{Name: "⚠ " + section + "数据警告", Value: warning, Inline: false}
}

func formatMarketNumber(value float64) string {
	abs := math.Abs(value)
	switch {
	case abs >= 1:
		return fmt.Sprintf("%.2f", value)
	case abs >= 0.01:
		return fmt.Sprintf("%.4f", value)
	default:
		return fmt.Sprintf("%.8f", value)
	}
}

func formatMarketLarge(value float64) string {
	abs := math.Abs(value)
	sign := ""
	if value < 0 {
		sign = "-"
	}
	switch {
	case abs >= 1e12:
		return fmt.Sprintf("%s%.2fT", sign, abs/1e12)
	case abs >= 1e9:
		return fmt.Sprintf("%s%.2fB", sign, abs/1e9)
	case abs >= 1e6:
		return fmt.Sprintf("%s%.2fM", sign, abs/1e6)
	case abs >= 1e3:
		return fmt.Sprintf("%s%.2fK", sign, abs/1e3)
	default:
		return fmt.Sprintf("%s%.2f", sign, abs)
	}
}

func formatSignedPercent(value float64) string { return fmt.Sprintf("%+.2f%%", value) }

func formatSignedMoney(value float64) string {
	if value < 0 {
		return "-$" + formatMarketLarge(-value)
	}
	return "+$" + formatMarketLarge(value)
}

func formatOptionalSignedMoney(value *float64) string {
	if value == nil {
		return "N/A"
	}
	return formatSignedMoney(*value)
}

func formatOptionalPercent(value *float64) string {
	if value == nil {
		return "N/A"
	}
	return formatSignedPercent(*value)
}

func formatOptionalMoney(value *float64) string {
	if value == nil {
		return "N/A"
	}
	return "$" + formatMarketLarge(*value)
}

func formatOptionalPrice(value *float64) string {
	if value == nil {
		return "N/A"
	}
	return "$" + formatMarketNumber(*value)
}

func formatOptionalTime(value *time.Time, location *time.Location) string {
	if value == nil {
		return "N/A"
	}
	return value.In(location).Format("01-02 15:04 MST")
}

func formatFunding(value *float64) string {
	if value == nil {
		return "N/A"
	}
	return fmt.Sprintf("%+.6f%%", *value*100)
}

func formatSpread(absolute, percent *float64) string {
	if absolute == nil || percent == nil {
		return "溢折价 N/A"
	}
	if *percent >= 0 {
		return fmt.Sprintf("溢价 +$%s (+%.3f%%)", formatMarketNumber(*absolute), *percent)
	}
	return fmt.Sprintf("折价 -$%s (%.3f%%)", formatMarketNumber(math.Abs(*absolute)), *percent)
}

func formatOIDelta(value *float64) string {
	if value == nil {
		return "较上次 N/A"
	}
	if *value >= 0 {
		return "增仓 +$" + formatMarketLarge(*value)
	}
	return "减仓 -$" + formatMarketLarge(-*value)
}

func formatDirection(value *float64) string {
	if value == nil {
		return "方向 N/A"
	}
	if *value > 0 {
		return "小时上涨"
	}
	if *value < 0 {
		return "小时下跌"
	}
	return "小时持平"
}

func formatSourceTime(raw string, location *time.Location) string {
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return "N/A"
	}
	return parsed.In(location).Format("01-02 15:04:05 MST")
}

func valueClass(value float64) string {
	if value < 0 {
		return "negative"
	}
	return "positive"
}

func optionalClass(value *float64) string {
	if value != nil && *value < 0 {
		return "negative"
	}
	return "positive"
}

func marketReportColor(report *MarketReport) int {
	total := 0.0
	count := 0
	for _, coin := range report.Crypto.Items {
		total += coin.PriceChangePerc24h
		count++
	}
	for _, stock := range report.Stocks.Items {
		if stock.DailyChangePercent != nil {
			total += *stock.DailyChangePercent
			count++
		}
	}
	for _, perp := range report.Perpetuals.Items {
		if perp.PriceChange24hPct != nil {
			total += *perp.PriceChange24hPct
			count++
		}
	}
	if count == 0 {
		return 0xFFD700
	}
	if total >= 0 {
		return 0x27AE60
	}
	return 0xE74C3C
}

func configuredSourceNames(report *MarketReport) []string {
	sources := make([]string, 0, 3)
	if report.Crypto.Configured {
		sources = append(sources, "CoinGecko")
	}
	if report.Stocks.Configured {
		sources = append(sources, "Alpaca "+report.Stocks.Feed)
	}
	if report.Perpetuals.Configured {
		sources = append(sources, "Hyperliquid")
	}
	return sources
}
