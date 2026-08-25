package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAlpacaClientBatchesDelayedSnapshotsAndBars(t *testing.T) {
	now := time.Date(2026, 8, 25, 12, 5, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.Header.Get("APCA-API-KEY-ID") != "key" || r.Header.Get("APCA-API-SECRET-KEY") != "secret" {
			t.Error("Alpaca auth headers missing")
		}
		if r.URL.Query().Get("symbols") != "QQQ,SPCX" || r.URL.Query().Get("feed") != "sip" {
			t.Errorf("query = %s", r.URL.RawQuery)
		}
		if r.URL.Path != "/v2/stocks/bars" {
			http.NotFound(w, r)
			return
		}
		switch r.URL.Query().Get("timeframe") {
		case "1Min":
			fmt.Fprint(w, `{"bars":{
  "QQQ":[{"t":"2026-08-25T11:40:00Z","o":499,"c":500,"v":10,"vw":499.8}],
  "SPCX":[{"t":"2026-08-25T11:41:00Z","o":25,"c":25,"v":3,"vw":25}]
}}`)
		case "1Day":
			fmt.Fprint(w, `{"bars":{
  "QQQ":[{"t":"2026-08-24T04:00:00Z","o":480,"c":490,"v":100,"vw":485}],
  "SPCX":[{"t":"2026-08-24T04:00:00Z","o":25,"c":25,"v":10,"vw":25}]
}}`)
		case "1Hour":
			fmt.Fprint(w, `{"bars":{
  "QQQ":[{"t":"2026-08-25T11:00:00Z","o":499,"c":500,"v":999,"vw":499.5},{"t":"2026-08-25T10:00:00Z","o":495,"c":499,"v":1000,"vw":497}],
  "SPCX":[{"t":"2026-08-25T10:00:00Z","o":26,"c":25,"v":200,"vw":25.5}]
}}`)
		default:
			http.Error(w, "bad timeframe", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	client := NewAlpacaClient("key", "secret", "delayed_sip", false, "")
	client.baseURL = server.URL
	client.maxAttempts = 1
	prices, err := client.GetStockPrices([]string{"SPCX", "qqq"}, now)
	if err != nil {
		t.Fatalf("GetStockPrices: %v", err)
	}
	if len(prices) != 2 || prices[0].Symbol != "QQQ" {
		t.Fatalf("prices = %#v", prices)
	}
	qqq := prices[0]
	if qqq.DailyChangePercent == nil || *qqq.DailyChangePercent != 100.0/49.0 {
		t.Fatalf("QQQ daily change = %v", qqq.DailyChangePercent)
	}
	if qqq.HourlyTurnover == nil || *qqq.HourlyTurnover != 497000 {
		t.Fatalf("QQQ hourly turnover = %v", qqq.HourlyTurnover)
	}
	if qqq.HourlyBarStart == nil || qqq.HourlyBarStart.Hour() != 10 {
		t.Fatalf("应排除尚未完成的 11:00 bar: %v", qqq.HourlyBarStart)
	}
}

func TestAlpacaClientReturnsPartialDataForMissingSymbol(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("timeframe") {
		case "1Min":
			fmt.Fprint(w, `{"bars":{"QQQ":[{"t":"2026-08-25T10:00:00Z","o":500,"c":500,"v":10,"vw":500}]}}`)
		case "1Day":
			fmt.Fprint(w, `{"bars":{"QQQ":[{"t":"2026-08-24T04:00:00Z","o":490,"c":490,"v":10,"vw":490}]}}`)
		case "1Hour":
			fmt.Fprint(w, `{"bars":{"QQQ":[{"t":"2026-08-25T08:00:00Z","o":499,"c":500,"v":10,"vw":499.5}]}}`)
		}
	}))
	defer server.Close()
	client := NewAlpacaClient("key", "secret", "delayed_sip", false, "")
	client.baseURL = server.URL
	client.maxAttempts = 1
	prices, err := client.GetStockPrices([]string{"QQQ", "SPCX"}, time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC))
	if len(prices) != 1 || err == nil || !strings.Contains(err.Error(), "SPCX") {
		t.Fatalf("prices=%d err=%v", len(prices), err)
	}
}

func TestAlpacaClientRejectsNonSuccessAndMalformedJSON(t *testing.T) {
	t.Run("status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "denied", http.StatusForbidden) }))
		defer server.Close()
		client := NewAlpacaClient("key", "secret", "delayed_sip", false, "")
		client.baseURL, client.maxAttempts = server.URL, 1
		if _, err := client.GetStockPrices([]string{"QQQ"}, time.Now()); err == nil || !strings.Contains(err.Error(), "403") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{`) }))
		defer server.Close()
		client := NewAlpacaClient("key", "secret", "delayed_sip", false, "")
		client.baseURL, client.maxAttempts = server.URL, 1
		if _, err := client.GetStockPrices([]string{"QQQ"}, time.Now()); err == nil {
			t.Fatal("malformed JSON should fail")
		}
	})
}

func TestAlpacaClientFollowsBarPagination(t *testing.T) {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timeframe := r.URL.Query().Get("timeframe")
		if timeframe == "1Min" && r.URL.Query().Get("page_token") == "" {
			fmt.Fprint(w, `{"bars":{"QQQ":[{"t":"2026-08-25T11:40:00Z","c":500}]},"next_page_token":"next"}`)
			return
		}
		if timeframe == "1Min" {
			fmt.Fprint(w, `{"bars":{"SPCX":[{"t":"2026-08-25T11:40:00Z","c":25}]},"next_page_token":null}`)
			return
		}
		if timeframe == "1Day" {
			fmt.Fprint(w, `{"bars":{"QQQ":[{"t":"2026-08-24T04:00:00Z","c":490}],"SPCX":[{"t":"2026-08-24T04:00:00Z","c":24}]}}`)
			return
		}
		fmt.Fprint(w, `{"bars":{}}`)
	}))
	defer server.Close()
	client := NewAlpacaClient("key", "secret", "delayed_sip", false, "")
	client.baseURL, client.maxAttempts = server.URL, 1
	prices, err := client.GetStockPrices([]string{"QQQ", "SPCX"}, now)
	if len(prices) != 2 || err == nil || !strings.Contains(err.Error(), "completed 1h") {
		t.Fatalf("prices=%#v err=%v", prices, err)
	}
}
