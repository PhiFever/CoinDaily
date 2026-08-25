package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type HyperliquidClient struct {
	baseURL     string
	client      *http.Client
	maxAttempts int
	retryDelay  time.Duration
}

func NewHyperliquidClient(proxyEnabled bool, proxyURL string) *HyperliquidClient {
	return &HyperliquidClient{
		baseURL:     "https://api.hyperliquid.xyz",
		client:      newMarketHTTPClient(proxyEnabled, proxyURL),
		maxAttempts: 3,
		retryDelay:  2 * time.Second,
	}
}

type hyperMeta struct {
	Universe []struct {
		Name string `json:"name"`
	} `json:"universe"`
}

type hyperAssetContext struct {
	MarkPrice     json.RawMessage `json:"markPx"`
	OraclePrice   json.RawMessage `json:"oraclePx"`
	PreviousPrice json.RawMessage `json:"prevDayPx"`
	Funding       json.RawMessage `json:"funding"`
	OpenInterest  json.RawMessage `json:"openInterest"`
	DayTurnover   json.RawMessage `json:"dayNtlVlm"`
}

type hyperCandle struct {
	Start    int64           `json:"t"`
	End      int64           `json:"T"`
	Symbol   string          `json:"s"`
	Interval string          `json:"i"`
	Open     json.RawMessage `json:"o"`
	High     json.RawMessage `json:"h"`
	Low      json.RawMessage `json:"l"`
	Close    json.RawMessage `json:"c"`
	Volume   json.RawMessage `json:"v"`
}

func (c *HyperliquidClient) GetPerpetualPrices(symbols []string, now time.Time) ([]PerpetualPrice, error) {
	if len(symbols) == 0 {
		return nil, nil
	}

	contextsByDEX := make(map[string]map[string]json.RawMessage)
	var issues []error
	for _, symbol := range symbols {
		dex, ok := perpetualDEX(symbol)
		if !ok {
			issues = append(issues, fmt.Errorf("%s has no DEX prefix", symbol))
			continue
		}
		if _, loaded := contextsByDEX[dex]; loaded {
			continue
		}
		contexts, err := c.getAssetContexts(dex)
		if err != nil {
			issues = append(issues, fmt.Errorf("DEX %s metadata: %w", dex, err))
			contextsByDEX[dex] = nil
			continue
		}
		contextsByDEX[dex] = contexts
	}

	prices := make([]PerpetualPrice, 0, len(symbols))
	seen := make(map[string]bool, len(symbols))
	for _, configuredSymbol := range symbols {
		symbol := strings.TrimSpace(configuredSymbol)
		if symbol == "" || seen[symbol] {
			continue
		}
		seen[symbol] = true
		dex, ok := perpetualDEX(symbol)
		if !ok {
			continue
		}
		contexts := contextsByDEX[dex]
		if contexts == nil {
			continue
		}
		raw, found := contexts[symbol]
		if !found {
			issues = append(issues, fmt.Errorf("%s is absent from Hyperliquid universe", symbol))
			continue
		}

		item, err := parsePerpetualContext(symbol, raw, now)
		if err != nil {
			issues = append(issues, err)
			continue
		}
		candle, err := c.getLatestCompletedCandle(symbol, now)
		if err != nil {
			issues = append(issues, fmt.Errorf("%s 1h candle: %w", symbol, err))
		} else if candle != nil {
			item.HourlyCandleStart = timePtr(time.UnixMilli(candle.Start))
			item.HourlyVolume = floatPtr(candle.Volume)
			item.HourlyTurnover = floatPtr(candle.Volume * candle.RepresentativePrice)
		} else {
			issues = append(issues, fmt.Errorf("%s has no completed 1h candle", symbol))
		}
		prices = append(prices, item)
	}

	return prices, errors.Join(issues...)
}

func (c *HyperliquidClient) getAssetContexts(dex string) (map[string]json.RawMessage, error) {
	body, err := c.post(map[string]any{"type": "metaAndAssetCtxs", "dex": dex})
	if err != nil {
		return nil, err
	}
	var response []json.RawMessage
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode metaAndAssetCtxs: %w", err)
	}
	if len(response) != 2 {
		return nil, fmt.Errorf("metaAndAssetCtxs returned %d elements, want 2", len(response))
	}
	var meta hyperMeta
	if err := json.Unmarshal(response[0], &meta); err != nil {
		return nil, fmt.Errorf("decode perpetual metadata: %w", err)
	}
	var contexts []json.RawMessage
	if err := json.Unmarshal(response[1], &contexts); err != nil {
		return nil, fmt.Errorf("decode asset contexts: %w", err)
	}
	if len(contexts) < len(meta.Universe) {
		return nil, fmt.Errorf("asset contexts length %d is shorter than universe length %d", len(contexts), len(meta.Universe))
	}
	result := make(map[string]json.RawMessage, len(meta.Universe))
	for index, asset := range meta.Universe {
		result[asset.Name] = contexts[index]
	}
	return result, nil
}

type completedHyperCandle struct {
	Start               int64
	Volume              float64
	RepresentativePrice float64
}

func (c *HyperliquidClient) getLatestCompletedCandle(symbol string, now time.Time) (*completedHyperCandle, error) {
	body, err := c.post(map[string]any{
		"type": "candleSnapshot",
		"req": map[string]any{
			"coin":      symbol,
			"interval":  "1h",
			"startTime": now.Add(-8 * time.Hour).UnixMilli(),
			"endTime":   now.UnixMilli(),
		},
	})
	if err != nil {
		return nil, err
	}
	var candles []hyperCandle
	if err := json.Unmarshal(body, &candles); err != nil {
		return nil, fmt.Errorf("decode candleSnapshot: %w", err)
	}

	cutoff := now.Truncate(time.Hour).UnixMilli()
	var selected *hyperCandle
	for index := range candles {
		candle := &candles[index]
		if candle.Symbol != symbol || candle.Interval != "1h" || candle.End >= cutoff {
			continue
		}
		if selected == nil || candle.Start > selected.Start {
			selected = candle
		}
	}
	if selected == nil {
		return nil, nil
	}

	open, err := requiredJSONNumber(selected.Open, "candle open")
	if err != nil {
		return nil, err
	}
	high, err := requiredJSONNumber(selected.High, "candle high")
	if err != nil {
		return nil, err
	}
	low, err := requiredJSONNumber(selected.Low, "candle low")
	if err != nil {
		return nil, err
	}
	closePrice, err := requiredJSONNumber(selected.Close, "candle close")
	if err != nil {
		return nil, err
	}
	volume, err := requiredJSONNumber(selected.Volume, "candle volume")
	if err != nil {
		return nil, err
	}
	if volume < 0 {
		return nil, fmt.Errorf("candle volume is negative")
	}
	return &completedHyperCandle{
		Start:               selected.Start,
		Volume:              volume,
		RepresentativePrice: (open + high + low + closePrice) / 4,
	}, nil
}

func (c *HyperliquidClient) post(payload any) ([]byte, error) {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}
	var lastErr error
	for attempt := 1; attempt <= c.maxAttempts; attempt++ {
		req, err := http.NewRequest(http.MethodPost, c.baseURL+"/info", bytes.NewReader(requestBody))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
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

func parsePerpetualContext(symbol string, raw json.RawMessage, now time.Time) (PerpetualPrice, error) {
	var context hyperAssetContext
	if err := json.Unmarshal(raw, &context); err != nil {
		return PerpetualPrice{}, fmt.Errorf("%s context JSON: %w", symbol, err)
	}
	mark, err := requiredJSONNumber(context.MarkPrice, "markPx")
	if err != nil || mark <= 0 {
		if err == nil {
			err = fmt.Errorf("must be positive")
		}
		return PerpetualPrice{}, fmt.Errorf("%s markPx: %w", symbol, err)
	}
	item := PerpetualPrice{Symbol: symbol, MarkPrice: mark, ObservedAt: now}

	oracle, err := optionalJSONNumber(context.OraclePrice, "oraclePx")
	if err != nil {
		return PerpetualPrice{}, fmt.Errorf("%s: %w", symbol, err)
	}
	item.OraclePrice = oracle
	if oracle != nil && *oracle != 0 {
		spread := mark - *oracle
		item.MarkOracleSpread = floatPtr(spread)
		item.MarkOracleSpreadPct = floatPtr(spread / *oracle * 100)
	}

	previous, err := optionalJSONNumber(context.PreviousPrice, "prevDayPx")
	if err != nil {
		return PerpetualPrice{}, fmt.Errorf("%s: %w", symbol, err)
	}
	if previous != nil && *previous != 0 {
		item.PriceChange24hPct = floatPtr((mark - *previous) / *previous * 100)
	}

	item.FundingRate, err = optionalJSONNumber(context.Funding, "funding")
	if err != nil {
		return PerpetualPrice{}, fmt.Errorf("%s: %w", symbol, err)
	}
	item.DayTurnover, err = optionalJSONNumber(context.DayTurnover, "dayNtlVlm")
	if err != nil {
		return PerpetualPrice{}, fmt.Errorf("%s: %w", symbol, err)
	}
	openInterest, err := optionalJSONNumber(context.OpenInterest, "openInterest")
	if err != nil {
		return PerpetualPrice{}, fmt.Errorf("%s: %w", symbol, err)
	}
	if openInterest != nil && *openInterest >= 0 {
		notional := *openInterest * mark
		if isFinite(notional) {
			item.NotionalOpenInterest = floatPtr(notional)
		}
	}
	return item, nil
}

func requiredJSONNumber(raw json.RawMessage, name string) (float64, error) {
	value, err := optionalJSONNumber(raw, name)
	if err != nil {
		return 0, err
	}
	if value == nil {
		return 0, fmt.Errorf("%s is unavailable", name)
	}
	return *value, nil
}

func optionalJSONNumber(raw json.RawMessage, name string) (*float64, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || trimmed == `""` {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, `"`) {
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return nil, fmt.Errorf("%s is malformed: %w", name, err)
		}
		trimmed = text
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, fmt.Errorf("%s is not a finite number", name)
	}
	return floatPtr(value), nil
}

func perpetualDEX(symbol string) (string, bool) {
	dex, _, ok := strings.Cut(strings.TrimSpace(symbol), ":")
	return dex, ok && dex != ""
}
