# NSE Trading Intelligence System (nseTradr)

A compact, self-contained Go toolkit for daily NSE price scanning, AI-driven sentiment, DCA recommendations, portfolio rebalancing, and Gmail notifications. Designed to run as a compiled binary (for n8n scheduling) or from VS Code during development.

**Quick Summary**
- **Language:** Go (1.22+)
- **Features:** price scanner, sentiment scoring (Groq), DCA engine, rebalance engine, price alerts via Gmail, Google Sheets integration
- **Run modes:** CLI commands (scanner, sentiment, dca, rebalance, alerts) invoked via `main.go` or the compiled binary in `bin/`

**Get Started (quick)**
1. Copy `.env.example` to `.env` and fill keys described in Guide.md.
2. Place your Google service account JSON as `credentials.json` in the project root.
3. Install dependencies and build:

```bash
go mod tidy
go build -o bin/nseTradr main.go
```

4. Run a command locally for a quick test:

```bash
./bin/nseTradr scanner
```

Project layout highlights
- `main.go` — CLI entrypoint and command dispatch
- `cmd/` — command implementations: `scanner`, `sentiment`, `dca`, `rebalance`, `alerts`
- `internal/` — helpers: `sheets`, `mansa`, `groq`, `gmail`, `logger`, `models`
- `bin/` — compiled binary for n8n
- `logs/trading.jsonl` — newline JSON log stream

Environment & secrets
- Copy `.env.example` → `.env` and populate: `MANSA_API_KEY`, `GROQ_API_KEY`, `GOOGLE_SHEETS_ID`, `GMAIL_*`, `GOOGLE_CREDENTIALS_FILE`, `LOG_FILE`
- Never commit `credentials.json` or `.env`.

n8n + Docker usage
- Build the binary on the host and mount `./bin` into the n8n container. n8n only schedules the binary, it does not run `go run` inside the container.
- Example execute command for a workflow (set working dir to the mounted project):

```bash
cd /opt/nse-tradr && ./bin/nseTradr scanner
```

Operational notes
- Use Gmail App Password for SMTP in `.env`.
- Share the target Google Sheet with the service account `client_email` from `credentials.json`.
- `logs/trading.jsonl` captures structured JSON logs for audits and debugging.

Troubleshooting
- If builds fail: run `go mod tidy` and inspect `go build` errors.
- Sheets API permission errors: confirm Drive & Sheets APIs enabled and the sheet shared with the service account.
- SMTP errors: verify `GMAIL_FROM`, `GMAIL_APP_PASSWORD` and that 2-step verification and app password are enabled.


Contributing
- File issues and PRs on this repository. Keep secrets out of commits.


