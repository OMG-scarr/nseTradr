package dca

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
