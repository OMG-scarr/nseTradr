package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
)

func Run() error {
	http.HandleFunc("/scanner", handler("scanner"))
	http.HandleFunc("/sentiment", handler("sentiment"))
	http.HandleFunc("/dca", handler("dca"))
	http.HandleFunc("/rebalance", handler("rebalance"))
	http.HandleFunc("/alerts", handler("alerts"))
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	fmt.Println("NSE Trader API listening on :8080")
	return http.ListenAndServe(":8080", nil)
}

func handler(command string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		out, err := exec.Command("/opt/nseTradr/bin/nseTradr", command).Output()
		if err != nil {
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		// Find the JSON output (last JSON block in stdout)
		w.WriteHeader(200)
		w.Write(out)
	}
}
