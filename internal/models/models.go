package models

// Holding represents one stock position in your portfolio.
// This is what the "Holdings" sheet contains, one row per stock.
type Holding struct {
	Ticker       string  `json:"ticker"`        // e.g. "SCOM"
	Name         string  `json:"name"`          // e.g. "Safaricom PLC"
	SharesHeld   float64 `json:"shares_held"`   // how many shares you own
	AvgBuyPrice  float64 `json:"avg_buy_price"` // your weighted average cost
	TargetPct    float64 `json:"target_pct"`    // desired % of total portfolio
	WeeklyBudget float64 `json:"weekly_budget"` // KES to invest per week in this stock
}

// AlertRule is one row in the "Watchlist" sheet.
// Used by the alerts command to monitor stocks you don't yet own.
type AlertRule struct {
	Ticker     string  `json:"ticker"`
	AlertAbove float64 `json:"alert_above"` // fire alert when price crosses above this
	AlertBelow float64 `json:"alert_below"` // fire alert when price drops below this
	Label      string  `json:"label"`       // optional label ("earnings play", etc.)
}

// ScanResult is one NSE stock's daily summary produced by the scanner.
type ScanResult struct {
	Ticker    string  `json:"ticker"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	ChangePct float64 `json:"change_pct"`
	Volume    int64   `json:"volume"`
	Signal    string  `json:"signal"` // STRONG_UP, UP, NORMAL, DOWN, STRONG_DOWN
	Date      string  `json:"date"`
}

// SentimentResult is one scored news article from the sentiment engine.
type SentimentResult struct {
	Ticker     string  `json:"ticker"`
	Company    string  `json:"company"`
	Sentiment  string  `json:"sentiment"`  // Bullish, Bearish, Neutral
	Confidence float64 `json:"confidence"` // 1–10
	Reason     string  `json:"reason"`
	Headline   string  `json:"headline"`
	Link       string  `json:"link"`
	Date       string  `json:"date"`
}

// DCARecommendation is the DCA engine's output for one holding.
type DCARecommendation struct {
	Ticker         string  `json:"ticker"`
	CurrentPrice   float64 `json:"current_price"`
	AvgBuyPrice    float64 `json:"avg_buy_price"`
	CurrentPnLPct  float64 `json:"current_pnl_pct"`
	DipPct         float64 `json:"dip_from_avg_pct"`
	BaseWeeklyKES  float64 `json:"base_weekly_kes"`
	AdjustedKES    float64 `json:"adjusted_weekly_kes"`
	SharesToBuy    float64 `json:"shares_to_buy"`
	NewAvgAfterBuy float64 `json:"new_avg_after_buy"`
	Reason         string  `json:"reason"`
	Action         string  `json:"action"` // STRONG_BUY, BUY, SKIP
}

// RebalanceAction is what the rebalancer recommends for one holding.
type RebalanceAction struct {
	Ticker       string  `json:"ticker"`
	CurrentPct   float64 `json:"current_pct"`       // actual % of portfolio today
	TargetPct    float64 `json:"target_pct"`        // what you want it to be
	DriftPct     float64 `json:"drift_pct"`         // current - target
	CurrentValue float64 `json:"current_value_kes"` // market value in KES
	TargetValue  float64 `json:"target_value_kes"`  // what it should be in KES
	Action       string  `json:"action"`            // BUY_MORE, TRIM, HOLD
	KESAmount    float64 `json:"kes_amount"`        //
	ValueAdjust  float64 `json:"value_adjust_kes"`
	Shares       float64 `json:"shares"`
	Reason       string  `json:"reason"`
}
