package ui

import (
	"strings"
	"testing"
)

func TestRenderMarkdown_NoColor_StripsBold(t *testing.T) {
	withEnv("NO_COLOR", "1", func() {
		got := RenderMarkdown("This is **important** text")
		if strings.Contains(got, "**") {
			t.Fatalf("expected ** markers stripped with NO_COLOR, got %q", got)
		}
		if !strings.Contains(got, "important") {
			t.Fatalf("expected 'important' in output, got %q", got)
		}
	})
}

func TestRenderMarkdown_NoColor_StripsHeader(t *testing.T) {
	withEnv("NO_COLOR", "1", func() {
		got := RenderMarkdown("## Results")
		if strings.Contains(got, "##") {
			t.Fatalf("expected ## markers stripped with NO_COLOR, got %q", got)
		}
		if !strings.Contains(got, "Results") {
			t.Fatalf("expected 'Results' in output, got %q", got)
		}
	})
}

func TestRenderMarkdown_NoColor_StripsInlineCode(t *testing.T) {
	withEnv("NO_COLOR", "1", func() {
		got := RenderMarkdown("Use `docker ps` here")
		if strings.Contains(got, "`") {
			t.Fatalf("expected backtick markers stripped with NO_COLOR, got %q", got)
		}
		if !strings.Contains(got, "docker ps") {
			t.Fatalf("expected 'docker ps' in output, got %q", got)
		}
	})
}

func TestRenderMarkdown_NoColor_StripsCodeBlock(t *testing.T) {
	withEnv("NO_COLOR", "1", func() {
		got := RenderMarkdown("before\n```\ncode line\n```\nafter")
		if strings.Contains(got, "```") {
			t.Fatalf("expected ``` markers stripped with NO_COLOR, got %q", got)
		}
		if !strings.Contains(got, "code line") {
			t.Fatalf("expected code block content, got %q", got)
		}
	})
}

func TestRenderMarkdown_Bold(t *testing.T) {
	withEnv("NO_COLOR", "", func() {
		withEnv("TERM", "", func() {
			got := RenderMarkdown("This is **important** text")
			if strings.Contains(got, "**") {
				t.Fatalf("expected ** markers stripped, got %q", got)
			}
			if !strings.Contains(got, "\x1b[1m") {
				t.Fatalf("expected bold ANSI, got %q", got)
			}
			if !strings.Contains(got, "important") {
				t.Fatalf("expected 'important' in output, got %q", got)
			}
		})
	})
}

func TestRenderMarkdown_InlineCode(t *testing.T) {
	withEnv("NO_COLOR", "", func() {
		withEnv("TERM", "", func() {
			got := RenderMarkdown("Use `docker ps` to list")
			if strings.Contains(got, "`") {
				t.Fatalf("expected backtick stripped, got %q", got)
			}
			if !strings.Contains(got, "docker ps") {
				t.Fatalf("expected 'docker ps' in output, got %q", got)
			}
			if !strings.Contains(got, "\x1b[90m") {
				t.Fatalf("expected muted ANSI for code, got %q", got)
			}
		})
	})
}

func TestRenderMarkdown_Header(t *testing.T) {
	withEnv("NO_COLOR", "", func() {
		withEnv("TERM", "", func() {
			got := RenderMarkdown("## Results")
			if strings.Contains(got, "##") {
				t.Fatalf("expected ## stripped, got %q", got)
			}
			if !strings.Contains(got, "Results") {
				t.Fatalf("expected 'Results' in output, got %q", got)
			}
			if !strings.Contains(got, "\x1b[1m") {
				t.Fatalf("expected bold in header, got %q", got)
			}
		})
	})
}

func TestRenderMarkdown_List(t *testing.T) {
	withEnv("NO_COLOR", "", func() {
		withEnv("TERM", "", func() {
			got := RenderMarkdown("- first item\n- second item")
			if !strings.Contains(got, "•") {
				t.Fatalf("expected bullet char, got %q", got)
			}
			if !strings.Contains(got, "first item") {
				t.Fatalf("expected list content, got %q", got)
			}
		})
	})
}

func TestRenderMarkdown_ListNoColor(t *testing.T) {
	withEnv("NO_COLOR", "1", func() {
		got := RenderMarkdown("- first item")
		if !strings.Contains(got, "- first item") {
			t.Fatalf("expected dash bullet without color, got %q", got)
		}
	})
}

func TestRenderMarkdown_CodeBlock(t *testing.T) {
	withEnv("NO_COLOR", "", func() {
		withEnv("TERM", "", func() {
			input := "before\n```\ncode line\n```\nafter"
			got := RenderMarkdown(input)
			if !strings.Contains(got, "code line") {
				t.Fatalf("expected code block content, got %q", got)
			}
			if !strings.Contains(got, "\x1b[90m") {
				t.Fatalf("expected muted style for code block, got %q", got)
			}
		})
	})
}

func TestRenderMarkdown_PlainText(t *testing.T) {
	withEnv("NO_COLOR", "", func() {
		withEnv("TERM", "", func() {
			input := "Just plain text without any markdown"
			got := RenderMarkdown(input)
			if got != input {
				t.Fatalf("expected unchanged plain text, got %q", got)
			}
		})
	})
}
