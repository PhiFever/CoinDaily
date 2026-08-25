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
	coins, err := client.doRequest(server.URL + "/coins/markets")
	if err != nil || len(coins) != 1 || coins[0].MarketCapChange24h == nil || *coins[0].MarketCapChange24h != -12345 {
		t.Fatalf("coins=%#v err=%v", coins, err)
	}
}
