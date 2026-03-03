package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://api.coingecko.com/api/v3"

type Client struct {
	http *http.Client
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetPrices fetches USD prices for the given CoinGecko token IDs.
func (c *Client) GetPrices(ctx context.Context, tokenIDs []string) (map[string]float64, error) {
	if len(tokenIDs) == 0 {
		return map[string]float64{}, nil
	}

	url := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=usd", baseURL, strings.Join(tokenIDs, ","))
	fmt.Println("url", url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("coingecko: build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("coingecko: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coingecko: unexpected status %d", resp.StatusCode)
	}

	// Response shape: {"bitcoin": {"usd": 67000.50}, "ethereum": {"usd": 3500.20}}
	var raw map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("coingecko: decode response: %w", err)
	}

	prices := make(map[string]float64, len(raw))
	for token, currencies := range raw {
		if usd, ok := currencies["usd"]; ok {
			prices[token] = usd
		}
	}
	return prices, nil
}
