package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type CoinPrice struct {
	ID                string  `json:"id"`
	Symbol            string  `json:"symbol"`
	Name              string  `json:"name"`
	CurrentPrice      float64 `json:"current_price"`
	MarketCap         float64 `json:"market_cap"`
	PriceChange24h    float64 `json:"price_change_24h"`
	PriceChangePerc24h float64 `json:"price_change_percentage_24h"`
	Volume24h         float64 `json:"total_volume"`
	LastUpdated       string  `json:"last_updated"`
}

type CoinGeckoClient struct {
	baseURL string
	client  *http.Client
}

func NewCoinGeckoClient() *CoinGeckoClient {
	return &CoinGeckoClient{
		baseURL: "https://api.coingecko.com/api/v3",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *CoinGeckoClient) GetCoinPrices(coinIDs []string) ([]CoinPrice, error) {
	idsParam := strings.Join(coinIDs, ",")
	url := fmt.Sprintf("%s/coins/markets?vs_currency=usd&ids=%s&order=market_cap_desc&per_page=100&page=1&sparkline=false", 
		c.baseURL, idsParam)
	
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data from CoinGecko: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CoinGecko API returned status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var coins []CoinPrice
	if err := json.Unmarshal(body, &coins); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return coins, nil
}