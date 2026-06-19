# NSE Stock Trading Intelligence System v3
### Complete · Gap-Free · Go · Gmail Notifications · Google Sheets · DCA Engine · Groq AI · n8n Orchestration

---

## What's Fixed in v3

Every gap from v2 is now closed:

| Gap | Fix |
|-----|-----|
| `internal/models/` empty | Fully defined `models` package with all shared types |
| `internal/sheets/sheets.go` missing | Complete Google Sheets client with OAuth2 service account |
| `cmd/alerts/alerts.go` empty | Full price alert engine with Gmail notifications |
| `cmd/rebalance/rebalance.go` empty | Full portfolio drift analysis and rebalance engine |
| `.env` — no setup walkthrough | Step-by-step instructions for every key |
| `internal/mansa/` missing models import | Removed — models now live in `internal/models/` |
| Google Sheets OAuth2 for Go | Full walkthrough: service account credentials, scopes, token |
| `.env.example` never shown | Written out in full |
| `go.mod` incomplete | Complete with all required dependencies |
| n8n runs Go inside Docker | Full solution: compile binary, mount host path in Docker |
| Telegram → Gmail | All notifications use Gmail via Go's `net/smtp` |

---

## Architecture Overview

```
VSCode (main.go — your single entry point)
    │
    ├── cmd/scanner/     — scrapes NSE prices via Mansa API
    ├── cmd/sentiment/   — fetches and scores news via Groq
    ├── cmd/dca/         — calculates DCA buy amounts
    ├── cmd/rebalance/   — portfolio drift analysis
    ├── cmd/alerts/      — price threshold alerts via Gmail
    ├── internal/logger/ — structured JSON logging (all steps)
    ├── internal/sheets/ — Google Sheets read/write (OAuth2 service account)
    ├── internal/groq/   — Groq API client (free LLM calls)
    ├── internal/mansa/  — Mansa Markets API client
    ├── internal/gmail/  — Gmail SMTP notification client
    └── internal/models/ — shared data types used across packages

n8n (Docker, same machine)
    ├── Workflow 1: Schedule → runs compiled Go binary → parses JSON
    ├── Workflow 2: Schedule → runs Go sentiment engine
    ├── Workflow 3: Monday trigger → runs Go DCA + buy list
    └── Workflow 4: Friday trigger → runs Go rebalancer
```

n8n's role is **scheduling only**. The Go binary handles all logic, data, and email. n8n invokes the pre-compiled binary (not `go run`) so it works inside Docker.

---

## Part 1 — Project Setup

### Install Go on Linux Mint

```bash
wget https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz

echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
source ~/.bashrc

go version   # → go version go1.22.5 linux/amd64
```

### Install VSCode Extensions

Open VSCode → Extensions (`Ctrl+Shift+X`) → install:
- **Go** (by Google) — IntelliSense, debugging, test runner
- **REST Client** — test your API calls from inside VSCode

### Create the Project

```bash
mkdir ~/nse-trader && cd ~/nse-trader
go mod init nse-trader

mkdir -p cmd/{scanner,sentiment,dca,rebalance,alerts}
mkdir -p internal/{logger,sheets,groq,mansa,models,gmail}
mkdir -p logs bin

code .
```

### Complete Directory Layout

```
nse-trader/
├── main.go
├── go.mod
├── go.sum
├── .env                         ← your secrets (never commit)
├── .env.example                 ← template to share
├── credentials.json             ← Google service account key (never commit)
├── bin/
│   └── nse-trader               ← compiled binary (for n8n to call)
├── .vscode/
│   └── launch.json
├── cmd/
│   ├── scanner/scanner.go
│   ├── sentiment/sentiment.go
│   ├── dca/dca.go
│   ├── rebalance/rebalance.go
│   └── alerts/alerts.go
├── internal/
│   ├── logger/logger.go
│   ├── sheets/sheets.go
│   ├── groq/groq.go
│   ├── mansa/mansa.go
│   ├── gmail/gmail.go
│   └── models/models.go
└── logs/
    └── trading.jsonl
```

---

## Part 2 — API Keys: How to Get Each One

### 2.1 Mansa Markets API Key

1. Go to **mansamarkets.com** → click **Developers** in the top nav
2. Create a free account (email + password, no card)
3. Go to **Dashboard → API Keys → Generate New Key**
4. Copy the key — it starts with `mansa_live_sk_`

Free tier: 500 requests/day. More than enough for daily scanning.

### 2.2 Groq API Key

1. Go to **console.groq.com**
2. Sign in with Google or email
3. Click **API Keys** in the left sidebar → **Create API Key**
4. Copy the key — it starts with `gsk_`

Free limits: 14,400 requests/day on the fast model, 1,000/day on the quality model. The sentiment engine uses ~5–20 requests per morning run.

### 2.3 Google Sheets — Service Account Setup

You need a **service account** — a bot identity that your Go program uses to read and write Sheets without a browser login popup.

**Step 1: Create a Google Cloud Project**

1. Go to **console.cloud.google.com**
2. Click the project dropdown at the top → **New Project**
3. Name it `nse-trader` → **Create**
4. Make sure this project is selected in the dropdown

**Step 2: Enable the Google Sheets API**

1. Left sidebar: **APIs & Services → Library**
2. Search `Google Sheets API` → click it → **Enable**
3. Also search `Google Drive API` → **Enable**

**Step 3: Create a Service Account**

1. Left sidebar: **APIs & Services → Credentials**
2. **+ Create Credentials → Service Account**
3. Name: `nse-trader-bot` → **Create and Continue**
4. Role: `Editor` → **Continue → Done**

**Step 4: Download the JSON Key**

1. Click on the service account you just created
2. Tab: **Keys → Add Key → Create New Key → JSON**
3. The file downloads automatically
4. Rename it to `credentials.json` and place it in `~/nse-trader/`
5. **Never commit this file.** Add `credentials.json` to `.gitignore`

**Step 5: Create Your Google Sheet**

1. Go to **sheets.google.com → Blank spreadsheet**
2. Name it `NSE Trading Log`
3. Create these tabs (right-click Sheet1 → Rename):
   - `Holdings`
   - `DailyPrices`
   - `Sentiment`
   - `DCA Plan`
   - `Alerts`
   - `Rebalance`

**Step 6: Share the Sheet with Your Service Account**

1. Open the Sheet → **Share** button
2. Add the service account email — visible in `credentials.json` under `client_email`, looks like `nse-trader-bot@nse-trader-XXXXX.iam.gserviceaccount.com`
3. Set permission to **Editor** → **Send**

**Step 7: Get the Sheet ID**

```
https://docs.google.com/spreadsheets/d/THIS_IS_YOUR_SHEET_ID/edit
```

Copy everything between `/d/` and `/edit`. That goes in your `.env` as `GOOGLE_SHEETS_ID`.

### 2.4 Gmail App Password

Gmail notifications use SMTP with an App Password — a special password just for this app.

**You need 2-Step Verification enabled on your Google account first.**

1. Go to **myaccount.google.com → Security**
2. Confirm **2-Step Verification** is ON
3. Go to **myaccount.google.com/apppasswords**
4. Click **Create** → name it `NSE Trader` → **Create**
5. Google shows a 16-character password like `abcd efgh ijkl mnop`
6. Copy it — you won't see it again. Remove spaces before pasting to `.env`

---

## Part 3 — Environment Files

### `.env.example` (commit this to git)

```bash
# .env.example — copy to .env and fill in real values
# See Part 2 of this guide for how to get each key

MANSA_API_KEY=mansa_live_sk_your_key_here
GROQ_API_KEY=gsk_your_groq_key_here
GOOGLE_SHEETS_ID=your_sheet_id_here_from_the_url

# Gmail notifications
GMAIL_FROM=your.email@gmail.com
GMAIL_APP_PASSWORD=your16charapppasswordnospaces
GMAIL_TO=your.email@gmail.com

# Google service account credentials file path
GOOGLE_CREDENTIALS_FILE=./credentials.json

# Log file path
LOG_FILE=./logs/trading.jsonl
```

### `.env` (never commit)

```bash
cp .env.example .env
# Then open .env and fill in real values
```

### `.gitignore`

```bash
cat > .gitignore << 'EOF'
.env
credentials.json
bin/
logs/
*.jsonl
EOF
```

---

## Part 4 — go.mod (Complete)

```go
module nse-trader

go 1.22

require (
    github.com/joho/godotenv v1.5.1
    golang.org/x/oauth2 v0.21.0
    google.golang.org/api v0.188.0
)
```

Install all dependencies:

```bash
cd ~/nse-trader

go get github.com/joho/godotenv
go get google.golang.org/api/sheets/v4
go get google.golang.org/api/option
go get golang.org/x/oauth2

go mod tidy

# Verify everything compiles
go build ./...
```

If `go build ./...` produces no output, everything is correct.

---

## Part 5 — internal/models/models.go

All shared data structures live here. Every other package imports this — no more circular dependencies, no more missing types.

**`internal/models/models.go`**

```go
package models

// Holding is one stock position in your portfolio.
// Populated from the "Holdings" tab of your Google Sheet.
// Columns: Ticker | Name | SharesHeld | AvgBuyPrice | TargetPct | WeeklyBudget
type Holding struct {
	Ticker       string  `json:"ticker"`
	Name         string  `json:"name"`
	SharesHeld   float64 `json:"shares_held"`
	AvgBuyPrice  float64 `json:"avg_buy_price"`
	TargetPct    float64 `json:"target_pct"`    // desired % of total portfolio
	WeeklyBudget float64 `json:"weekly_budget"` // KES to invest weekly in this stock
}

// AlertRule is one row in your "Alerts" watchlist tab.
// Columns: Ticker | AlertAbove | AlertBelow | Label
// AlertAbove: send email if price rises above this (0 = disabled)
// AlertBelow: send email if price falls below this (0 = disabled)
type AlertRule struct {
	Ticker     string  `json:"ticker"`
	AlertAbove float64 `json:"alert_above"`
	AlertBelow float64 `json:"alert_below"`
	Label      string  `json:"label"` // friendly name, e.g. "Safaricom"
}

// DCARecommendation is the output of the DCA engine for one holding.
type DCARecommendation struct {
	Ticker         string  `json:"ticker"`
	CurrentPrice   float64 `json:"current_price"`
	AvgBuyPrice    float64 `json:"avg_buy_price"`
	CurrentPnLPct  float64 `json:"current_pnl_pct"`
	DipPct         float64 `json:"dip_from_avg_pct"` // negative = below avg cost
	BaseWeeklyKES  float64 `json:"base_weekly_kes"`
	AdjustedKES    float64 `json:"adjusted_weekly_kes"`
	SharesToBuy    float64 `json:"shares_to_buy"`
	NewAvgAfterBuy float64 `json:"new_avg_after_buy"`
	Reason         string  `json:"reason"`
	Action         string  `json:"action"` // BUY | SKIP | STRONG_BUY
}

// ScanResult is one row of daily price scanner output.
type ScanResult struct {
	Ticker    string  `json:"ticker"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	ChangePct float64 `json:"change_pct"`
	Volume    int64   `json:"volume"`
	Signal    string  `json:"signal"` // NORMAL | UP | DOWN | STRONG_UP | STRONG_DOWN
	Date      string  `json:"date"`
}

// SentimentResult is the AI-scored output for one news article.
type SentimentResult struct {
	Ticker     string  `json:"ticker"`
	Company    string  `json:"company"`
	Sentiment  string  `json:"sentiment"`  // Bullish | Bearish | Neutral
	Confidence float64 `json:"confidence"` // 1-10
	Reason     string  `json:"reason"`
	Headline   string  `json:"headline"`
	Link       string  `json:"link"`
	Date       string  `json:"date"`
}

// RebalanceAction is one recommended portfolio adjustment from the rebalancer.
type RebalanceAction struct {
	Ticker       string  `json:"ticker"`
	CurrentPct   float64 `json:"current_pct"`
	TargetPct    float64 `json:"target_pct"`
	DriftPct     float64 `json:"drift_pct"`     // current - target
	CurrentValue float64 `json:"current_value"` // KES
	TargetValue  float64 `json:"target_value"`  // KES
	Action       string  `json:"action"`        // BUY_MORE | TRIM | HOLD
	KESAmount    float64 `json:"kes_amount"`
	Shares       float64 `json:"shares"`
	Reason       string  `json:"reason"`
}
```

---

## Part 6 — Google Sheets Tab Layout

Before running the code, set up your tabs so the Go client can parse rows correctly.

### Holdings Tab

Row 1 is the header (the client skips it automatically):

| A | B | C | D | E | F |
|---|---|---|---|---|---|
| Ticker | Name | SharesHeld | AvgBuyPrice | TargetPct | WeeklyBudget |
| SCOM | Safaricom | 500 | 31.50 | 25 | 5000 |
| EQTY | Equity Group | 200 | 48.00 | 20 | 4000 |
| KCB | KCB Group | 300 | 38.00 | 15 | 3000 |

`TargetPct` is your desired allocation as a percentage. All rows should add up to 100.

`WeeklyBudget` is the KES amount to invest in this stock per week under normal DCA conditions. The engine adjusts this up or down based on current price vs your average.

### Alerts Tab

| A | B | C | D |
|---|---|---|---|
| Ticker | AlertAbove | AlertBelow | Label |
| SCOM | 40 | 25 | Safaricom |
| EQTY | 0 | 40 | Equity Group |
| KCB | 45 | 30 | KCB Group |

`0` in AlertAbove or AlertBelow means that threshold is disabled for that ticker.

---

## Part 7 — internal/sheets/sheets.go

The complete Google Sheets client using a service account. No browser popup, no user login — works headlessly.

**`internal/sheets/sheets.go`**

```go
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

	"nse-trader/internal/models"
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
	for i, row := range resp.Values {
		if len(row) < 4 {
			return nil, fmt.Errorf("holdings row %d: expected at least 4 columns, got %d", i+2, len(row))
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
```

---

## Part 8 — internal/gmail/gmail.go

All notifications go through your Gmail account using SMTP and an App Password. No third-party email service needed.

**`internal/gmail/gmail.go`**

```go
package gmail

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
	"time"
)

type Client struct {
	from     string
	password string
	to       string
}

func New() *Client {
	return &Client{
		from:     os.Getenv("GMAIL_FROM"),
		password: os.Getenv("GMAIL_APP_PASSWORD"),
		to:       os.Getenv("GMAIL_TO"),
	}
}

// Send sends an HTML email via Gmail SMTP.
func (c *Client) Send(subject, htmlBody string) error {
	if c.from == "" || c.password == "" || c.to == "" {
		return fmt.Errorf("gmail: GMAIL_FROM, GMAIL_APP_PASSWORD, and GMAIL_TO must all be set in .env")
	}

	auth := smtp.PlainAuth("", c.from, c.password, "smtp.gmail.com")

	headers := strings.Join([]string{
		"From: NSE Trader <" + c.from + ">",
		"To: " + c.to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
	}, "\r\n")

	body := headers + "\r\n\r\n" + htmlBody

	return smtp.SendMail("smtp.gmail.com:587", auth, c.from, []string{c.to}, []byte(body))
}

// ScannerDigest formats and sends the daily strong-moves email.
func (c *Client) ScannerDigest(date string, strongMoves []map[string]interface{}) error {
	subject := fmt.Sprintf("📈 NSE Strong Moves — %s", date)

	if len(strongMoves) == 0 {
		return c.Send(subject, "<p>No strong moves today (±3%+ threshold not reached). Markets quiet.</p>")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<h2>NSE Strong Moves — %s</h2>\n<ul>\n", date))

	for _, m := range strongMoves {
		ticker := fmt.Sprintf("%v", m["ticker"])
		price := fmt.Sprintf("%v", m["price"])
		changePct := fmt.Sprintf("%v", m["change_pct"])
		signal := fmt.Sprintf("%v", m["signal"])
		volume := fmt.Sprintf("%v", m["volume"])

		emoji := "🟢"
		if signal == "STRONG_DOWN" || signal == "DOWN" {
			emoji = "🔴"
		}
		sb.WriteString(fmt.Sprintf(
			"  <li>%s <b>%s</b> — %s%% | KSh %s | Vol %s</li>\n",
			emoji, ticker, changePct, price, volume,
		))
	}
	sb.WriteString("</ul>\n<p><i>Review your DCA plan if any of your holdings appear above.</i></p>")
	return c.Send(subject, sb.String())
}

// DCAReport formats and sends the Monday DCA email.
func (c *Client) DCAReport(date string, recs []map[string]interface{}, commentary string) error {
	subject := fmt.Sprintf("📊 Weekly DCA Plan — %s", date)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<h2>Weekly DCA Plan — %s</h2>\n", date))
	sb.WriteString(fmt.Sprintf("<h3>AI Commentary</h3>\n<p>%s</p>\n<hr>\n", commentary))
	sb.WriteString("<h3>Recommendations</h3>\n<ul>\n")

	for _, r := range recs {
		action := fmt.Sprintf("%v", r["action"])
		if action == "SKIP" {
			continue
		}
		emoji := "✅"
		if action == "STRONG_BUY" {
			emoji = "🔥"
		}
		sb.WriteString(fmt.Sprintf(
			"  <li>%s <b>%v</b> — %s %v shares @ KSh%v | New avg: KSh%v</li>\n",
			emoji, r["ticker"], action, r["shares_to_buy"], r["current_price"], r["new_avg_after_buy"],
		))
	}
	sb.WriteString("</ul>\n<p><i>Call your broker to place orders. Update Holdings sheet after purchase.</i></p>")
	return c.Send(subject, sb.String())
}

// RebalanceReport formats and sends the Friday rebalance email.
func (c *Client) RebalanceReport(date string, actions []map[string]interface{}, commentary string) error {
	subject := fmt.Sprintf("⚖️ Portfolio Rebalance Report — %s", date)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<h2>Portfolio Rebalance — %s</h2>\n", date))
	sb.WriteString(fmt.Sprintf("<h3>AI Commentary</h3>\n<p>%s</p>\n<hr>\n", commentary))
	sb.WriteString("<h3>Actions Required</h3>\n<ul>\n")

	for _, a := range actions {
		action := fmt.Sprintf("%v", a["action"])
		if action == "HOLD" {
			continue
		}
		emoji := "➕"
		if action == "TRIM" {
			emoji = "✂️"
		}
		sb.WriteString(fmt.Sprintf(
			"  <li>%s <b>%v</b> — %s KSh%v | Drift: %v → %v</li>\n",
			emoji, a["ticker"], action, a["kes_amount"], a["current_pct"], a["target_pct"],
		))
	}
	sb.WriteString("</ul>")
	return c.Send(subject, sb.String())
}

// PriceAlert sends an immediate alert for a threshold breach.
func (c *Client) PriceAlert(ticker, label string, currentPrice, threshold float64, direction string) error {
	date := time.Now().Format("02 Jan 2006 15:04")
	subject := fmt.Sprintf("🚨 Price Alert: %s %s KSh%.2f — %s", ticker, direction, currentPrice, date)

	body := fmt.Sprintf(`
<h2>🚨 Price Alert Triggered</h2>
<p><b>%s</b> (%s)</p>
<p>Current price: <b>KSh %.2f</b></p>
<p>Alert threshold: KSh %.2f (%s)</p>
<p>Time: %s</p>
<p><i>Log in to your broker to review your position.</i></p>
`, label, ticker, currentPrice, threshold, direction, date)

	return c.Send(subject, body)
}
```

---

## Part 9 — internal/logger/logger.go

**`internal/logger/logger.go`**

```go
package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type LogEntry struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Command   string         `json:"command"`
	Step      string         `json:"step"`
	Ticker    string         `json:"ticker,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	Error     string         `json:"error,omitempty"`
}

type Logger struct {
	file    *os.File
	mu      sync.Mutex
	command string
}

func New(command string) (*Logger, error) {
	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		logFile = "./logs/trading.jsonl"
	}
	os.MkdirAll("./logs", 0755)

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("cannot open log file: %w", err)
	}

	return &Logger{file: f, command: command}, nil
}

func (l *Logger) log(level, step, ticker string, data map[string]any, err error) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Command:   l.command,
		Step:      step,
		Ticker:    ticker,
		Data:      data,
	}
	if err != nil {
		entry.Error = err.Error()
	}

	b, _ := json.Marshal(entry)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.file.Write(b)
	l.file.Write([]byte("\n"))
	fmt.Printf("[%s] %s | %s | %s\n", level, l.command, step, ticker)
}

func (l *Logger) Info(step, ticker string, data map[string]any)  { l.log("INFO", step, ticker, data, nil) }
func (l *Logger) Error(step, ticker string, err error)            { l.log("ERROR", step, ticker, nil, err) }
func (l *Logger) Warn(step, ticker string, data map[string]any)  { l.log("WARN", step, ticker, data, nil) }
func (l *Logger) Close()                                          { l.file.Close() }

func (l *Logger) TimedStep(step, ticker string, fn func() (map[string]any, error)) {
	start := time.Now()
	data, err := fn()
	duration := fmt.Sprintf("%dms", time.Since(start).Milliseconds())
	if data == nil {
		data = map[string]any{}
	}
	data["duration_ms"] = duration
	if err != nil {
		l.Error(step, ticker, err)
	} else {
		l.Info(step, ticker, data)
	}
}
```

---

## Part 10 — internal/groq/groq.go

**`internal/groq/groq.go`**

```go
package groq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	BaseURL      = "https://api.groq.com/openai/v1/chat/completions"
	ModelFast    = "llama-3.1-8b-instant"
	ModelQuality = "llama-3.3-70b-versatile"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

type Response struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func New() *Client {
	return &Client{
		apiKey:     os.Getenv("GROQ_API_KEY"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Complete(model, system, user string, maxTokens int) (string, error) {
	req := Request{
		Model:     model,
		MaxTokens: maxTokens,
		Messages: []Message{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequest("POST", BaseURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var result Response
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("groq api error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return result.Choices[0].Message.Content, nil
}
```

---

## Part 11 — internal/mansa/mansa.go

**`internal/mansa/mansa.go`**

```go
package mansa

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const BaseURL = "https://www.mansaapi.com/api/v1"

type Stock struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	ChangePct float64 `json:"change_percent"`
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

	return json.NewDecoder(resp.Body).Decode(target)
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
```

---

## Part 12 — cmd/scanner/scanner.go

**`cmd/scanner/scanner.go`**

```go
package scanner

import (
	"encoding/json"
	"fmt"
	"time"

	"nse-trader/internal/gmail"
	"nse-trader/internal/logger"
	"nse-trader/internal/mansa"
	"nse-trader/internal/models"
	"nse-trader/internal/sheets"
)

const MinVolume = 500_000

func Run() error {
	log, err := logger.New("scanner")
	if err != nil {
		return err
	}
	defer log.Close()

	log.Info("scan_start", "ALL", map[string]any{"min_volume": MinVolume})

	mc := mansa.New()
	var stocks []mansa.Stock

	log.TimedStep("fetch_all_stocks", "NSE", func() (map[string]any, error) {
		var fetchErr error
		stocks, fetchErr = mc.AllStocks()
		return map[string]any{"total_fetched": len(stocks)}, fetchErr
	})

	results := make([]models.ScanResult, 0)
	strongMoves := make([]models.ScanResult, 0)

	for _, s := range stocks {
		if s.Volume < MinVolume {
			continue
		}

		signal := "NORMAL"
		switch {
		case s.ChangePct >= 5:
			signal = "STRONG_UP"
		case s.ChangePct <= -5:
			signal = "STRONG_DOWN"
		case s.ChangePct >= 3:
			signal = "UP"
		case s.ChangePct <= -3:
			signal = "DOWN"
		}

		r := models.ScanResult{
			Ticker:    s.Symbol,
			Name:      s.Name,
			Price:     s.Price,
			ChangePct: s.ChangePct,
			Volume:    s.Volume,
			Signal:    signal,
			Date:      time.Now().Format("2006-01-02"),
		}
		results = append(results, r)

		if signal == "STRONG_UP" || signal == "STRONG_DOWN" {
			strongMoves = append(strongMoves, r)
		}

		log.Info("stock_scanned", s.Symbol, map[string]any{
			"price": s.Price, "change_pct": s.ChangePct,
			"volume": s.Volume, "signal": signal,
		})
	}

	sh, err := sheets.New()
	if err != nil {
		log.Error("sheets_init", "ALL", err)
		return err
	}
	if err := sh.AppendDailyPrices(results); err != nil {
		log.Error("write_to_sheets", "ALL", err)
		return err
	}

	// Email digest for strong moves (only fires when something notable happened)
	if len(strongMoves) > 0 {
		gc := gmail.New()
		movesAsMap := make([]map[string]interface{}, 0, len(strongMoves))
		for _, m := range strongMoves {
			movesAsMap = append(movesAsMap, map[string]interface{}{
				"ticker": m.Ticker, "price": m.Price,
				"change_pct": m.ChangePct, "volume": m.Volume, "signal": m.Signal,
			})
		}
		date := time.Now().Format("02 Jan 2006")
		if emailErr := gc.ScannerDigest(date, movesAsMap); emailErr != nil {
			log.Error("send_email", "ALL", emailErr)
		} else {
			log.Info("email_sent", "ALL", map[string]any{"strong_moves": len(strongMoves)})
		}
	}

	log.Info("scan_complete", "ALL", map[string]any{
		"total_scanned": len(results), "strong_moves": len(strongMoves),
	})

	output := map[string]any{
		"results":      results,
		"strong_moves": strongMoves,
		"scan_time":    time.Now().Format(time.RFC3339),
	}
	outJSON, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(outJSON))

	return nil
}
```

---

## Part 13 — cmd/sentiment/sentiment.go

**`cmd/sentiment/sentiment.go`**

```go
package sentiment

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"nse-trader/internal/groq"
	"nse-trader/internal/logger"
	"nse-trader/internal/models"
	"nse-trader/internal/sheets"
)

var NewsSources = []string{
	"https://thekenyawallstreet.com/feed/",
	"https://businessdailyafrica.com/rss",
	"https://mwangocapital.substack.com/feed",
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

type RSS struct {
	Items []RSSItem `xml:"channel>item"`
}

type GroqSentimentResponse struct {
	Ticker     string  `json:"ticker"`
	Company    string  `json:"company"`
	Sentiment  string  `json:"sentiment"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

func Run() error {
	log, err := logger.New("sentiment")
	if err != nil {
		return err
	}
	defer log.Close()

	log.Info("session_start", "", nil)

	allItems := []RSSItem{}
	for _, url := range NewsSources {
		items, fetchErr := fetchRSS(url)
		if fetchErr != nil {
			log.Error("fetch_rss", url, fetchErr)
			continue
		}
		log.Info("rss_fetched", url, map[string]any{"items": len(items)})
		allItems = append(allItems, items...)
	}

	cutoff := time.Now().Add(-24 * time.Hour)
	recent := []RSSItem{}
	for _, item := range allItems {
		t, _ := time.Parse(time.RFC1123Z, item.PubDate)
		if t.After(cutoff) {
			recent = append(recent, item)
		}
	}
	log.Info("articles_filtered", "", map[string]any{
		"total": len(allItems), "recent_24h": len(recent),
	})

	gc := groq.New()
	results := []models.SentimentResult{}

	for _, item := range recent {
		var sr models.SentimentResult

		log.TimedStep("score_article", trunc(item.Title, 40), func() (map[string]any, error) {
			prompt := fmt.Sprintf(
				"Headline: %s\n\nSummary: %s",
				item.Title, trunc(strings.TrimSpace(item.Description), 500),
			)

			raw, callErr := gc.Complete(
				groq.ModelFast,
				`You are an NSE Kenya stock analyst. Given a news headline and summary, identify which NSE-listed company or sector is affected.
Return ONLY valid JSON, no other text: {"ticker": "SCOM or null", "company": "Company Name or Sector", "sentiment": "Bullish|Bearish|Neutral", "confidence": 7, "reason": "one sentence"}
If no specific company is identifiable, set ticker to null and company to the affected sector.`,
				prompt,
				200,
			)
			if callErr != nil {
				return nil, callErr
			}

			var parsed GroqSentimentResponse
			clean := strings.TrimSpace(strings.ReplaceAll(raw, "```json", ""))
			clean = strings.ReplaceAll(clean, "```", "")
			if jsonErr := json.Unmarshal([]byte(clean), &parsed); jsonErr != nil {
				return nil, fmt.Errorf("parse json: %w (raw: %s)", jsonErr, trunc(raw, 100))
			}

			sr = models.SentimentResult{
				Ticker:     parsed.Ticker,
				Company:    parsed.Company,
				Sentiment:  parsed.Sentiment,
				Confidence: parsed.Confidence,
				Reason:     parsed.Reason,
				Headline:   item.Title,
				Link:       item.Link,
				Date:       time.Now().Format("2006-01-02"),
			}

			return map[string]any{
				"ticker": parsed.Ticker, "sentiment": parsed.Sentiment,
			}, nil
		})

		if sr.Ticker != "" || sr.Company != "" {
			results = append(results, sr)
		}
	}

	sh, err := sheets.New()
	if err != nil {
		log.Error("sheets_init", "", err)
		return err
	}
	if writeErr := sh.AppendSentiment(results); writeErr != nil {
		log.Error("write_sentiment", "", writeErr)
		return writeErr
	}

	log.Info("session_complete", "", map[string]any{"articles_scored": len(results)})

	outJSON, _ := json.MarshalIndent(results, "", "  ")
	fmt.Println(string(outJSON))
	return nil
}

func fetchRSS(url string) ([]RSSItem, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var rss RSS
	if err := xml.Unmarshal(body, &rss); err != nil {
		return nil, err
	}
	return rss.Items, nil
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
```

---

## Part 14 — cmd/dca/dca.go

**`cmd/dca/dca.go`**

```go
package dca

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"nse-trader/internal/gmail"
	"nse-trader/internal/groq"
	"nse-trader/internal/logger"
	"nse-trader/internal/mansa"
	"nse-trader/internal/models"
	"nse-trader/internal/sheets"
)

func Run() error {
	log, err := logger.New("dca")
	if err != nil {
		return err
	}
	defer log.Close()

	log.Info("session_start", "", map[string]any{"time": time.Now().Format(time.RFC3339)})

	sh, err := sheets.New()
	if err != nil {
		log.Error("sheets_init", "", err)
		return err
	}

	holdings, err := sh.ReadHoldings()
	if err != nil {
		log.Error("load_holdings", "", err)
		return err
	}
	log.Info("holdings_loaded", "", map[string]any{"count": len(holdings)})

	mc := mansa.New()
	recommendations := make([]models.DCARecommendation, 0, len(holdings))

	for _, h := range holdings {
		var rec models.DCARecommendation

		log.TimedStep("fetch_price", h.Ticker, func() (map[string]any, error) {
			stock, err := mc.SingleStock(h.Ticker)
			if err != nil {
				return nil, err
			}

			pnlPct := (stock.Price - h.AvgBuyPrice) / h.AvgBuyPrice * 100
			dipPct := pnlPct

			multiplier := 1.0
			switch {
			case dipPct <= -15:
				multiplier = 2.5
			case dipPct <= -10:
				multiplier = 2.0
			case dipPct <= -5:
				multiplier = 1.5
			case dipPct <= 5:
				multiplier = 1.0
			case dipPct <= 15:
				multiplier = 0.75
			default:
				multiplier = 0.5
			}

			adjustedKES := h.WeeklyBudget * multiplier
			sharesToBuy := math.Floor(adjustedKES / stock.Price)

			totalCost := (h.SharesHeld * h.AvgBuyPrice) + (sharesToBuy * stock.Price)
			totalShares := h.SharesHeld + sharesToBuy
			newAvg := 0.0
			if totalShares > 0 {
				newAvg = totalCost / totalShares
			}

			action := "BUY"
			if multiplier >= 2.0 {
				action = "STRONG_BUY"
			} else if sharesToBuy < 1 {
				action = "SKIP"
			}

			label := "above avg"
			if dipPct < 0 {
				label = "below avg"
			}
			reason := fmt.Sprintf(
				"Price KSh%.2f vs avg KSh%.2f (%.1f%% %s). DCA %.1fx → buy %d shares → new avg KSh%.2f",
				stock.Price, h.AvgBuyPrice, math.Abs(dipPct), label,
				multiplier, int(sharesToBuy), newAvg,
			)

			rec = models.DCARecommendation{
				Ticker:         h.Ticker,
				CurrentPrice:   stock.Price,
				AvgBuyPrice:    h.AvgBuyPrice,
				CurrentPnLPct:  pnlPct,
				DipPct:         dipPct,
				BaseWeeklyKES:  h.WeeklyBudget,
				AdjustedKES:    adjustedKES,
				SharesToBuy:    sharesToBuy,
				NewAvgAfterBuy: newAvg,
				Reason:         reason,
				Action:         action,
			}

			return map[string]any{
				"price": stock.Price, "dip_pct": dipPct,
				"multiplier": multiplier, "action": action,
			}, nil
		})

		recommendations = append(recommendations, rec)
	}

	gc := groq.New()
	recJSON, _ := json.MarshalIndent(recommendations, "", "  ")
	var commentary string

	log.TimedStep("ai_dca_commentary", "ALL", func() (map[string]any, error) {
		var callErr error
		commentary, callErr = gc.Complete(
			groq.ModelQuality,
			`You are an NSE Kenya portfolio analyst focused on Dollar-Cost Averaging.
Be direct and specific. Use KES amounts.
Cover: (1) which STRONG_BUY positions deserve immediate action and why, (2) any positions to pause DCA on, (3) projected impact on overall portfolio average cost after this week's purchases.
Keep under 250 words.`,
			fmt.Sprintf("This week's DCA recommendations:\n%s", string(recJSON)),
			500,
		)
		return map[string]any{"commentary_length": len(commentary)}, callErr
	})

	if err := sh.WriteDCAPlan(recommendations, commentary); err != nil {
		log.Error("write_dca_plan", "", err)
		return err
	}

	gc2 := gmail.New()
	recsAsMap := make([]map[string]interface{}, 0, len(recommendations))
	for _, r := range recommendations {
		recsAsMap = append(recsAsMap, map[string]interface{}{
			"ticker": r.Ticker, "action": r.Action,
			"shares_to_buy": r.SharesToBuy, "current_price": r.CurrentPrice,
			"new_avg_after_buy": r.NewAvgAfterBuy,
		})
	}
	date := time.Now().Format("02 Jan 2006")
	if emailErr := gc2.DCAReport(date, recsAsMap, commentary); emailErr != nil {
		log.Error("send_email", "", emailErr)
	} else {
		log.Info("email_sent", "", map[string]any{"report": "dca"})
	}

	output := map[string]any{
		"recommendations": recommendations,
		"commentary":      commentary,
		"generated_at":    time.Now().Format(time.RFC3339),
	}
	outJSON, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(outJSON))

	log.Info("session_complete", "", map[string]any{"count": len(recommendations)})
	return nil
}
```

---

## Part 15 — cmd/rebalance/rebalance.go (Complete)

**`cmd/rebalance/rebalance.go`**

```go
package rebalance

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"nse-trader/internal/gmail"
	"nse-trader/internal/groq"
	"nse-trader/internal/logger"
	"nse-trader/internal/mansa"
	"nse-trader/internal/models"
	"nse-trader/internal/sheets"
)

// DriftThreshold: only flag a position if it has drifted more than this
// percentage from its target allocation. 5% is a sensible default.
const DriftThreshold = 5.0

func Run() error {
	log, err := logger.New("rebalance")
	if err != nil {
		return err
	}
	defer log.Close()

	log.Info("session_start", "", map[string]any{"time": time.Now().Format(time.RFC3339)})

	sh, err := sheets.New()
	if err != nil {
		log.Error("sheets_init", "", err)
		return err
	}

	holdings, err := sh.ReadHoldings()
	if err != nil {
		log.Error("load_holdings", "", err)
		return err
	}
	log.Info("holdings_loaded", "", map[string]any{"count": len(holdings)})

	mc := mansa.New()

	type positionValue struct {
		holding models.Holding
		price   float64
		value   float64
	}

	positions := make([]positionValue, 0, len(holdings))
	totalPortfolioValue := 0.0

	for _, h := range holdings {
		stock, err := mc.SingleStock(h.Ticker)
		if err != nil {
			log.Error("fetch_price", h.Ticker, err)
			return err
		}

		value := h.SharesHeld * stock.Price
		positions = append(positions, positionValue{holding: h, price: stock.Price, value: value})
		totalPortfolioValue += value

		log.Info("position_valued", h.Ticker, map[string]any{
			"shares": h.SharesHeld, "price": stock.Price, "market_value": value,
		})
	}

	log.Info("portfolio_valued", "", map[string]any{
		"total_kes": totalPortfolioValue, "positions": len(positions),
	})

	actions := make([]models.RebalanceAction, 0, len(positions))

	for _, pos := range positions {
		currentPct := 0.0
		if totalPortfolioValue > 0 {
			currentPct = (pos.value / totalPortfolioValue) * 100
		}
		targetPct := pos.holding.TargetPct
		drift := currentPct - targetPct
		targetValue := (targetPct / 100) * totalPortfolioValue
		kesDifference := math.Abs(pos.value - targetValue)
		approxShares := math.Floor(kesDifference / pos.price)

		action := "HOLD"
		reason := fmt.Sprintf("%.1f%% vs target %.1f%% — within ±%.1f%% threshold",
			currentPct, targetPct, DriftThreshold)

		if math.Abs(drift) >= DriftThreshold {
			if drift > 0 {
				action = "TRIM"
				reason = fmt.Sprintf(
					"%.1f%% overweight (target %.1f%%, current %.1f%%). Sell ~%d shares (KSh%.0f) to rebalance.",
					drift, targetPct, currentPct, int(approxShares), kesDifference,
				)
			} else {
				action = "BUY_MORE"
				reason = fmt.Sprintf(
					"%.1f%% underweight (target %.1f%%, current %.1f%%). Buy ~%d shares (KSh%.0f) to rebalance.",
					math.Abs(drift), targetPct, currentPct, int(approxShares), kesDifference,
				)
			}
		}

		a := models.RebalanceAction{
			Ticker:       pos.holding.Ticker,
			CurrentPct:   currentPct,
			TargetPct:    targetPct,
			DriftPct:     drift,
			CurrentValue: pos.value,
			TargetValue:  targetValue,
			Action:       action,
			KESAmount:    kesDifference,
			Shares:       approxShares,
			Reason:       reason,
		}
		actions = append(actions, a)

		log.Info("rebalance_action", pos.holding.Ticker, map[string]any{
			"current_pct": currentPct, "target_pct": targetPct,
			"drift": drift, "action": action,
		})
	}

	gc := groq.New()
	actionsJSON, _ := json.MarshalIndent(actions, "", "  ")
	var commentary string

	log.TimedStep("ai_rebalance_commentary", "ALL", func() (map[string]any, error) {
		var callErr error
		commentary, callErr = gc.Complete(
			groq.ModelQuality,
			`You are an NSE Kenya portfolio manager. Review this weekly rebalance analysis.
Be specific: (1) which TRIM actions are most urgent and why, (2) which BUY_MORE actions align with current NSE conditions, (3) total capital required if all BUY_MORE actions are executed.
Keep under 250 words. Use KES.`,
			fmt.Sprintf("Portfolio rebalance analysis (total KSh %.0f):\n%s",
				totalPortfolioValue, string(actionsJSON)),
			500,
		)
		return map[string]any{"length": len(commentary)}, callErr
	})

	if err := sh.WriteRebalancePlan(actions, commentary); err != nil {
		log.Error("write_rebalance_plan", "", err)
		return err
	}

	gc2 := gmail.New()
	actionsAsMap := make([]map[string]interface{}, 0, len(actions))
	for _, a := range actions {
		actionsAsMap = append(actionsAsMap, map[string]interface{}{
			"ticker": a.Ticker, "action": a.Action,
			"current_pct": fmt.Sprintf("%.1f%%", a.CurrentPct),
			"target_pct":  fmt.Sprintf("%.1f%%", a.TargetPct),
			"kes_amount":  fmt.Sprintf("%.0f", a.KESAmount),
		})
	}
	date := time.Now().Format("02 Jan 2006")
	if emailErr := gc2.RebalanceReport(date, actionsAsMap, commentary); emailErr != nil {
		log.Error("send_email", "", emailErr)
	} else {
		log.Info("email_sent", "", map[string]any{"report": "rebalance"})
	}

	output := map[string]any{
		"actions":         actions,
		"commentary":      commentary,
		"total_portfolio": totalPortfolioValue,
		"generated_at":    time.Now().Format(time.RFC3339),
	}
	outJSON, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(outJSON))

	log.Info("session_complete", "", map[string]any{"actions": len(actions)})
	return nil
}
```

---

## Part 16 — cmd/alerts/alerts.go (Complete)

**`cmd/alerts/alerts.go`**

```go
package alerts

import (
	"encoding/json"
	"fmt"
	"time"

	"nse-trader/internal/gmail"
	"nse-trader/internal/logger"
	"nse-trader/internal/mansa"
	"nse-trader/internal/sheets"
)

type AlertFired struct {
	Ticker    string  `json:"ticker"`
	Label     string  `json:"label"`
	Price     float64 `json:"price"`
	Threshold float64 `json:"threshold"`
	Direction string  `json:"direction"` // ABOVE | BELOW
	Time      string  `json:"time"`
}

func Run() error {
	log, err := logger.New("alerts")
	if err != nil {
		return err
	}
	defer log.Close()

	log.Info("session_start", "", map[string]any{"time": time.Now().Format(time.RFC3339)})

	sh, err := sheets.New()
	if err != nil {
		log.Error("sheets_init", "", err)
		return err
	}

	rules, err := sh.ReadAlertRules()
	if err != nil {
		log.Error("load_alert_rules", "", err)
		return err
	}
	log.Info("alert_rules_loaded", "", map[string]any{"count": len(rules)})

	if len(rules) == 0 {
		fmt.Println(`{"message": "No alert rules defined. Add rows to the Alerts tab in your Google Sheet."}`)
		return nil
	}

	mc := mansa.New()
	gc := gmail.New()
	fired := make([]AlertFired, 0)

	for _, rule := range rules {
		stock, err := mc.SingleStock(rule.Ticker)
		if err != nil {
			log.Error("fetch_price", rule.Ticker, err)
			continue // non-fatal: check the next ticker
		}

		log.Info("price_checked", rule.Ticker, map[string]any{
			"price": stock.Price, "alert_above": rule.AlertAbove, "alert_below": rule.AlertBelow,
		})

		if rule.AlertAbove > 0 && stock.Price >= rule.AlertAbove {
			fired = append(fired, AlertFired{
				Ticker: rule.Ticker, Label: rule.Label, Price: stock.Price,
				Threshold: rule.AlertAbove, Direction: "ABOVE",
				Time: time.Now().Format(time.RFC3339),
			})
			if emailErr := gc.PriceAlert(rule.Ticker, rule.Label, stock.Price, rule.AlertAbove, "ABOVE"); emailErr != nil {
				log.Error("send_alert_email", rule.Ticker, emailErr)
			} else {
				log.Info("alert_fired", rule.Ticker, map[string]any{
					"direction": "ABOVE", "price": stock.Price, "threshold": rule.AlertAbove,
				})
			}
		}

		if rule.AlertBelow > 0 && stock.Price <= rule.AlertBelow {
			fired = append(fired, AlertFired{
				Ticker: rule.Ticker, Label: rule.Label, Price: stock.Price,
				Threshold: rule.AlertBelow, Direction: "BELOW",
				Time: time.Now().Format(time.RFC3339),
			})
			if emailErr := gc.PriceAlert(rule.Ticker, rule.Label, stock.Price, rule.AlertBelow, "BELOW"); emailErr != nil {
				log.Error("send_alert_email", rule.Ticker, emailErr)
			} else {
				log.Info("alert_fired", rule.Ticker, map[string]any{
					"direction": "BELOW", "price": stock.Price, "threshold": rule.AlertBelow,
				})
			}
		}
	}

	log.Info("session_complete", "", map[string]any{
		"rules_checked": len(rules), "alerts_fired": len(fired),
	})

	output := map[string]any{
		"alerts_fired":  fired,
		"rules_checked": len(rules),
		"checked_at":    time.Now().Format(time.RFC3339),
	}
	outJSON, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(outJSON))

	return nil
}
```

---

## Part 17 — main.go

**`main.go`**

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"nse-trader/cmd/alerts"
	"nse-trader/cmd/dca"
	"nse-trader/cmd/rebalance"
	"nse-trader/cmd/scanner"
	"nse-trader/cmd/sentiment"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Note: .env file not found, using environment variables")
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	var err error
	switch command {
	case "scanner":
		fmt.Println("=== NSE Market Scanner ===")
		err = scanner.Run()
	case "sentiment":
		fmt.Println("=== News Sentiment Engine ===")
		err = sentiment.Run()
	case "dca":
		fmt.Println("=== DCA Buy List Generator ===")
		err = dca.Run()
	case "rebalance":
		fmt.Println("=== Portfolio Rebalancer ===")
		err = rebalance.Run()
	case "alerts":
		fmt.Println("=== Price Alert Monitor ===")
		err = alerts.Run()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		log.Fatalf("Command '%s' failed: %v\n", command, err)
	}

	fmt.Printf("\n✓ Command '%s' completed successfully\n", command)
}

func printUsage() {
	fmt.Println(`NSE Trading Intelligence System v3

Usage:
  go run main.go <command>

Commands:
  scanner    — Fetch all NSE prices, detect strong moves, email digest
  sentiment  — Score today's business news for NSE impact, write to Sheets
  dca        — Generate DCA buy recommendations, email Monday report
  rebalance  — Analyse portfolio drift, email Friday rebalance report
  alerts     — Check watchlist prices, fire Gmail alerts if threshold crossed

VSCode:
  Use Run panel (Ctrl+Shift+D) to launch any command with debug support.
  All commands log to ./logs/trading.jsonl for analysis.

Environment:
  Copy .env.example to .env and fill in your API keys.
  Place credentials.json (Google service account) in project root.`)
}
```

---

## Part 18 — .vscode/launch.json

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Run: Scanner",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/main.go",
      "args": ["scanner"],
      "envFile": "${workspaceFolder}/.env"
    },
    {
      "name": "Run: Sentiment",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/main.go",
      "args": ["sentiment"],
      "envFile": "${workspaceFolder}/.env"
    },
    {
      "name": "Run: DCA + Buy List",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/main.go",
      "args": ["dca"],
      "envFile": "${workspaceFolder}/.env"
    },
    {
      "name": "Run: Rebalancer",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/main.go",
      "args": ["rebalance"],
      "envFile": "${workspaceFolder}/.env"
    },
    {
      "name": "Run: Price Alerts",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/main.go",
      "args": ["alerts"],
      "envFile": "${workspaceFolder}/.env"
    },
    {
      "name": "Debug: Scanner (breakpoints)",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/main.go",
      "args": ["scanner"],
      "envFile": "${workspaceFolder}/.env"
    }
  ]
}
```

---

## Part 19 — n8n + Docker: The Binary Mount Solution

n8n runs inside Docker and cannot call `go run` on your host because Go is not installed inside the container. The fix: compile the Go code into a self-contained binary on your host, then mount it into the Docker container.

### Step 1: Compile the binary

```bash
cd ~/nse-trader

# Build a standalone Linux binary
go build -o bin/nse-trader main.go

# Test it directly
./bin/nse-trader scanner
```

### Step 2: Update docker-compose.yml

```yaml
version: "3.8"
services:
  n8n:
    image: n8nio/n8n
    ports:
      - "5678:5678"
    environment:
      - N8N_BASIC_AUTH_ACTIVE=true
      - N8N_BASIC_AUTH_USER=admin
      - N8N_BASIC_AUTH_PASSWORD=changeme
    volumes:
      - n8n_data:/home/node/.n8n
      # Mount the compiled binary into the container
      - /home/YOUR_USERNAME/nse-trader/bin:/opt/nse-trader/bin
      # Mount secrets and credentials
      - /home/YOUR_USERNAME/nse-trader/credentials.json:/opt/nse-trader/credentials.json
      - /home/YOUR_USERNAME/nse-trader/.env:/opt/nse-trader/.env
      # Mount logs so both host and container share the same log file
      - /home/YOUR_USERNAME/nse-trader/logs:/opt/nse-trader/logs

volumes:
  n8n_data:
```

Replace `YOUR_USERNAME` with your actual Linux username (run `echo $USER` to check).

```bash
# Restart n8n to pick up the new mounts
docker compose down && docker compose up -d
```

### Step 3: n8n Execute Command node

In each n8n workflow, the **Execute Command** node runs:

```bash
cd /opt/nse-trader && ./bin/nse-trader scanner
```

The `cd` sets the working directory so that `./credentials.json`, `.env`, and `./logs/` all resolve correctly.

Commands for each workflow:

| Workflow | Execute Command |
|----------|----------------|
| Scanner | `cd /opt/nse-trader && ./bin/nse-trader scanner` |
| Sentiment | `cd /opt/nse-trader && ./bin/nse-trader sentiment` |
| DCA | `cd /opt/nse-trader && ./bin/nse-trader dca` |
| Rebalancer | `cd /opt/nse-trader && ./bin/nse-trader rebalance` |
| Alerts | `cd /opt/nse-trader && ./bin/nse-trader alerts` |

After the Execute Command node, add a **Code** node to parse the JSON if you need it in n8n logic:

```javascript
const output = JSON.parse($json.stdout);
return [{ json: output }];
```

Since Gmail is handled inside the Go binary, you don't need any notification node in n8n. n8n's only job is to trigger the binary on schedule.

### Step 4: Rebuild after code changes

```bash
cd ~/nse-trader
go build -o bin/nse-trader main.go
# No Docker restart needed — the next n8n run picks up the new binary automatically
```

### n8n Schedule Reference

| Workflow | Cron | When |
|----------|------|------|
| Scanner | `5 9 * * 1-5` | 9:05 AM weekdays |
| Sentiment | `0 8 * * 1-5` | 8:00 AM weekdays |
| DCA | `30 7 * * 1` | Monday 7:30 AM |
| Rebalancer | `30 15 * * 5` | Friday 3:30 PM |
| Alerts | `*/30 9-15 * * 1-5` | Every 30 min, market hours |

---

## Part 20 — Quick Start Checklist

Work through this top to bottom. Each step depends on the ones before it.

**Environment:**
- [ ] Go 1.22+ installed (`go version`)
- [ ] VSCode with Go extension installed
- [ ] Project created at `~/nse-trader` (`go mod init nse-trader`)
- [ ] All directories created (`cmd/`, `internal/`, `logs/`, `bin/`)

**API keys (follow Part 2):**
- [ ] Mansa Markets key → `MANSA_API_KEY` in `.env`
- [ ] Groq key → `GROQ_API_KEY` in `.env`
- [ ] Gmail App Password → `GMAIL_FROM`, `GMAIL_APP_PASSWORD`, `GMAIL_TO` in `.env`
- [ ] Google Cloud project created, Sheets API + Drive API enabled
- [ ] Service account created, `credentials.json` in project root
- [ ] Google Sheet created with 6 tabs: Holdings, DailyPrices, Sentiment, DCA Plan, Alerts, Rebalance
- [ ] Sheet shared with service account email (Editor)
- [ ] Sheet ID → `GOOGLE_SHEETS_ID` in `.env`
- [ ] `GOOGLE_CREDENTIALS_FILE=./credentials.json` in `.env`

**Holdings data:**
- [ ] At least 2–3 rows in Holdings tab with real tickers, shares, avg buy price, target %, weekly budget
- [ ] At least 1–2 rows in Alerts tab with above/below thresholds

**Code:**
- [ ] All files written: models, sheets, gmail, logger, groq, mansa, scanner, sentiment, dca, rebalance, alerts, main
- [ ] `go mod tidy` — no errors
- [ ] `go build ./...` — no output means success

**Test from VSCode (Ctrl+Shift+D):**
- [ ] `scanner` — prints JSON, rows appear in DailyPrices tab, log grows
- [ ] `sentiment` — scores articles, rows appear in Sentiment tab
- [ ] `dca` — reads Holdings, prints recommendations, writes DCA Plan tab, email arrives in inbox
- [ ] `rebalance` — reads Holdings, prints drift analysis, writes Rebalance tab, email arrives in inbox
- [ ] `alerts` — reads Alerts tab, checks prices, fires email if threshold is crossed

**n8n:**
- [ ] `go build -o bin/nse-trader main.go` succeeds
- [ ] `docker-compose.yml` updated with volume mounts
- [ ] n8n restarted: `docker compose down && docker compose up -d`
- [ ] One workflow created and tested manually: runs the binary, no errors in n8n execution log

**First full week:**
- [ ] Monday 7:30 AM: DCA plan email arrives
- [ ] Daily 9:05 AM: scanner runs, email arrives if strong moves detected
- [ ] Friday 3:30 PM: rebalance report email arrives
- [ ] Log file growing: `~/nse-trader/logs/trading.jsonl`

---

## Appendix — Analysing Your Logs

```bash
# All STRONG_BUY recommendations
cat logs/trading.jsonl | grep '"action":"STRONG_BUY"' | jq .

# Tickers that appeared most as strong movers
cat logs/trading.jsonl | grep '"signal":"STRONG_UP"' | jq -r '.ticker' | sort | uniq -c | sort -rn

# Average Groq response time for sentiment
cat logs/trading.jsonl | grep '"step":"score_article"' | jq '.data.duration_ms' | sort

# All errors
cat logs/trading.jsonl | grep '"level":"ERROR"' | jq '{timestamp,command,step,error}'

# All alerts that fired
cat logs/trading.jsonl | grep '"step":"alert_fired"' | jq '{timestamp, ticker: .ticker, data}'
```

---

*NSE Intelligence System v3 · Go + n8n + Groq (free) + Gmail · All gaps closed · June 2026*
