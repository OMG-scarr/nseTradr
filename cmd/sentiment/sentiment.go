package sentiment

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"nseTradr/internal/groq"
	"nseTradr/internal/logger"
	"nseTradr/internal/models"
	"nseTradr/internal/sheets"
)

var NewsSources = []string{
	"https://www.standardmedia.co.ke/rss/business.php",
	"https://www.capitalfm.co.ke/business/feed/",
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
