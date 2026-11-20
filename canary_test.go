package main

import (
	"strings"
	"testing"
	"time"
)

func TestCanaryTrends(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	series := CanarySeries{Kind: pointChange}
	for day := range 21 {
		if day == 13 {
			continue
		} // No observation exactly seven days before the latest.
		series.Points = append(series.Points, CanaryPoint{Date: start.AddDate(0, 0, day), Value: float64(day)})
	}
	if change := series.Change(7); change == nil || *change != 8 {
		t.Fatalf("7d change=%v", change)
	}
	if series.Change(30) != nil {
		t.Fatal("unexpected 30d change")
	}
	if bar := []rune(series.Sparkline()); len(bar) != 15 || bar[len(bar)-1] != '█' {
		t.Fatalf("sparkline=%s", string(bar))
	}
	for i := range series.Points {
		series.Points[i].Value = 100
	}
	if bar := series.Sparkline(); bar != strings.Repeat("▄", 15) {
		t.Fatalf("flat sparkline=%s", bar)
	}
}
