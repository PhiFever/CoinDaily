package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeCryptoCollector struct {
	items []CoinPrice
	err   error
}

func (f fakeCryptoCollector) GetCoinPrices([]string) ([]CoinPrice, error) { return f.items, f.err }

type fakeStockCollector struct {
	items []StockPrice
	err   error
}

func (f fakeStockCollector) GetStockPrices([]string, time.Time) ([]StockPrice, error) {
	return f.items, f.err
}

type fakePerpCollector struct {
	items []PerpetualPrice
	err   error
}

func (f fakePerpCollector) GetPerpetualPrices([]string, time.Time) ([]PerpetualPrice, error) {
	return f.items, f.err
}

type fakeEmailSender struct {
	calls int
	html  string
	err   error
}

func (f *fakeEmailSender) SendReport(_ string, html string) error {
	f.calls++
	f.html = html
	return f.err
}

type fakeDiscordSender struct {
	calls  int
	report *MarketReport
	err    error
}

func (f *fakeDiscordSender) SendMarketReport(report *MarketReport) error {
	f.calls++
	f.report = report
	return f.err
}

func TestReportRunnerPartialFailureStateAndIndependentDelivery(t *testing.T) {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	stateStore := NewStateStore(filepath.Join(t.TempDir(), ".coindaily-state.json"))
	state := emptyLocalState()
	state.Perpetuals["xyz:ZHIPU"] = PerpetualSnapshot{Symbol: "xyz:ZHIPU", ObservedAt: now.Add(-time.Hour), NotionalOpenInterest: 100}
	if err := stateStore.Save(state); err != nil {
		t.Fatal(err)
	}

	email := &fakeEmailSender{err: errors.New("SMTP down")}
	discord := &fakeDiscordSender{}
	runner := &ReportRunner{
		CoinIDs:          []string{"bitcoin"},
		StockSymbols:     []string{"QQQ"},
		PerpetualSymbols: []string{"xyz:ZHIPU"},
		StockFeed:        "delayed_sip",
		CryptoCollector:  fakeCryptoCollector{err: errors.New("rate limited")},
		StockCollector: fakeStockCollector{items: []StockPrice{{
			Symbol: "QQQ", Price: 500, PriceTime: now.Add(-15 * time.Minute),
		}}},
		PerpCollector: fakePerpCollector{items: []PerpetualPrice{{
			Symbol: "xyz:ZHIPU", MarkPrice: 100, NotionalOpenInterest: floatPtr(130), ObservedAt: now,
		}}},
		StateStore:    stateStore,
		EmailSender:   email,
		DiscordSender: discord,
		Now:           func() time.Time { return now },
	}
	report, err := runner.Run()
	if err == nil || !strings.Contains(err.Error(), "email delivery") {
		t.Fatalf("delivery err = %v", err)
	}
	if !report.HasData() || !strings.Contains(report.Crypto.Warning, "CoinGecko") {
		t.Fatalf("report = %#v", report)
	}
	if email.calls != 1 || discord.calls != 1 {
		t.Fatalf("email calls=%d discord calls=%d", email.calls, discord.calls)
	}
	delta := report.Perpetuals.Items[0].OpenInterestChange
	if delta == nil || *delta != 30 {
		t.Fatalf("OI delta = %v", delta)
	}
	loaded, loadErr := stateStore.Load()
	if loadErr != nil || loaded.Perpetuals["xyz:ZHIPU"].NotionalOpenInterest != 130 {
		t.Fatalf("state=%#v err=%v", loaded, loadErr)
	}
	if !strings.Contains(email.html, "CoinGecko 获取失败") || !strings.Contains(email.html, "QQQ") || !strings.Contains(email.html, "xyz:ZHIPU") {
		t.Fatalf("partial HTML missing sections: %s", email.html)
	}
}

func TestReportRunnerAllSourcesFailSuppressesNotifications(t *testing.T) {
	email := &fakeEmailSender{}
	discord := &fakeDiscordSender{}
	runner := &ReportRunner{
		CoinIDs:          []string{"bitcoin"},
		StockSymbols:     []string{"QQQ"},
		PerpetualSymbols: []string{"xyz:ZHIPU"},
		CryptoCollector:  fakeCryptoCollector{err: errors.New("crypto down")},
		StockCollector:   fakeStockCollector{err: errors.New("stocks down")},
		PerpCollector:    fakePerpCollector{err: errors.New("perps down")},
		EmailSender:      email,
		DiscordSender:    discord,
	}
	report, err := runner.Run()
	if err == nil || report.HasData() {
		t.Fatalf("report=%#v err=%v", report, err)
	}
	if email.calls != 0 || discord.calls != 0 {
		t.Fatalf("empty report must not notify: email=%d discord=%d", email.calls, discord.calls)
	}
}

func TestReportRunnerStaleSnapshotProducesNoOIDelta(t *testing.T) {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	store := NewStateStore(filepath.Join(t.TempDir(), ".coindaily-state.json"))
	state := emptyLocalState()
	state.Perpetuals["xyz:ZHIPU"] = PerpetualSnapshot{Symbol: "xyz:ZHIPU", ObservedAt: now.Add(-91 * time.Minute), NotionalOpenInterest: 100}
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	runner := &ReportRunner{
		PerpetualSymbols: []string{"xyz:ZHIPU"},
		PerpCollector: fakePerpCollector{items: []PerpetualPrice{{
			Symbol: "xyz:ZHIPU", MarkPrice: 100, NotionalOpenInterest: floatPtr(110), ObservedAt: now,
		}}},
		StateStore: store,
		Now:        func() time.Time { return now },
	}
	report, err := runner.Run()
	if err != nil {
		t.Fatal(err)
	}
	if report.Perpetuals.Items[0].OpenInterestChange != nil {
		t.Fatalf("stale snapshot should yield N/A delta: %v", report.Perpetuals.Items[0].OpenInterestChange)
	}
}

func TestReportRunnerStateWriteFailureDoesNotSuppressReport(t *testing.T) {
	tempDir := t.TempDir()
	blockingFile := filepath.Join(tempDir, "not-a-directory")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	runner := &ReportRunner{
		PerpetualSymbols: []string{"xyz:ZHIPU"},
		PerpCollector: fakePerpCollector{items: []PerpetualPrice{{
			Symbol: "xyz:ZHIPU", MarkPrice: 1, NotionalOpenInterest: floatPtr(10), ObservedAt: now,
		}}},
		StateStore: NewStateStore(filepath.Join(blockingFile, ".coindaily-state.json")),
		Now:        func() time.Time { return now },
	}
	report, err := runner.Run()
	if err != nil || !report.HasData() || !strings.Contains(report.Perpetuals.Warning, "保存失败") {
		t.Fatalf("report=%#v err=%v", report, err)
	}
}
