package main

import (
	"math"
	"sort"
	"strings"
	"time"
)

type CanaryPoint struct {
	Date  time.Time
	Value float64
}

type CanaryKind int

const (
	percentChange CanaryKind = iota
	rateChange
	pointChange
	basisPointChange
)
const (
	stablecoinGroup = "稳定币"
	etfGroup        = "ETF"
	fredGroup       = "FRED"
)

type CanarySeries struct {
	Group, ID, Name string
	Kind            CanaryKind
	Points          []CanaryPoint // Ascending UTC dates, one observation per day.
}

var canaryNames = map[string]string{
	"tether": "USDT 供应", "usd-coin": "USDC 供应",
	"TLT": "TLT 长债", "GLD": "GLD 黄金", "UUP": "UUP 美元", "HYG": "HYG 高收益债",
}

func marketCanary(group, id string, samples []CanaryPoint) CanarySeries {
	sort.SliceStable(samples, func(i, j int) bool { return samples[i].Date.Before(samples[j].Date) })
	points := samples[:0]
	for _, p := range samples {
		p.Date = p.Date.UTC().Truncate(24 * time.Hour)
		if n := len(points); n > 0 && points[n-1].Date.Equal(p.Date) {
			points[n-1] = p
		} else {
			points = append(points, p)
		}
	}
	name := canaryNames[id]
	if name == "" {
		name = strings.ToUpper(id)
	}
	return CanarySeries{Group: group, ID: id, Name: name, Kind: percentChange, Points: points}
}

func (s CanarySeries) Change(days int) *float64 {
	if len(s.Points) == 0 {
		return nil
	}
	latest := s.Points[len(s.Points)-1]
	cutoff := latest.Date.AddDate(0, 0, -days)
	index := sort.Search(len(s.Points), func(i int) bool { return s.Points[i].Date.After(cutoff) }) - 1
	if index < 0 {
		return nil
	}
	base := s.Points[index].Value
	delta := latest.Value - base
	switch s.Kind {
	case percentChange:
		if base == 0 {
			return nil
		}
		delta = delta / base * 100
	case rateChange:
		delta *= 100
	}
	return &delta
}

func (s CanarySeries) Sparkline() string {
	if len(s.Points) == 0 {
		return ""
	}
	points := s.Points
	cutoff := points[len(points)-1].Date.AddDate(0, 0, -30)
	start := sort.Search(len(points), func(i int) bool { return !points[i].Date.Before(cutoff) })
	points = points[start:]
	n := min(len(points), 15)
	values := make([]float64, n)
	for i := range values {
		values[i] = points[i*(len(points)-1)/max(1, n-1)].Value
	}
	low, high := values[0], values[0]
	for _, v := range values {
		low = min(low, v)
		high = max(high, v)
	}
	if low == high {
		return strings.Repeat("▄", n)
	}
	levels := []rune("▁▂▃▄▅▆▇█")
	var bar strings.Builder
	for _, v := range values {
		bar.WriteRune(levels[int(math.Round((v-low)/(high-low)*7))])
	}
	return bar.String()
}
