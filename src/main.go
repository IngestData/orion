// Orion - hunter of the hidden.
//
// SCA tool for the transformer code IngestData Pipe customers deploy into their Workspace.
// A single HTTP endpoint (/scan) that ingestdataproto calls synchronously whenever a project's
// transformer step is created or updated: Orion analyzes the source and reports whether it's
// safe to run, blocking deployment on any Critical finding. Language-specific scanning lives
// under src/<language>/ (see src/javascript for the reference implementation) behind the
// shared scan.Language interface - add a package and a registry entry here to support another
// transformer language.
package main

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Ingestdata/orion/src/javascript"
	"github.com/Ingestdata/orion/src/scan"
)

const maxRequestBody = 256 * 1024 // comfortably above a worst-case 20,000-char UTF-8 transformer plus JSON envelope overhead

var orionSecret string

type scanRequest struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

type scanResponse struct {
	Passed   bool           `json:"passed"`
	Findings []scan.Finding `json:"findings"`
}

func main() {
	port := os.Getenv("ORION_PORT")
	if port == "" {
		port = "8090"
	}

	// Fail closed at boot, matching proto's own fail-closed-on-scan-error behavior - an
	// unauthenticated Orion is worse than a down one, since proto would otherwise happily let
	// every transformer through against an Orion that silently accepts any caller.
	orionSecret = os.Getenv("ORION_SECRET")
	if orionSecret == "" {
		log.Fatal("ORION_SECRET is not set - refusing to start unauthenticated")
	}

	registry := map[string]scan.Language{
		"javascript": javascript.New(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/scan", handleScan(registry))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	log.Printf("Orion listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func handleScan(registry map[string]scan.Language) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Checked before the body is ever touched - an unauthenticated caller shouldn't be
		// able to spend any body-parsing cost trying to probe or DoS this endpoint.
		provided := r.Header.Get("X-Orion-Secret")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(orionSecret)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

		var req scanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Default to javascript when omitted: proto and orion deploy independently with no
		// atomic joint rollout, so an orion redeploy ahead of proto's own update must not
		// start hard-rejecting proto's current (language-less) request shape. Drop this
		// default once a second language ships and proto always sends one explicitly.
		language := req.Language
		if language == "" {
			language = "javascript"
		}

		scanner, ok := registry[language]
		if !ok {
			http.Error(w, "unsupported language: "+language, http.StatusBadRequest)
			return
		}

		findings := scanner.Scan(req.Code)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(scanResponse{
			Passed:   scan.Passed(findings),
			Findings: findings,
		})
	}
}
