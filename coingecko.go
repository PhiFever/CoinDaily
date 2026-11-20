package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CoinPrice struct {
	ID                     string   `json:"id"`
	Symbol                 string   `json:"symbol"`
	Name                   string   `json:"name"`
	CurrentPrice           float64  `json:"current_price"`
	MarketCap              float64  `json:"market_cap"`
	MarketCapChange24h     *float64 `json:"market_cap_change_24h"`
	MarketCapChangePerc24h *float64 `json:"market_cap_change_percentage_24h"`
	PriceChange24h         float64  `json:"price_change_24h"`
	PriceChangePerc24h     float64  `json:"price_change_percentage_24h"`
	Volume24h              float64  `json:"total_volume"`
	LastUpdated            string   `json:"last_updated"`
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

	return getCoinGeckoJSON[[]CoinPrice](c, url)
}

func (c *CoinGeckoClient) GetMarketCaps(ids []string) ([]CanarySeries, error) {
	var items []CanarySeries
	var issues []error
	for _, id := range ids {
		type marketChart struct {
			MarketCaps [][]float64 `json:"market_caps"`
		}
		endpoint := c.baseURL + "/coins/" + url.PathEscape(id) + "/market_chart?vs_currency=usd&days=31&interval=daily"
		chart, err := getCoinGeckoJSON[marketChart](c, endpoint)
		if err != nil {
			issues = append(issues, fmt.Errorf("%s: %w", id, err))
			continue
		}
		var samples []CanaryPoint
		for _, row := range chart.MarketCaps {
			if len(row) == 2 {
				samples = append(samples, CanaryPoint{Date: time.UnixMilli(int64(row[0])), Value: row[1]})
			}
		}
		if len(samples) == 0 {
			issues = append(issues, fmt.Errorf("%s has no market cap observations", id))
			continue
		}
		items = append(items, marketCanary(stablecoinGroup, id, samples))
	}
	return items, errors.Join(issues...)
}

func getCoinGeckoJSON[T any](c *CoinGeckoClient, url string) (T, error) {
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		var result T
		err := c.doRequest(url, &result)
		if err == nil {
			return result, nil
		}
		lastErr = err

		if attempt < maxRetries {
			fmt.Printf("请求失败 (尝试 %d/%d): %v，%v 后重试...\n", attempt, maxRetries, err, retryInterval)
			time.Sleep(retryInterval)
		}
	}

	var zero T
	return zero, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func (c *CoinGeckoClient) doRequest(url string, target any) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-cg-demo-api-key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch data from CoinGecko: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CoinGecko API returned status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return nil
}
