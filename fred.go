package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type FredClient struct {
	baseURL     string
	apiKey      string
	client      *http.Client
	maxAttempts int
	retryDelay  time.Duration
}

func NewFredClient(apiKey string, proxyEnabled bool, proxyURL string) *FredClient {
	return &FredClient{
		baseURL:     "https://api.stlouisfed.org",
		apiKey:      apiKey,
		client:      newMarketHTTPClient(proxyEnabled, proxyURL),
		maxAttempts: 3,
		retryDelay:  2 * time.Second,
	}
}

type FredObservation struct {
	Date  time.Time
	Value float64
}

type FredSeries struct {
	ID string
	// Observations contains valid values in descending observation date order.
	Observations []FredObservation
}

func (c *FredClient) GetSeries(ids []string) ([]FredSeries, error) {
	var items []FredSeries
	var issues []error
	for _, id := range normalizedSymbols(ids) {
		observations, err := c.getObservations(id)
		if err != nil {
			issues = append(issues, fmt.Errorf("%s: %w", id, err))
			continue
		}
		items = append(items, FredSeries{ID: id, Observations: observations})
	}
	return items, errors.Join(issues...)
}

func (c *FredClient) getObservations(id string) ([]FredObservation, error) {
	query := url.Values{"series_id": {id}, "api_key": {c.apiKey}, "file_type": {"json"}, "sort_order": {"desc"}, "limit": {"45"}}
	endpoint := c.baseURL + "/fred/series/observations?" + query.Encode()
	for attempt := 1; ; attempt++ {
		resp, err := c.client.Get(endpoint)
		if err != nil {
			// Transport errors may contain the complete URL, including the key.
			return nil, errors.New("FRED request failed")
		}
		if resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			return parseFredObservations(resp.Body)
		}
		resp.Body.Close()
		retry := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 && resp.StatusCode < 600
		if !retry || attempt >= c.maxAttempts {
			return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		time.Sleep(c.retryDelay)
	}
}

func parseFredObservations(body io.Reader) ([]FredObservation, error) {
	var response struct {
		Observations []struct{ Date, Value string }
	}
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return nil, errors.New("invalid FRED response")
	}
	var observations []FredObservation
	for _, row := range response.Observations {
		if row.Value == "." {
			continue
		}
		date, dateErr := time.Parse(time.DateOnly, row.Date)
		value, err := strconv.ParseFloat(row.Value, 64)
		if dateErr != nil || err != nil || !isFinite(value) {
			return nil, errors.New("invalid FRED observation")
		}
		observations = append(observations, FredObservation{Date: date, Value: value})
	}
	if len(observations) == 0 {
		return nil, errors.New("no usable observations")
	}
	return observations, nil
}

func fredCanaries(series []FredSeries) []CanarySeries {
	var items []CanarySeries
	var sofr []CanaryPoint
	reserves := make(map[time.Time]float64)
	for _, s := range series {
		item := CanarySeries{Group: fredGroup, ID: s.ID, Name: fredSeriesName(s.ID), Kind: pointChange}
		switch s.ID {
		case "BAMLH0A0HYM2", "SOFR", "IORB":
			item.Kind = rateChange
		}
		for i := len(s.Observations) - 1; i >= 0; i-- {
			obs := s.Observations[i]
			item.Points = append(item.Points, CanaryPoint(obs))
			if s.ID == "IORB" {
				reserves[obs.Date] = obs.Value
			}
		}
		if s.ID == "SOFR" {
			sofr = item.Points
		}
		items = append(items, item)
	}
	spread := CanarySeries{Group: fredGroup, ID: "SOFR-IORB", Name: "SOFR − IORB", Kind: basisPointChange}
	for _, p := range sofr {
		if reserve, ok := reserves[p.Date]; ok {
			spread.Points = append(spread.Points, CanaryPoint{Date: p.Date, Value: (p.Value - reserve) * 100})
		}
	}
	if len(spread.Points) > 0 {
		items = append(items, spread)
	}
	return items
}
