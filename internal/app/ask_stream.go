package app

import (
	"fmt"
	"strings"
	"sync"

	"cli/internal/ui"
)

type answerStreamer struct {
	mu        sync.Mutex
	buf       strings.Builder
	answerBuf strings.Builder
	detected  bool
	printing  bool
	printed   bool
	escaped   bool
	spinner   *ui.Spinner
	jsonOut   bool
}

func newAnswerStreamer(spinner *ui.Spinner, jsonOut bool) *answerStreamer {
	return &answerStreamer{spinner: spinner, jsonOut: jsonOut}
}

func (s *answerStreamer) OnToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buf.WriteString(token)

	if s.jsonOut {
		return
	}

	if s.detected {
		if s.printing {
			s.emitAnswerChars(token)
		}
		return
	}

	accumulated := s.buf.String()
	idx := findAnswerValueStart(accumulated)
	if idx < 0 {
		return
	}

	if s.spinner != nil {
		s.spinner.Stop()
	}
	s.detected = true
	s.printing = true
	s.printed = true

	tail := accumulated[idx:]
	if tail != "" {
		s.emitAnswerChars(tail)
	}
}

func (s *answerStreamer) emitAnswerChars(token string) {
	for _, ch := range token {
		if s.escaped {
			s.escaped = false
			switch ch {
			case 'n':
				s.answerBuf.WriteString("\n")
			case 't':
				s.answerBuf.WriteString("\t")
			case '"':
				s.answerBuf.WriteString("\"")
			case '\\':
				s.answerBuf.WriteString("\\")
			default:
				s.answerBuf.WriteRune(ch)
			}
			continue
		}
		if ch == '\\' {
			s.escaped = true
			continue
		}
		if ch == '"' {
			s.printing = false
			return
		}
		s.answerBuf.WriteRune(ch)
	}
}

func (s *answerStreamer) DidStream() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.printed
}

func (s *answerStreamer) Finish() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.printed {
		return
	}
	text := strings.TrimSpace(s.answerBuf.String())
	if text != "" {
		fmt.Println()
		fmt.Println(ui.RenderMarkdown(text))
	}
}

func findAnswerValueStart(s string) int {
	patterns := []string{`"answer":"`, `"answer": "`}
	for _, p := range patterns {
		if idx := strings.Index(s, p); idx >= 0 {
			return idx + len(p)
		}
	}
	return -1
}
