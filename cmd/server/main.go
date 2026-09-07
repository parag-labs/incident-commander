// Command server runs the AI Incident Commander HTTP service. By default it uses the
// deterministic mock model so it needs no API key; point it at a real OpenAI-compatible
// endpoint with the LLM_* environment variables to use a live model.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/parag-labs/incident-commander/internal/agent/llm"
	"github.com/parag-labs/incident-commander/internal/api"
)

func main() {
	addr := flag.String("addr", envOr("ADDR", ":8080"), "listen address")
	autoMedium := flag.Bool("auto-approve-medium", envBool("AUTO_APPROVE_MEDIUM", false), "run MEDIUM remediations without a human")
	workers := flag.Int("workers", 4, "evidence-collection worker pool size")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	model := selectModel(log)
	srv := api.New(api.Config{Model: model, Workers: *workers, AutoApproveMedium: *autoMedium, Logger: log})

	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           srv.Handler(*autoMedium, *workers),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("incident-commander listening", "addr", *addr, "model", model.Name(), "auto_approve_medium", *autoMedium)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "err", err)
	}
}

// selectModel wires a real OpenAI-compatible client when LLM_BASE_URL is set, otherwise
// the deterministic mock. CI and the default demo never need an API key.
func selectModel(log *slog.Logger) llm.LLMClient {
	base := os.Getenv("LLM_BASE_URL")
	if base == "" {
		return llm.MockLLM{}
	}
	log.Info("using OpenAI-compatible model", "base_url", base)
	return llm.NewOpenAI(llm.OpenAIConfig{
		BaseURL: base,
		APIKey:  os.Getenv("LLM_API_KEY"),
		Model:   envOr("LLM_MODEL", "gpt-4o-mini"),
	})
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	switch os.Getenv(key) {
	case "1", "true", "TRUE", "yes":
		return true
	case "0", "false", "FALSE", "no":
		return false
	default:
		return def
	}
}
