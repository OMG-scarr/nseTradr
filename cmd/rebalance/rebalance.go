package rebalance

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"nseTradr/internal/gmail"
	"nseTradr/internal/groq"
	"nseTradr/internal/logger"
	"nseTradr/internal/mansa"
	"nseTradr/internal/models"
	"nseTradr/internal/sheets"
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
