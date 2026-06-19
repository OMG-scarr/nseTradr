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
