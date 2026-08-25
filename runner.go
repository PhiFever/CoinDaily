package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

type CryptoCollector interface {
	GetCoinPrices(coinIDs []string) ([]CoinPrice, error)
}

type StockCollector interface {
	GetStockPrices(symbols []string, now time.Time) ([]StockPrice, error)
}

type PerpetualCollector interface {
	GetPerpetualPrices(symbols []string, now time.Time) ([]PerpetualPrice, error)
}

type EmailReportSender interface {
	SendReport(subject, htmlContent string) error
}

type DiscordReportSender interface {
	SendMarketReport(report *MarketReport) error
}

type ReportRunner struct {
	CoinIDs          []string
	StockSymbols     []string
	PerpetualSymbols []string
	StockFeed        string
	CryptoCollector  CryptoCollector
	StockCollector   StockCollector
	PerpCollector    PerpetualCollector
	StateStore       *StateStore
	ReportGenerator  *ReportGenerator
	EmailSender      EmailReportSender
	DiscordSender    DiscordReportSender
	Now              func() time.Time
}

func (r *ReportRunner) Run() (*MarketReport, error) {
	now := time.Now()
	if r.Now != nil {
		now = r.Now()
	}
	report := &MarketReport{GeneratedAt: now}
	var sourceErrors []error

	if len(r.CoinIDs) > 0 {
		report.Crypto.Configured = true
		if r.CryptoCollector == nil {
			err := fmt.Errorf("CoinGecko collector is unavailable")
			report.Crypto.Warning = err.Error()
			sourceErrors = append(sourceErrors, err)
		} else {
			items, err := r.CryptoCollector.GetCoinPrices(r.CoinIDs)
			report.Crypto.Items = items
			report.Crypto.Warning = sectionWarning("CoinGecko", len(items), err)
			if err != nil || len(items) == 0 {
				sourceErrors = append(sourceErrors, sourceFailure("CoinGecko", len(items), err))
			}
		}
	}

	if len(r.StockSymbols) > 0 {
		report.Stocks.Configured = true
		report.Stocks.Feed = r.StockFeed
		if r.StockCollector == nil {
			err := fmt.Errorf("Alpaca collector is unavailable")
			report.Stocks.Warning = err.Error()
			sourceErrors = append(sourceErrors, err)
		} else {
			items, err := r.StockCollector.GetStockPrices(r.StockSymbols, now)
			report.Stocks.Items = items
			report.Stocks.Warning = sectionWarning("Alpaca", len(items), err)
			if err != nil || len(items) == 0 {
				sourceErrors = append(sourceErrors, sourceFailure("Alpaca", len(items), err))
			}
		}
	}

	if len(r.PerpetualSymbols) > 0 {
		report.Perpetuals.Configured = true
		if r.PerpCollector == nil {
			err := fmt.Errorf("Hyperliquid collector is unavailable")
			report.Perpetuals.Warning = err.Error()
			sourceErrors = append(sourceErrors, err)
		} else {
			items, err := r.PerpCollector.GetPerpetualPrices(r.PerpetualSymbols, now)
			report.Perpetuals.Items = items
			report.Perpetuals.Warning = sectionWarning("Hyperliquid", len(items), err)
			if err != nil || len(items) == 0 {
				sourceErrors = append(sourceErrors, sourceFailure("Hyperliquid", len(items), err))
			}
			if len(items) > 0 {
				r.applyPerpetualState(report)
			}
		}
	}

	for _, err := range sourceErrors {
		log.Printf("行情源异常: %v", err)
	}
	if !report.HasData() {
		return report, fmt.Errorf("all configured market data sources failed or returned no usable data: %w", errors.Join(sourceErrors...))
	}

	generator := r.ReportGenerator
	if generator == nil {
		generator = NewReportGenerator()
	}
	var deliveryErrors []error
	if r.EmailSender != nil {
		subject := fmt.Sprintf("市场行情报表 - %s", now.Format("2006年01月02日 15:04"))
		if err := r.EmailSender.SendReport(subject, generator.GenerateMarketHTML(report)); err != nil {
			deliveryErrors = append(deliveryErrors, fmt.Errorf("email delivery: %w", err))
		}
	}
	if r.DiscordSender != nil {
		if err := r.DiscordSender.SendMarketReport(report); err != nil {
			deliveryErrors = append(deliveryErrors, fmt.Errorf("Discord delivery: %w", err))
		}
	}
	return report, errors.Join(deliveryErrors...)
}

func (r *ReportRunner) applyPerpetualState(report *MarketReport) {
	if r.StateStore == nil {
		return
	}
	state, err := r.StateStore.Load()
	if err != nil {
		log.Printf("读取本地持仓快照失败，将建立新基准: %v", err)
		state = emptyLocalState()
	}
	updated := false
	for index := range report.Perpetuals.Items {
		item := &report.Perpetuals.Items[index]
		if item.NotionalOpenInterest == nil || !isFinite(*item.NotionalOpenInterest) || *item.NotionalOpenInterest < 0 {
			continue
		}
		if previous, ok := state.Perpetuals[item.Symbol]; ok && previous.Symbol == item.Symbol {
			age := item.ObservedAt.Sub(previous.ObservedAt)
			if age >= 0 && age <= 90*time.Minute {
				item.OpenInterestChange = floatPtr(*item.NotionalOpenInterest - previous.NotionalOpenInterest)
			}
		}
		state.Perpetuals[item.Symbol] = PerpetualSnapshot{
			Symbol:               item.Symbol,
			ObservedAt:           item.ObservedAt,
			NotionalOpenInterest: *item.NotionalOpenInterest,
		}
		updated = true
	}
	if updated {
		if err := r.StateStore.Save(state); err != nil {
			log.Printf("保存本地持仓快照失败: %v", err)
			report.Perpetuals.Warning = appendWarning(report.Perpetuals.Warning, "本地持仓快照保存失败，下次持仓变化可能为 N/A")
		}
	}
}

func sectionWarning(provider string, itemCount int, err error) string {
	if err != nil {
		if itemCount > 0 {
			return fmt.Sprintf("%s 返回了部分数据：%v", provider, err)
		}
		return fmt.Sprintf("%s 获取失败：%v", provider, err)
	}
	if itemCount == 0 {
		return fmt.Sprintf("%s 未返回可用数据", provider)
	}
	return ""
}

func sourceFailure(provider string, itemCount int, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", provider, err)
	}
	if itemCount == 0 {
		return fmt.Errorf("%s: no usable observations", provider)
	}
	return nil
}

func appendWarning(current, addition string) string {
	if current == "" {
		return addition
	}
	return strings.TrimSpace(current) + "；" + addition
}
