package app

import "testing"

func TestParseLegacyAskArgs(t *testing.T) {
	opts, confirm, prompt, err := parseLegacyAskArgs([]string{
		"--provider", "ollama",
		"--model", "deepseek-coder-v2:latest",
		"--base-url", "http://127.0.0.1:11434",
		"--no-confirm-tools",
		"spiegami", "questo", "errore",
	})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Provider != "ollama" {
		t.Fatalf("expected provider ollama, got %q", opts.Provider)
	}
	if opts.Model != "deepseek-coder-v2:latest" {
		t.Fatalf("unexpected model: %q", opts.Model)
	}
	if opts.BaseURL != "http://127.0.0.1:11434" {
		t.Fatalf("unexpected base-url: %q", opts.BaseURL)
	}
	if confirm {
		t.Fatalf("expected confirmTools=false")
	}
	if prompt != "spiegami questo errore" {
		t.Fatalf("unexpected prompt: %q", prompt)
	}
}

func TestParseLegacyAskArgsMissingProviderValue(t *testing.T) {
	_, _, _, err := parseLegacyAskArgs([]string{"--provider"})
	if err == nil {
		t.Fatal("expected error for missing --provider value")
	}
}
