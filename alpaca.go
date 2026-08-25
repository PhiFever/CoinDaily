package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const alpacaDelayedSIPLag = 15 * time.Minute

type AlpacaClient struct {
	baseURL     string
	apiKey      string
	secretKey   string
	feed        string
	client      *http.Client
	maxAttempts int
	retryDelay  time.Duration
}

func NewAlpacaClient(apiKey, secretKey, feed string, proxyEnabled bool, proxyURL string) *AlpacaClient {
	if feed == "" {
		feed = "delayed_sip"
	}
	return &AlpacaClient{
		baseURL:     "https://data.alpaca.markets",
		apiKey:      apiKey,
		secretKey:   secretKey,
		feed:        feed,
		client:      newMarketHTTPClient(proxyEnabled, proxyURL),
		maxAttempts: 3,
		retryDelay:  2 * time.Second,
	}
}

type alpacaBar struct {
	Timestamp time.Time `json:"t"`
	Open      float64   `json:"o"`
	Close     float64   `json:"c"`
	Volume    float64   `json:"v"`
	VWAP      float64   `json:"vw"`
}

func (c *AlpacaClient) GetStockPrices(symbols []string, now time.Time) ([]StockPrice, error) {
	if len(symbols) == 0 {
		return nil, nil
	}

	requested := normalizedSymbols(symbols)
	cutoff := now.UTC().Add(-alpacaDelayedSIPLag)
	minuteBars, err := c.getBars(requested, "1Min", cutoff.Add(-10*24*time.Hour), cutoff, 10000)
	if err != nil {
		return nil, fmt.Errorf("Alpaca delayed observations: %w", err)
	}
	dailyBars, dailyErr := c.getBars(requested, "1Day", cutoff.Add(-20*24*time.Hour), cutoff, 1000)

	prices := make([]StockPrice, 0, len(requested))
	var issues []error
	if dailyErr != nil {
		issues = append(issues, fmt.Errorf("Alpaca previous daily bars: %w", dailyErr))
	}
	for _, symbol := range requested {
		latest, ok := latestCompletedPriceBar(minuteBars[symbol], cutoff)
		if !ok {
			issues = append(issues, fmt.Errorf("%s has no delayed SIP observation", symbol))
			continue
		}
		item := StockPrice{Symbol: symbol, Price: latest.Close, PriceTime: latest.Timestamp}
		if previousBar, ok := latestPreviousDailyBar(dailyBars[symbol], latest.Timestamp); ok {
			previousClose := previousBar.Close
			change := latest.Close - previousClose
			changePct := change / previousClose * 100
			item.PreviousClose = floatPtr(previousClose)
			item.DailyChange = floatPtr(change)
			item.DailyChangePercent = floatPtr(changePct)
		}
		prices = append(prices, item)
	}

	if len(prices) == 0 {
		return nil, errors.Join(issues...)
	}

	bars, barErr := c.getBars(requested, "1Hour", cutoff.Add(-10*24*time.Hour), cutoff, 1000)
	if barErr != nil {
		issues = append(issues, fmt.Errorf("Alpaca hourly bars: %w", barErr))
	} else {
		for i := range prices {
			bar, ok := latestCompletedStockBarForDuration(bars[prices[i].Symbol], cutoff, time.Hour)
			if !ok {
				issues = append(issues, fmt.Errorf("%s has no completed 1h bar", prices[i].Symbol))
				continue
			}
			turnover := bar.VWAP * bar.Volume
			direction := bar.Close - bar.Open
			prices[i].HourlyBarStart = timePtr(bar.Timestamp)
			prices[i].HourlyVWAP = floatPtr(bar.VWAP)
			prices[i].HourlyVolume = floatPtr(bar.Volume)
			prices[i].HourlyTurnover = floatPtr(turnover)
			prices[i].HourlyDirection = floatPtr(direction)
		}
	}

	return prices, errors.Join(issues...)
}

func (c *AlpacaClient) getBars(symbols []string, timeframe string, start, end time.Time, limit int) (map[string][]alpacaBar, error) {
	query := url.Values{}
	query.Set("symbols", strings.Join(symbols, ","))
	query.Set("timeframe", timeframe)
	query.Set("start", start.UTC().Format(time.RFC3339))
	query.Set("end", end.UTC().Format(time.RFC3339))
	// delayed_sip is a streaming feed name. Historical REST uses SIP with an
	// end time at least 15 minutes old for delayed consolidated observations.
	query.Set("feed", "sip")
	query.Set("adjustment", "raw")
	query.Set("limit", strconv.Itoa(limit))

	result := make(map[string][]alpacaBar)
	for page := 0; page < 20; page++ {
		body, err := c.get("/v2/stocks/bars", query)
		if err != nil {
			return nil, err
		}
		var response struct {
			Bars          map[string][]alpacaBar `json:"bars"`
			NextPageToken string                 `json:"next_page_token"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("Alpaca %s bars JSON: %w", timeframe, err)
		}
		for symbol, bars := range response.Bars {
			result[symbol] = append(result[symbol], bars...)
		}
		if response.NextPageToken == "" {
			return result, nil
		}
		query.Set("page_token", response.NextPageToken)
	}
	return nil, fmt.Errorf("Alpaca %s bars exceeded 20 pages", timeframe)
}

func (c *AlpacaClient) get(path string, query url.Values) ([]byte, error) {
	requestURL := c.baseURL + path + "?" + query.Encode()
	var lastErr error
	for attempt := 1; attempt <= c.maxAttempts; attempt++ {
		req, err := http.NewRequest(http.MethodGet, requestURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("APCA-API-KEY-ID", c.apiKey)
		req.Header.Set("APCA-API-SECRET-KEY", c.secretKey)

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
		} else {
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
			resp.Body.Close()
			if readErr != nil {
				lastErr = fmt.Errorf("read response: %w", readErr)
			} else if resp.StatusCode != http.StatusOK {
				lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
			} else {
				return body, nil
			}
		}
		if attempt < c.maxAttempts {
			time.Sleep(c.retryDelay)
		}
	}
	return nil, lastErr
}

func latestCompletedStockBarForDuration(bars []alpacaBar, cutoff time.Time, duration time.Duration) (alpacaBar, bool) {
	var latest alpacaBar
	found := false
	for _, bar := range bars {
		if bar.Timestamp.IsZero() || bar.Timestamp.Add(duration).After(cutoff) || bar.Open <= 0 || bar.Close <= 0 || bar.Volume < 0 || bar.VWAP <= 0 || !isFinite(bar.Open) || !isFinite(bar.Close) || !isFinite(bar.VWAP) || !isFinite(bar.Volume) {
			continue
		}
		if !found || bar.Timestamp.After(latest.Timestamp) {
			latest = bar
			found = true
		}
	}
	return latest, found
}

func latestCompletedPriceBar(bars []alpacaBar, cutoff time.Time) (alpacaBar, bool) {
	var latest alpacaBar
	found := false
	for _, bar := range bars {
		if bar.Timestamp.IsZero() || bar.Timestamp.Add(time.Minute).After(cutoff) || bar.Close <= 0 || !isFinite(bar.Close) {
			continue
		}
		if !found || bar.Timestamp.After(latest.Timestamp) {
			latest = bar
			found = true
		}
	}
	return latest, found
}

func latestPreviousDailyBar(bars []alpacaBar, observationTime time.Time) (alpacaBar, bool) {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		location = time.FixedZone("America/New_York", -5*60*60)
	}
	observationDate := observationTime.In(location).Format("2006-01-02")
	var latest alpacaBar
	found := false
	for _, bar := range bars {
		barDate := bar.Timestamp.In(location).Format("2006-01-02")
		if barDate >= observationDate || bar.Close <= 0 || !isFinite(bar.Close) {
			continue
		}
		if !found || bar.Timestamp.After(latest.Timestamp) {
			latest = bar
			found = true
		}
	}
	return latest, found
}

func normalizedSymbols(symbols []string) []string {
	seen := make(map[string]bool, len(symbols))
	result := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		symbol = strings.ToUpper(strings.TrimSpace(symbol))
		if symbol != "" && !seen[symbol] {
			seen[symbol] = true
			result = append(result, symbol)
		}
	}
	sort.Strings(result)
	return result
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
