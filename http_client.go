package main

import (
	"net/http"
	"net/url"
	"time"
)

func newMarketHTTPClient(proxyEnabled bool, proxyURL string) *http.Client {
	client := &http.Client{Timeout: 30 * time.Second}
	if proxyEnabled && proxyURL != "" {
		if parsed, err := url.Parse(proxyURL); err == nil {
			client.Transport = &http.Transport{Proxy: http.ProxyURL(parsed)}
		}
	}
	return client
}
