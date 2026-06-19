package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"nseTradr/cmd/alerts"
	"nseTradr/cmd/dca"
	"nseTradr/cmd/rebalance"
	"nseTradr/cmd/scanner"
	"nseTradr/cmd/sentiment"
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
