package thresholds

// This command recalculates alert thresholds for all tickers in the Alerts sheet.
import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"nseTradr/internal/gmail"
	"nseTradr/internal/logger"
	"nseTradr/internal/models"
	"nseTradr/internal/sheets"
)

func Run() error {
	log, err := logger.New("thresholds")
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

	// Read existing alert rules to get tickers and labels
	rules, err := sh.ReadAlertRules()
	if err != nil {
		log.Error("read_alert_rules", "", err)
		return err
	}

	// Read daily prices — all rows
	prices, err := sh.ReadDailyPrices()
	if err != nil {
		log.Error("read_daily_prices", "", err)
		return err
	}

	// Calculate weekly average per ticker (last 5 trading days)
	// Group prices by ticker
	pricesByTicker := make(map[string][]float64)
	datesByTicker := make(map[string][]string)
	for _, p := range prices {
		pricesByTicker[p.Ticker] = append(pricesByTicker[p.Ticker], p.Price)
		datesByTicker[p.Ticker] = append(datesByTicker[p.Ticker], p.Date)
	}

	updates := make([]models.ThresholdUpdate, 0, len(rules))

	for _, rule := range rules {
		tickerPrices := pricesByTicker[rule.Ticker]
		if len(tickerPrices) == 0 {
			log.Warn("no_price_data", rule.Ticker, map[string]any{"skipping": true})
			continue
		}

		// Use last 5 entries (last 5 trading days)
		last5 := tickerPrices
		if len(last5) > 5 {
			last5 = tickerPrices[len(tickerPrices)-5:]
		}

		// Calculate average
		sum := 0.0
		for _, p := range last5 {
			sum += p
		}
		avg := sum / float64(len(last5))

		// Set thresholds at ±5% of weekly average
		newAbove := math.Round(avg*1.05*100) / 100
		newBelow := math.Round(avg*0.95*100) / 100

		updates = append(updates, models.ThresholdUpdate{
			Ticker:    rule.Ticker,
			Label:     rule.Label,
			WeeklyAvg: math.Round(avg*100) / 100,
			NewAbove:  newAbove,
			NewBelow:  newBelow,
			PrevAbove: rule.AlertAbove,
			PrevBelow: rule.AlertBelow,
		})

		log.Info("threshold_calculated", rule.Ticker, map[string]any{
			"weekly_avg": avg, "new_above": newAbove, "new_below": newBelow,
		})
	}

	// Write updated thresholds back to Alerts sheet
	if err := sh.WriteAlertThresholds(updates); err != nil {
		log.Error("write_thresholds", "", err)
		return err
	}

	// Send summary email
	gc := gmail.New()
	date := time.Now().Format("02 Jan 2006")
	subject := fmt.Sprintf("📊 Alert Thresholds Updated — %s", date)

	var body string
	body += fmt.Sprintf("<h2>Alert Thresholds Updated — %s</h2>\n", date)
	body += "<p>Weekly averages calculated from last 5 trading days. Thresholds set at ±5%.</p>\n"
	body += "<table style='border-collapse:collapse;width:100%'>\n"
	body += "<tr><th>Ticker</th><th>Weekly Avg</th><th>Alert Above</th><th>Alert Below</th></tr>\n"
	for _, u := range updates {
		body += fmt.Sprintf(
			"<tr><td><b>%s</b></td><td>KSh%.2f</td><td>KSh%.2f</td><td>KSh%.2f</td></tr>\n",
			u.Ticker, u.WeeklyAvg, u.NewAbove, u.NewBelow,
		)
	}
	body += "</table>"

	if emailErr := gc.Send(subject, body); emailErr != nil {
		log.Error("send_email", "", emailErr)
	} else {
		log.Info("email_sent", "", map[string]any{"updates": len(updates)})
	}

	log.Info("session_complete", "", map[string]any{"updated": len(updates)})

	outJSON, _ := json.MarshalIndent(updates, "", "  ")
	fmt.Println(string(outJSON))
	return nil
}
