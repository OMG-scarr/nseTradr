package scanner

import (
	"encoding/json"
	"fmt"
	"time"

	"nseTradr/internal/gmail"
	"nseTradr/internal/logger"
	"nseTradr/internal/mansa"
	"nseTradr/internal/models"
	"nseTradr/internal/sheets"
)

const MinVolume = 1_000

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
