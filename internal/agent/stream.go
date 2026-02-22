package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

type TokenCallback func(token string)

func askOpenAIStream(prompt string, cfg openAIConfig, opts AskOptions, onToken TokenCallback) (string, string, error) {
	baseURL, model, apiKey := normalizedOpenAIValues(cfg)
	if apiKey == "" {
		return "", "", fmt.Errorf("missing OpenAI API key (set in %s or OPENAI_API_KEY)", configPath())
	}
	slog.Debug("LLM stream request", "provider", "openai", "model", model, "prompt_chars", len(prompt))

	systemMsg := "You are a pragmatic coding assistant."
	if strings.TrimSpace(opts.SystemPrompt) != "" {
		systemMsg = opts.SystemPrompt
	}

	reqBody := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemMsg},
			{"role": "user", "content": prompt},
		},
		"stream": true,
	}
	if opts.Temperature != nil {
		reqBody["temperature"] = *opts.Temperature
	}
	if opts.MaxTokens > 0 {
		reqBody["max_tokens"] = opts.MaxTokens
	}
	if opts.JSONMode {
		reqBody["response_format"] = map[string]string{"type": "json_object"}
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return "", model, err
	}
	res, err := doWithRetry(func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)
		return req, nil
	})
	if err != nil {
		return "", model, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", model, fmt.Errorf("openai status: %s", res.Status)
	}

	var buf strings.Builder
	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 {
			token := chunk.Choices[0].Delta.Content
			if token != "" {
				buf.WriteString(token)
				if onToken != nil {
					onToken(token)
				}
			}
		}
	}

	answer := strings.TrimSpace(buf.String())
	if answer == "" {
		return "", model, fmt.Errorf("empty openai stream response")
	}
	return answer, model, nil
}

func askOllamaStream(prompt string, cfg ollamaConfig, opts AskOptions, onToken TokenCallback) (string, string, error) {
	baseURL, model := normalizedOllamaValues(cfg)
	slog.Debug("LLM stream request", "provider", "ollama", "model", model, "prompt_chars", len(prompt))

	reqBody := map[string]any{
		"model":  model,
		"prompt": prompt,
		"stream": true,
	}
	if strings.TrimSpace(opts.SystemPrompt) != "" {
		reqBody["system"] = opts.SystemPrompt
	}
	if opts.JSONMode {
		reqBody["format"] = "json"
	}
	ollamaOpts := map[string]any{}
	if opts.Temperature != nil {
		ollamaOpts["temperature"] = *opts.Temperature
	}
	if opts.MaxTokens > 0 {
		ollamaOpts["num_predict"] = opts.MaxTokens
	}
	if len(ollamaOpts) > 0 {
		reqBody["options"] = ollamaOpts
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return "", model, err
	}
	res, err := doWithRetry(func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, baseURL+"/api/generate", bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	})
	if err != nil {
		return "", model, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", model, fmt.Errorf("ollama status: %s", res.Status)
	}

	var buf strings.Builder
	decoder := json.NewDecoder(res.Body)
	for decoder.More() {
		var chunk struct {
			Response string `json:"response"`
			Done     bool   `json:"done"`
		}
		if err := decoder.Decode(&chunk); err != nil {
			break
		}
		if chunk.Response != "" {
			buf.WriteString(chunk.Response)
			if onToken != nil {
				onToken(chunk.Response)
			}
		}
		if chunk.Done {
			break
		}
	}

	answer := strings.TrimSpace(buf.String())
	if answer == "" {
		return "", model, fmt.Errorf("empty ollama stream response")
	}
	return answer, model, nil
}

func AskStream(prompt string, opts AskOptions, onToken TokenCallback) (AskResult, error) {
	text := strings.TrimSpace(prompt)
	if text == "" {
		return AskResult{}, fmt.Errorf("prompt is required")
	}

	cfg, cfgErr := loadUserConfig()
	if cfgErr != nil {
		return AskResult{}, cfgErr
	}

	provider := strings.ToLower(strings.TrimSpace(opts.Provider))
	if provider == "" {
		provider = "openai"
	}

	switch provider {
	case "ollama":
		applyOllamaOverrides(&cfg, opts)
		answer, model, err := askOllamaStream(text, cfg.Ollama, opts, onToken)
		if err != nil {
			return AskResult{}, err
		}
		return AskResult{Text: answer, Provider: "ollama", Model: model}, nil
	case "openai":
		applyOpenAIOverrides(&cfg, opts)
		answer, model, err := askOpenAIStream(text, cfg.OpenAI, opts, onToken)
		if err != nil {
			return AskResult{}, err
		}
		return AskResult{Text: answer, Provider: "openai", Model: model}, nil
	case "auto":
		applyOllamaOverrides(&cfg, opts)
		if answer, model, err := askOllamaStream(text, cfg.Ollama, opts, onToken); err == nil {
			return AskResult{Text: answer, Provider: "ollama", Model: model}, nil
		}
		applyOpenAIOverrides(&cfg, opts)
		answer, model, err := askOpenAIStream(text, cfg.OpenAI, opts, onToken)
		if err != nil {
			return AskResult{}, fmt.Errorf("ollama unavailable and openai fallback failed: %w", err)
		}
		return AskResult{Text: answer, Provider: "openai", Model: model}, nil
	default:
		return AskResult{}, fmt.Errorf("invalid provider %q (use auto|ollama|openai)", opts.Provider)
	}
}

func DecideWithPluginsStream(userPrompt, pluginCatalog, toolCatalog string, opts AskOptions, envContext string, onToken TokenCallback) (DecisionResult, error) {
	p := strings.TrimSpace(userPrompt)
	if p == "" {
		return DecisionResult{}, fmt.Errorf("prompt is required")
	}

	systemPrompt := buildDecisionSystemPrompt(pluginCatalog, toolCatalog)
	userMsg := buildDecisionUserPrompt(p, envContext)
	dOpts := decisionOpts(opts, systemPrompt)

	raw, err := AskStream(userMsg, dOpts, onToken)
	if err != nil {
		return DecisionResult{}, err
	}
	parsed, err := parseDecisionJSON(raw.Text)
	if err != nil {
		repaired, repErr := askDecisionJSONRepair(raw.Text, dOpts)
		if repErr == nil {
			if parsed2, p2Err := parseDecisionJSON(repaired.Text); p2Err == nil {
				parsed2.Provider = repaired.Provider
				parsed2.Model = repaired.Model
				if parsed2.Action != "run_plugin" && parsed2.Action != "run_tool" && parsed2.Action != "create_function" {
					parsed2.Action = "answer"
				}
				return parsed2, nil
			}
		}
		return DecisionResult{
			Action:   "answer",
			Answer:   raw.Text,
			Provider: raw.Provider,
			Model:    raw.Model,
		}, nil
	}
	if parsed.Provider == "" {
		parsed.Provider = raw.Provider
	}
	if parsed.Model == "" {
		parsed.Model = raw.Model
	}
	return parsed, nil
}
