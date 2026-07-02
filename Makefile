.PHONY: build run-scanner run-sentiment run-dca run-rebalance run-alerts run-digest run-performance run-thresholds morning friday errors logs

build:
	go build -o bin/nseTradr main.go
	@echo "✓ Built bin/nseTradr"

run-scanner:
	./bin/nseTradr scanner

run-sentiment:
	./bin/nseTradr sentiment

run-dca:
	./bin/nseTradr dca

run-rebalance:
	./bin/nseTradr rebalance

run-alerts:
	./bin/nseTradr alerts

run-digest:
	./bin/nseTradr digest

run-performance:
	./bin/nseTradr performance

run-thresholds:
	./bin/nseTradr thresholds

trade:
	./bin/nseTradr trade --ticker $(TICKER) --shares $(SHARES) --price $(PRICE)

morning:
	./bin/nseTradr sentiment
	./bin/nseTradr dca
	./bin/nseTradr scanner
	./bin/nseTradr digest

friday:
	./bin/nseTradr rebalance
	./bin/nseTradr performance

errors:
	cat logs/trading.jsonl | grep '"level":"ERROR"' | tail -20 | jq .

logs:
	tail -50 logs/trading.jsonl | jq .

clean:
	rm -f bin/nseTradr
	@echo "✓ Cleaned"
