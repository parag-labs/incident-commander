package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/parag-labs/incident-commander/pkg/models"
)

// OpenAIConfig configures the OpenAI-compatible client. It works against OpenAI, Azure
// OpenAI, or any server that speaks the /chat/completions API (Ollama, vLLM, LM Studio),
// so "local model" support is just a different BaseURL.
type OpenAIConfig struct {
	BaseURL string        // e.g. https://api.openai.com/v1 or http://localhost:11434/v1
	APIKey  string        // may be empty for local servers
	Model   string        // e.g. gpt-4o-mini or a local model name
	Timeout time.Duration // per-request timeout
}

// OpenAILLM is the production client. It is compiled and usable but is never exercised
// in CI - tests use MockLLM/ScriptedLLM so no API key is ever required.
type OpenAILLM struct {
	cfg  OpenAIConfig
	http *http.Client
}

// NewOpenAI builds a client from config.
func NewOpenAI(cfg OpenAIConfig) *OpenAILLM {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &OpenAILLM{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

// Name identifies the provider.
func (c *OpenAILLM) Name() string { return "openai:" + c.cfg.Model }

const diagnoseSystem = `You are an incident investigator. Reason ONLY over the supplied evidence.
Never invent tool results, metrics, or evidence IDs. Return a single JSON object matching:
{"summary":string,"hypotheses":[{"cause":string,"confidence":number,"evidence_ids":[string]}],
"recommended_action":string,"recommended_service":string,"risk":"low|medium|high","needs_human_approval":boolean}
recommended_action must be one of the allowed tool names. Cite evidence_ids that appear in the input.`

// Diagnose calls the chat completions API and returns the model's raw JSON text.
func (c *OpenAILLM) Diagnose(ctx context.Context, bundle EvidenceBundle) (string, error) {
	payload, err := json.Marshal(bundle)
	if err != nil {
		return "", err
	}
	return c.chat(ctx, diagnoseSystem, string(payload))
}

// Explain asks the model for a short human narrative of a finished incident.
func (c *OpenAILLM) Explain(ctx context.Context, report models.IncidentReport) (string, error) {
	payload, err := json.Marshal(report)
	if err != nil {
		return "", err
	}
	return c.chat(ctx, "Summarize this resolved incident in 3-4 sentences for an on-call engineer. Be specific and cite the root cause.", string(payload))
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func (c *OpenAILLM) chat(ctx context.Context, system, user string) (string, error) {
	reqBody, err := json.Marshal(chatRequest{
		Model: c.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	})
	if err != nil {
		return "", err
	}
	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("llm returned no choices")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}
