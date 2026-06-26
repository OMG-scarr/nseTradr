package trade

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"nseTradr/internal/gmail"
	"nseTradr/internal/logger"
	"nseTradr/internal/sheets"
)

type TradeResult struct {
	Ticker       string  `json:"ticker"`
	Action       string  `json:"action"`
	SharesTraded float64 `json:"shares_traded"`
	TradePrice   float64 `json:"trade_price"`
	TotalCost    float64 `json:"total_cost_kes"`
	PrevShares   float64 `json:"prev_shares"`
	PrevAvg      float64 `json:"prev_avg_buy_price"`
	NewShares    float64 `json:"new_shares"`
	NewAvg       float64 `json:"new_avg_buy_price"`
	PnLPct       float64 `json:"pnl_pct"`
	Timestamp    string  `json:"timestamp"`
}

func Run() error {
	log, err := logger.New("trade")
	if err != nil {
		return err
	}
	defer log.Close()

	// Parse flags manually — avoids importing flag package complexity
	args := os.Args[2:] // skip "nseTradr trade"
	params := parseArgs(args)

	ticker := strings.ToUpper(params["ticker"])
	sharesStr := params["shares"]
	priceStr := params["price"]

	// Validate required fields
	if ticker == "" || sharesStr == "" || priceStr == "" {
		return fmt.Errorf(`
usage: nseTradr trade --ticker SCOM --shares 100 --price 32.50

  --ticker   NSE ticker symbol (required)
  --shares   number of shares bought (required)
  --price    price per share in KES (required)

example:
  nseTradr trade --ticker SCOM --shares 100 --price 32.50
  nseTradr trade --ticker EQTY --shares 50 --price 79.75`)
	}

	shares, err := strconv.ParseFloat(sharesStr, 64)
	if err != nil || shares <= 0 {
		return fmt.Errorf("invalid shares value: %s — must be a positive number", sharesStr)
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil || price <= 0 {
		return fmt.Errorf("invalid price value: %s — must be a positive number", priceStr)
	}

	log.Info("trade_start", ticker, map[string]any{
		"shares": shares, "price": price,
	})

	// Read current holding
	sh, err := sheets.New()
	if err != nil {
		log.Error("sheets_init", ticker, err)
		return err
	}

	holding, err := sh.ReadHolding(ticker)
	if err != nil {
		log.Error("read_holding", ticker, err)
		return err
	}

	// Calculate new average cost
	// Formula: (existing_shares * existing_avg + new_shares * buy_price) / total_shares
	prevShares := holding.SharesHeld
	prevAvg := holding.AvgBuyPrice

	newShares := prevShares + shares
	newAvg := 0.0
	if newShares > 0 {
		newAvg = ((prevShares * prevAvg) + (shares * price)) / newShares
	}
	newAvg = math.Round(newAvg*100) / 100 // round to 2 decimal places

	totalCost := shares * price
	pnlPct := 0.0
	if prevAvg > 0 {
		pnlPct = (price - prevAvg) / prevAvg * 100
	}

	// Write back to Holdings sheet
	if err := sh.UpdateHolding(ticker, newShares, newAvg); err != nil {
		log.Error("update_holding", ticker, err)
		return err
	}

	result := TradeResult{
		Ticker:       ticker,
		Action:       "BUY",
		SharesTraded: shares,
		TradePrice:   price,
		TotalCost:    totalCost,
		PrevShares:   prevShares,
		PrevAvg:      prevAvg,
		NewShares:    newShares,
		NewAvg:       newAvg,
		PnLPct:       pnlPct,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
	}

	log.Info("trade_complete", ticker, map[string]any{
		"prev_shares": prevShares, "prev_avg": prevAvg,
		"new_shares": newShares, "new_avg": newAvg,
		"total_cost": totalCost,
	})

	// Send confirmation email
	gc := gmail.New()
	emailErr := gc.Send(
		fmt.Sprintf("✅ Trade Recorded: %s — %d shares @ KSh%.2f", ticker, int(shares), price),
		fmt.Sprintf(`
<h2>✅ Trade Recorded Successfully</h2>
<table style="border-collapse:collapse;width:100%%">
  <tr><td><b>Ticker</b></td><td>%s</td></tr>
  <tr><td><b>Shares bought</b></td><td>%.0f</td></tr>
  <tr><td><b>Price per share</b></td><td>KSh %.2f</td></tr>
  <tr><td><b>Total cost</b></td><td>KSh %.2f</td></tr>
  <tr><td><b>Previous position</b></td><td>%.0f shares @ KSh%.2f avg</td></tr>
  <tr><td><b>New position</b></td><td>%.0f shares @ KSh%.2f avg</td></tr>
  <tr><td><b>Bought vs avg cost</b></td><td>%.2f%%</td></tr>
  <tr><td><b>Recorded at</b></td><td>%s</td></tr>
</table>
<p><i>Holdings sheet updated automatically.</i></p>
`,
			result.Ticker, result.SharesTraded, result.TradePrice, result.TotalCost,
			result.PrevShares, result.PrevAvg,
			result.NewShares, result.NewAvg,
			result.PnLPct, result.Timestamp,
		),
	)
	if emailErr != nil {
		log.Error("send_email", ticker, emailErr)
	} else {
		log.Info("email_sent", ticker, nil)
	}

	outJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(outJSON))

	fmt.Printf("\n✓ Holdings updated: %s now %.0f shares @ KSh%.2f avg\n",
		ticker, newShares, newAvg)

	return nil
}

// parseArgs parses --key value pairs from command line arguments.
func parseArgs(args []string) map[string]string {
	params := make(map[string]string)
	for i := 0; i < len(args)-1; i++ {
		if strings.HasPrefix(args[i], "--") {
			key := strings.TrimPrefix(args[i], "--")
			params[key] = args[i+1]
			i++ // skip the value
		}
	}
	return params
}
