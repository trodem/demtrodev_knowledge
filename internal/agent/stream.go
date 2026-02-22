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

func askOpenAIStream(prompt string, cfg openAIConfig, onToken TokenCallback) (string, string, error) {
	baseURL, model, apiKey := normalizedOpenAIValues(cfg)
	if apiKey == "" {
		return "", "", fmt.Errorf("missing OpenAI API key (set in %s or OPENAI_API_KEY)", configPath())
	}
	slog.Debug("LLM stream request", "provider", "openai", "model", model, "prompt_chars", len(prompt))

	reqBody := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a pragmatic coding assistant."},
			{"role": "user", "content": prompt},
		},
		"stream": true,
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

func askOllamaStream(prompt string, cfg ollamaConfig, onToken TokenCallback) (string, string, error) {
	baseURL, model := normalizedOllamaValues(cfg)
	slog.Debug("LLM stream request", "provider", "ollama", "model", model, "prompt_chars", len(prompt))

	reqBody := map[string]any{
		"model":  model,
		"prompt": prompt,
		"stream": true,
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
		answer, model, err := askOllamaStream(text, cfg.Ollama, onToken)
		if err != nil {
			return AskResult{}, err
		}
		return AskResult{Text: answer, Provider: "ollama", Model: model}, nil
	case "openai":
		applyOpenAIOverrides(&cfg, opts)
		answer, model, err := askOpenAIStream(text, cfg.OpenAI, onToken)
		if err != nil {
			return AskResult{}, err
		}
		return AskResult{Text: answer, Provider: "openai", Model: model}, nil
	case "auto":
		applyOllamaOverrides(&cfg, opts)
		if answer, model, err := askOllamaStream(text, cfg.Ollama, onToken); err == nil {
			return AskResult{Text: answer, Provider: "ollama", Model: model}, nil
		}
		applyOpenAIOverrides(&cfg, opts)
		answer, model, err := askOpenAIStream(text, cfg.OpenAI, onToken)
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
	if strings.TrimSpace(pluginCatalog) == "" {
		pluginCatalog = "(none)"
	}
	if strings.TrimSpace(toolCatalog) == "" {
		toolCatalog = "(none)"
	}
	parts := []string{
		"You are an execution planner for a CLI assistant.",
		"You can either answer directly, run a plugin (PowerShell function), run a built-in tool, or propose creating a new function.",
		"",
		"Available plugins (PowerShell functions):",
		pluginCatalog,
		"",
		"Available tools:",
		toolCatalog,
		"",
		"Return ONLY valid JSON. Use one of these schemas:",
		`{"action":"answer","answer":"text"}`,
		`{"action":"run_plugin","plugin":"name","plugin_args":{"ParamName":"value","SwitchParam":"true"},"reason":"why","answer":"optional text"}`,
		`{"action":"run_tool","tool":"name","tool_args":{"key":"value"},"reason":"why","answer":"optional text"}`,
		`{"action":"create_function","function_description":"detailed description of what the function should do, its inputs and outputs","reason":"why no existing plugin fits"}`,
		"",
		"Plugin argument rules:",
		"- Use plugin_args (object) for named PowerShell parameters, NOT the args array.",
		"- Keys are parameter names WITHOUT the leading dash (e.g. \"Host\" not \"-Host\").",
		"- For switch parameters (flags like -Force, -Confirm), set the value to \"true\".",
		"- ALWAYS include ALL required parameters listed in the catalog for the chosen plugin.",
		"- Map values from the user request to the correct parameter names in the catalog.",
		"  Example: user says 'search for mario in user table' with a plugin having params Table (required), Value (required), Limit:",
		`  => plugin_args: {"Table":"user","Value":"mario"}`,
		"- If a required parameter cannot be inferred from the user request at all, return action=answer and ask the user.",
		"- If a previous step failed with 'missing mandatory parameters', the NEXT attempt MUST include those parameters.",
		"",
		"General rules:",
		"- action must be answer, run_plugin, run_tool, or create_function.",
		"- Do not invent plugin or tool names; use only the catalog above.",
		"- If the user request requires an operation that no existing plugin or tool can handle, return action=create_function.",
		"- Only use create_function for tasks that genuinely need a new automation capability, not for general knowledge questions.",
		"- If a plugin requires confirmation or is destructive, mention it in the answer.",
		"- For search tool use tool_args keys: base, ext, name, sort, limit, offset.",
		"- For rename tool use tool_args keys: base, from, to, name, case_sensitive.",
		"- For recent tool use tool_args keys: base, limit, offset.",
		"- For clean tool use tool_args keys: base, apply (true for delete, otherwise preview).",
	}
	if strings.TrimSpace(envContext) != "" {
		parts = append(parts, "", "Environment context:", envContext)
	}
	parts = append(parts, "", "User request:", p)
	decisionPrompt := strings.Join(parts, "\n")

	raw, err := AskStream(decisionPrompt, opts, onToken)
	if err != nil {
		return DecisionResult{}, err
	}
	parsed, err := parseDecisionJSON(raw.Text)
	if err != nil {
		repaired, repErr := askDecisionJSONRepair(raw.Text, opts)
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
