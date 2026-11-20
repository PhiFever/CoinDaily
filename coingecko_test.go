package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCoinGeckoClientRetainsMarketCapChange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-cg-demo-api-key") != "key" {
			t.Error("CoinGecko API key header missing")
		}
		fmt.Fprint(w, `[{"id":"bitcoin","symbol":"btc","name":"Bitcoin","current_price":60000,"market_cap":1000000,"market_cap_change_24h":-12345,"market_cap_change_percentage_24h":-1.2,"price_change_percentage_24h":2,"total_volume":5000}]`)
	}))
	defer server.Close()
	client := NewCoinGeckoClient("key", false, "")
	client.baseURL = server.URL
	coins, err := client.GetCoinPrices([]string{"bitcoin"})
	if err != nil || len(coins) != 1 || coins[0].MarketCapChange24h == nil || *coins[0].MarketCapChange24h != -12345 {
		t.Fatalf("coins=%#v err=%v", coins, err)
	}
}

func TestCanaryMarketCapsUseLastUTCObservation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.Method != "GET" || r.URL.Path != "/coins/tether/market_chart" || q.Get("days") != "31" || q.Get("interval") != "daily" || q.Get("vs_currency") != "usd" || r.Header.Get("x-cg-demo-api-key") != "key" {
			t.Errorf("unexpected request %s", r.URL)
		}
		fmt.Fprint(w, `{"market_caps":[[0,100],[86400000,102],[140400000,105]]}`)
	}))
	defer server.Close()
	c := NewCoinGeckoClient("key", false, "")
	c.baseURL = server.URL
	items, err := c.GetMarketCaps([]string{"tether"})
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%v err=%v", items, err)
	}
	p := items[0].Points
	if len(p) != 2 || p[0].Value != 100 || p[1].Date.UnixMilli() != 86400000 || p[1].Value != 105 {
		t.Fatalf("points=%v", p)
	}
}
