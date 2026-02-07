package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	apiKey  string
	client  *http.Client
}

func NewCoinGeckoClient(apiKey string, proxyEnabled bool, proxyURL string) *CoinGeckoClient {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 如果启用了代理，配置 HTTP Transport
	if proxyEnabled && proxyURL != "" {
		parsedProxyURL, err := url.Parse(proxyURL)
		if err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(parsedProxyURL),
			}
		}
	}

	return &CoinGeckoClient{
		baseURL: "https://api.coingecko.com/api/v3",
		apiKey:  apiKey,
		client:  client,
	}
}

const (
	maxRetries    = 3
	retryInterval = 10 * time.Second
)

func (c *CoinGeckoClient) GetCoinPrices(coinIDs []string) ([]CoinPrice, error) {
	idsParam := strings.Join(coinIDs, ",")
	url := fmt.Sprintf("%s/coins/markets?vs_currency=usd&ids=%s&order=market_cap_desc&per_page=100&page=1&sparkline=false",
		c.baseURL, idsParam)

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		coins, err := c.doRequest(url)
		if err == nil {
			return coins, nil
		}
		lastErr = err

		if attempt < maxRetries {
			fmt.Printf("请求失败 (尝试 %d/%d): %v，%v 后重试...\n", attempt, maxRetries, err, retryInterval)
			time.Sleep(retryInterval)
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func (c *CoinGeckoClient) doRequest(url string) ([]CoinPrice, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-cg-demo-api-key", c.apiKey)

	resp, err := c.client.Do(req)
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