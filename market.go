package main

import "time"

// MarketReport keeps asset-specific concepts separate instead of forcing every
// market into CoinPrice's cryptocurrency schema.
type MarketReport struct {
	GeneratedAt time.Time
	Crypto      CryptoSection
	Stocks      StockSection
	Perpetuals  PerpetualSection
}

type CryptoSection struct {
	Configured bool
	Items      []CoinPrice
	Warning    string
}

type StockSection struct {
	Configured bool
	Feed       string
	Items      []StockPrice
	Warning    string
}

type PerpetualSection struct {
	Configured bool
	Items      []PerpetualPrice
	Warning    string
}

func (r *MarketReport) HasData() bool {
	return len(r.Crypto.Items) > 0 || len(r.Stocks.Items) > 0 || len(r.Perpetuals.Items) > 0
}

type StockPrice struct {
	Symbol             string
	Price              float64
	PriceTime          time.Time
	PreviousClose      *float64
	DailyChange        *float64
	DailyChangePercent *float64
	HourlyBarStart     *time.Time
	HourlyVWAP         *float64
	HourlyVolume       *float64
	HourlyTurnover     *float64
	HourlyDirection    *float64
}

type PerpetualPrice struct {
	Symbol               string
	MarkPrice            float64
	OraclePrice          *float64
	MarkOracleSpread     *float64
	MarkOracleSpreadPct  *float64
	PriceChange24hPct    *float64
	FundingRate          *float64
	DayTurnover          *float64
	NotionalOpenInterest *float64
	OpenInterestChange   *float64
	HourlyCandleStart    *time.Time
	HourlyVolume         *float64
	HourlyTurnover       *float64
	ObservedAt           time.Time
}

func floatPtr(value float64) *float64 { return &value }

func timePtr(value time.Time) *time.Time { return &value }
