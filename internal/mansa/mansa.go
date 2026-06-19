package mansa

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const BaseURL = "https://mansaapi.com/api/v1"

type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error"`
}
type Stock struct {
	Symbol    string  `json:"ticker"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	ChangePct float64 `json:"change_pct"`
	Volume    int64   `json:"volume"`
	MarketCap float64 `json:"market_cap"`
}

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func New() *Client {
	return &Client{
		apiKey:     os.Getenv("MANSA_API_KEY"),
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) get(path string, target any) error {
	req, err := http.NewRequest("GET", BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http get %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api returned %d: %s", resp.StatusCode, string(body))
	}

	var envelope apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("decode envelope: %w", err)
	}
	if envelope.Error != nil {
		return fmt.Errorf("api error: %s", envelope.Error.Message)
	}

	return json.Unmarshal(envelope.Data, target)
}

func (c *Client) AllStocks() ([]Stock, error) {
	var result []Stock
	err := c.get("/markets/exchanges/NSE/stocks", &result)
	return result, err
}

func (c *Client) SingleStock(ticker string) (*Stock, error) {
	var result Stock
	err := c.get("/markets/exchanges/NSE/stocks/resolve/"+ticker, &result)
	return &result, err
}
