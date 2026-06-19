package alerts

import (
	"encoding/json"
	"fmt"
	"time"

	"nseTradr/internal/gmail"
	"nseTradr/internal/logger"
	"nseTradr/internal/mansa"
	"nseTradr/internal/sheets"
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
