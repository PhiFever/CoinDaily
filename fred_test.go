package main

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fredTestClient(t *testing.T, handler http.HandlerFunc) *FredClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := NewFredClient("test-secret", false, "")
	client.baseURL = server.URL
	client.retryDelay = 0
	return client
}

func TestFredValidObservations(t *testing.T) {
	client := fredTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.Method != http.MethodGet || r.URL.Path != "/fred/series/observations" || q.Get("series_id") != "SOFR" || q.Get("api_key") != "test-secret" || q.Get("limit") != "10" || q.Get("sort_order") != "desc" || q.Get("file_type") != "json" {
			t.Error("unexpected FRED request")
		}
		fmt.Fprint(w, `{"observations":[{"date":"2026-11-19","value":"."},{"date":"2026-11-18","value":"3.12"},{"date":"2026-11-17","value":"."},{"date":"2026-11-16","value":"3.10"}]}`)
	})
	items, err := client.GetSeries([]string{"SOFR"})
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%v err=%v", items, err)
	}
	s := items[0]
	if len(s.Observations) != 2 || s.Observations[0].Value != 3.12 || s.Observations[0].Date.Format("01-02") != "11-18" || s.Observations[1].Value != 3.10 || s.Change() == nil || math.Abs(*s.Change()-0.02) > 1e-8 {
		t.Fatalf("series=%+v change=%v", s, s.Change())
	}
}

func TestFredErrorsHideAPIKey(t *testing.T) {
	for _, status := range []int{400, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client := fredTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, r.URL.String(), status)
			})
			_, err := client.GetSeries([]string{"SOFR"})
			if err == nil || strings.Contains(err.Error(), client.apiKey) || !strings.Contains(err.Error(), fmt.Sprintf("HTTP %d", status)) {
				t.Fatalf("unexpected error for status %d", status)
			}
		})
	}
}
