package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHyperliquidClientMatchesUniverseAndCalculatesPerpMetrics(t *testing.T) {
	now := time.Date(2026, 8, 25, 12, 5, 0, 0, time.UTC)
	completedStart := now.Truncate(time.Hour).Add(-time.Hour).UnixMilli()
	completedEnd := now.Truncate(time.Hour).Add(-time.Millisecond).UnixMilli()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/info" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var request struct {
			Type string `json:"type"`
			DEX  string `json:"dex"`
			Req  struct {
				Coin     string `json:"coin"`
				Interval string `json:"interval"`
			} `json:"req"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		switch request.Type {
		case "metaAndAssetCtxs":
			if request.DEX != "xyz" {
				t.Errorf("dex = %q, want xyz", request.DEX)
			}
			fmt.Fprint(w, `[
 {"universe":[{"name":"xyz:OTHER"},{"name":"xyz:ZHIPU"}]},
 [
  {"markPx":"1","oraclePx":"1","prevDayPx":"1","funding":"0","openInterest":"1","dayNtlVlm":"1"},
  {"markPx":"100","oraclePx":"99","prevDayPx":"90","funding":"0.0001","openInterest":"5","dayNtlVlm":"1234"}
 ]
]`)
		case "candleSnapshot":
			if request.Req.Coin != "xyz:ZHIPU" || request.Req.Interval != "1h" {
				t.Errorf("candle request = %#v", request.Req)
			}
			fmt.Fprintf(w, `[
 {"t":%d,"T":%d,"s":"xyz:ZHIPU","i":"1h","o":"99","h":"102","l":"98","c":"101","v":"2"},
 {"t":%d,"T":%d,"s":"xyz:ZHIPU","i":"1h","o":"101","h":"103","l":"100","c":"102","v":"3"}
]`, completedStart, completedEnd, now.Truncate(time.Hour).UnixMilli(), now.Truncate(time.Hour).Add(time.Hour-time.Millisecond).UnixMilli())
		default:
			http.Error(w, "bad type", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	client := NewHyperliquidClient(false, "")
	client.baseURL, client.maxAttempts = server.URL, 1
	prices, err := client.GetPerpetualPrices([]string{"xyz:ZHIPU"}, now)
	if err != nil {
		t.Fatalf("GetPerpetualPrices: %v", err)
	}
	if len(prices) != 1 {
		t.Fatalf("prices = %#v", prices)
	}
	perp := prices[0]
	if perp.MarkPrice != 100 || perp.NotionalOpenInterest == nil || *perp.NotionalOpenInterest != 500 {
		t.Fatalf("perp = %#v", perp)
	}
	if perp.MarkOracleSpreadPct == nil || math.Abs(*perp.MarkOracleSpreadPct-100.0/99.0) > 1e-9 {
		t.Fatalf("spread = %v", perp.MarkOracleSpreadPct)
	}
	if perp.PriceChange24hPct == nil || math.Abs(*perp.PriceChange24hPct-100.0/9.0) > 1e-9 {
		t.Fatalf("24h = %v", perp.PriceChange24hPct)
	}
	if perp.FundingRate == nil || *perp.FundingRate != 0.0001 {
		t.Fatalf("funding = %v", perp.FundingRate)
	}
	if perp.HourlyTurnover == nil || *perp.HourlyTurnover != 200 {
		t.Fatalf("turnover = %v", perp.HourlyTurnover)
	}
}

func TestHyperliquidClientHandlesAbsentAndMalformedContract(t *testing.T) {
	tests := []struct {
		name     string
		context  string
		wantText string
	}{
		{name: "absent", context: `[{"universe":[{"name":"xyz:OTHER"}]},[{"markPx":"1"}]]`, wantText: "absent"},
		{name: "malformed", context: `[{"universe":[{"name":"xyz:ZHIPU"}]},[{"markPx":"not-a-number"}]]`, wantText: "markPx"},
		{name: "null mark", context: `[{"universe":[{"name":"xyz:ZHIPU"}]},[{"markPx":null}]]`, wantText: "markPx"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, tt.context) }))
			defer server.Close()
			client := NewHyperliquidClient(false, "")
			client.baseURL, client.maxAttempts = server.URL, 1
			prices, err := client.GetPerpetualPrices([]string{"xyz:ZHIPU"}, time.Now())
			if len(prices) != 0 || err == nil || !strings.Contains(err.Error(), tt.wantText) {
				t.Fatalf("prices=%v err=%v", prices, err)
			}
		})
	}
}

func TestHyperliquidClientKeepsContextWhenCandleFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		_ = json.NewDecoder(r.Body).Decode(&request)
		if request["type"] == "metaAndAssetCtxs" {
			fmt.Fprint(w, `[{"universe":[{"name":"xyz:ZHIPU"}]},[{"markPx":"100","oraclePx":null,"prevDayPx":"0","funding":null,"openInterest":"0","dayNtlVlm":null}]]`)
			return
		}
		http.Error(w, "temporary", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := NewHyperliquidClient(false, "")
	client.baseURL, client.maxAttempts = server.URL, 1
	prices, err := client.GetPerpetualPrices([]string{"xyz:ZHIPU"}, time.Now())
	if len(prices) != 1 || err == nil || prices[0].OraclePrice != nil || prices[0].PriceChange24hPct != nil {
		t.Fatalf("prices=%#v err=%v", prices, err)
	}
}
