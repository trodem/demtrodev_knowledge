package ui

import (
	"regexp"
	"strings"
)

var (
	mdBold       = regexp.MustCompile(`\*\*(.+?)\*\*`)
	mdInlineCode = regexp.MustCompile("`([^`]+)`")
	mdHeader     = regexp.MustCompile(`^(#{1,3})\s+(.+)$`)
	mdListItem   = regexp.MustCompile(`^(\s*)[-*]\s+(.+)$`)
	mdNumbered   = regexp.MustCompile(`^(\s*)\d+\.\s+(.+)$`)
	mdHR         = regexp.MustCompile(`^---+$`)
)

// RenderMarkdown converts common markdown elements to terminal-friendly output.
// Syntax markers (**, ##, `, ```) are always stripped.
// ANSI styling is applied only when the terminal supports color.
func RenderMarkdown(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock {
				lang := strings.TrimPrefix(trimmed, "```")
				if lang != "" {
					out = append(out, Muted("  "+lang))
				}
			}
			continue
		}

		if inCodeBlock {
			out = append(out, Muted("  "+line))
			continue
		}

		if mdHR.MatchString(trimmed) {
			out = append(out, Muted("  ────────────────────"))
			continue
		}

		if m := mdHeader.FindStringSubmatch(trimmed); len(m) == 3 {
			out = append(out, bold(Accent(m[2])))
			continue
		}

		if m := mdListItem.FindStringSubmatch(line); len(m) == 3 {
			rendered := renderInline(m[2])
			out = append(out, m[1]+"  "+bullet()+" "+rendered)
			continue
		}

		if m := mdNumbered.FindStringSubmatch(line); len(m) == 3 {
			rendered := renderInline(m[2])
			out = append(out, m[1]+"  "+rendered)
			continue
		}

		out = append(out, renderInline(line))
	}

	return strings.Join(out, "\n")
}

func renderInline(line string) string {
	line = mdInlineCode.ReplaceAllStringFunc(line, func(match string) string {
		inner := mdInlineCode.FindStringSubmatch(match)
		if len(inner) == 2 {
			return Muted(inner[1])
		}
		return match
	})
	line = mdBold.ReplaceAllStringFunc(line, func(match string) string {
		inner := mdBold.FindStringSubmatch(match)
		if len(inner) == 2 {
			return bold(inner[1])
		}
		return match
	})
	return line
}

func bold(text string) string {
	if !supportsColor() {
		return text
	}
	return "\x1b[1m" + text + "\x1b[22m"
}

func bullet() string {
	if !supportsColor() {
		return "-"
	}
	return "•"
}
