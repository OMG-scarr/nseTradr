package sheets

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"nseTradr/internal/models"
)

// Client wraps the Google Sheets service with the spreadsheet ID.
type Client struct {
	svc         *sheets.Service
	spreadsheet string
}

// New creates a Sheets client using the service account credentials file.
func New() (*Client, error) {
	credsFile := os.Getenv("GOOGLE_CREDENTIALS_FILE")
	if credsFile == "" {
		credsFile = "./credentials.json"
	}

	sheetID := os.Getenv("GOOGLE_SHEETS_ID")
	if sheetID == "" {
		return nil, fmt.Errorf("GOOGLE_SHEETS_ID environment variable is not set")
	}

	ctx := context.Background()
	svc, err := sheets.NewService(ctx,
		option.WithCredentialsFile(credsFile),
		option.WithScopes(sheets.SpreadsheetsScope),
	)
	if err != nil {
		return nil, fmt.Errorf("create sheets service: %w", err)
	}

	return &Client{svc: svc, spreadsheet: sheetID}, nil
}

// ReadHoldings reads all rows from the Holdings tab.
// Expects header in row 1: Ticker | Name | SharesHeld | AvgBuyPrice | TargetPct | WeeklyBudget
func (c *Client) ReadHoldings() ([]models.Holding, error) {
	resp, err := c.svc.Spreadsheets.Values.Get(c.spreadsheet, "Holdings!A2:F").Do()
	if err != nil {
		return nil, fmt.Errorf("read holdings: %w", err)
	}

	holdings := make([]models.Holding, 0, len(resp.Values))
	for _, row := range resp.Values {
		if len(row) < 4 {
			continue // skip blank or incomplete rows
		}
		h := models.Holding{
			Ticker:       str(row, 0),
			Name:         str(row, 1),
			SharesHeld:   float64val(row, 2),
			AvgBuyPrice:  float64val(row, 3),
			TargetPct:    float64val(row, 4),
			WeeklyBudget: float64val(row, 5),
		}
		if h.Ticker == "" {
			continue
		}
		holdings = append(holdings, h)
	}

	return holdings, nil
}

// ReadAlertRules reads all rows from the Alerts tab.
// Expects header in row 1: Ticker | AlertAbove | AlertBelow | Label
func (c *Client) ReadAlertRules() ([]models.AlertRule, error) {
	resp, err := c.svc.Spreadsheets.Values.Get(c.spreadsheet, "Alerts!A2:D").Do()
	if err != nil {
		return nil, fmt.Errorf("read alert rules: %w", err)
	}

	rules := make([]models.AlertRule, 0, len(resp.Values))
	for _, row := range resp.Values {
		if len(row) < 1 {
			continue
		}
		r := models.AlertRule{
			Ticker:     str(row, 0),
			AlertAbove: float64val(row, 1),
			AlertBelow: float64val(row, 2),
			Label:      str(row, 3),
		}
		if r.Ticker == "" {
			continue
		}
		rules = append(rules, r)
	}

	return rules, nil
}

// AppendDailyPrices appends scanner results to the DailyPrices tab.
func (c *Client) AppendDailyPrices(results []models.ScanResult) error {
	if err := c.ensureHeader("DailyPrices!A1:G1", []interface{}{
		"Date", "Ticker", "Name", "Price", "ChangePct", "Volume", "Signal",
	}); err != nil {
		return err
	}

	rows := make([][]interface{}, 0, len(results))
	for _, r := range results {
		rows = append(rows, []interface{}{
			r.Date, r.Ticker, r.Name, r.Price, r.ChangePct, r.Volume, r.Signal,
		})
	}

	return c.append("DailyPrices!A:G", rows)
}

// AppendSentiment appends AI-scored sentiment results to the Sentiment tab.
func (c *Client) AppendSentiment(results []models.SentimentResult) error {
	if err := c.ensureHeader("Sentiment!A1:H1", []interface{}{
		"Date", "Ticker", "Company", "Sentiment", "Confidence", "Reason", "Headline", "Link",
	}); err != nil {
		return err
	}

	rows := make([][]interface{}, 0, len(results))
	for _, r := range results {
		rows = append(rows, []interface{}{
			r.Date, r.Ticker, r.Company, r.Sentiment, r.Confidence, r.Reason, r.Headline, r.Link,
		})
	}

	return c.append("Sentiment!A:H", rows)
}

// WriteDCAPlan overwrites the DCA Plan tab with this week's recommendations.
func (c *Client) WriteDCAPlan(recs []models.DCARecommendation, commentary string) error {
	timestamp := time.Now().Format("Mon 02 Jan 2006 15:04")

	values := [][]interface{}{
		{"DCA Plan — Generated: " + timestamp},
		{},
		{"AI Commentary:"},
		{commentary},
		{},
		{"Ticker", "CurrentPrice", "AvgBuyPrice", "PnL%", "DipFromAvg%",
			"BaseKES", "AdjustedKES", "SharesToBuy", "NewAvgAfterBuy", "Action", "Reason"},
	}

	for _, r := range recs {
		values = append(values, []interface{}{
			r.Ticker, r.CurrentPrice, r.AvgBuyPrice,
			fmt.Sprintf("%.2f%%", r.CurrentPnLPct),
			fmt.Sprintf("%.2f%%", r.DipPct),
			r.BaseWeeklyKES, r.AdjustedKES,
			r.SharesToBuy, r.NewAvgAfterBuy, r.Action, r.Reason,
		})
	}

	if err := c.clear("DCA Plan!A:Z"); err != nil {
		return err
	}
	return c.write("DCA Plan!A1", values)
}

// WriteRebalancePlan overwrites the Rebalance tab with latest rebalance actions.
func (c *Client) WriteRebalancePlan(actions []models.RebalanceAction, commentary string) error {
	timestamp := time.Now().Format("Mon 02 Jan 2006 15:04")

	values := [][]interface{}{
		{"Rebalance Report — Generated: " + timestamp},
		{},
		{"AI Commentary:"},
		{commentary},
		{},
		{"Ticker", "CurrentPct%", "TargetPct%", "Drift%", "CurrentValue(KES)",
			"TargetValue(KES)", "Action", "KESAmount", "Shares", "Reason"},
	}

	for _, a := range actions {
		values = append(values, []interface{}{
			a.Ticker,
			fmt.Sprintf("%.1f%%", a.CurrentPct),
			fmt.Sprintf("%.1f%%", a.TargetPct),
			fmt.Sprintf("%.1f%%", a.DriftPct),
			fmt.Sprintf("%.0f", a.CurrentValue),
			fmt.Sprintf("%.0f", a.TargetValue),
			a.Action,
			fmt.Sprintf("%.0f", a.KESAmount),
			fmt.Sprintf("%.0f", a.Shares),
			a.Reason,
		})
	}

	if err := c.clear("Rebalance!A:Z"); err != nil {
		return err
	}
	return c.write("Rebalance!A1", values)
}

// ─── internal helpers ────────────────────────────────────────────────────────

func (c *Client) append(rangeStr string, rows [][]interface{}) error {
	body := &sheets.ValueRange{Values: rows}
	_, err := c.svc.Spreadsheets.Values.Append(
		c.spreadsheet, rangeStr, body,
	).ValueInputOption("RAW").InsertDataOption("INSERT_ROWS").Do()
	if err != nil {
		return fmt.Errorf("append to %s: %w", rangeStr, err)
	}
	return nil
}

func (c *Client) write(rangeStr string, values [][]interface{}) error {
	body := &sheets.ValueRange{Values: values}
	_, err := c.svc.Spreadsheets.Values.Update(
		c.spreadsheet, rangeStr, body,
	).ValueInputOption("RAW").Do()
	if err != nil {
		return fmt.Errorf("write to %s: %w", rangeStr, err)
	}
	return nil
}

func (c *Client) clear(rangeStr string) error {
	_, err := c.svc.Spreadsheets.Values.Clear(
		c.spreadsheet, rangeStr, &sheets.ClearValuesRequest{},
	).Do()
	if err != nil {
		return fmt.Errorf("clear %s: %w", rangeStr, err)
	}
	return nil
}

func (c *Client) ensureHeader(rangeStr string, headers []interface{}) error {
	resp, err := c.svc.Spreadsheets.Values.Get(c.spreadsheet, rangeStr).Do()
	if err != nil {
		return err
	}
	if len(resp.Values) > 0 && len(resp.Values[0]) > 0 {
		return nil // header already exists
	}
	return c.write(rangeStr, [][]interface{}{headers})
}

func str(row []interface{}, i int) string {
	if i >= len(row) {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", row[i]))
}

func float64val(row []interface{}, i int) float64 {
	s := str(row, i)
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// UpdateHolding updates SharesHeld, AvgBuyPrice, and LastUpdated for a ticker after a trade.
func (c *Client) UpdateHolding(ticker string, newShares, newAvg float64) error {
	resp, err := c.svc.Spreadsheets.Values.Get(c.spreadsheet, "Holdings!A2:G").Do()
	if err != nil {
		return fmt.Errorf("read holdings for update: %w", err)
	}

	// Find the row index for this ticker
	rowIndex := -1
	for i, row := range resp.Values {
		if len(row) > 0 && strings.EqualFold(str(row, 0), ticker) {
			rowIndex = i
			break
		}
	}

	if rowIndex == -1 {
		return fmt.Errorf("ticker %s not found in Holdings sheet", ticker)
	}

	// Row index in sheet = rowIndex + 2 (1 for header, 1 for 0-based index)
	sheetRow := rowIndex + 2
	rangeStr := fmt.Sprintf("Holdings!C%d:G%d", sheetRow, sheetRow)

	values := [][]interface{}{
		{
			newShares,
			newAvg,
			// TargetPct and WeeklyBudget unchanged — read existing values
			str(resp.Values[rowIndex], 4),
			str(resp.Values[rowIndex], 5),
			time.Now().Format("2006-01-02 15:04:05"),
		},
	}

	return c.write(rangeStr, values)
}

// ReadHolding reads a single holding by ticker.
func (c *Client) ReadHolding(ticker string) (*models.Holding, error) {
	holdings, err := c.ReadHoldings()
	if err != nil {
		return nil, err
	}
	for _, h := range holdings {
		if strings.EqualFold(h.Ticker, ticker) {
			return &h, nil
		}
	}
	return nil, fmt.Errorf("ticker %s not found in Holdings sheet", ticker)
}
