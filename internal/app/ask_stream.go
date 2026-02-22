package app

import (
	"fmt"
	"strings"
	"sync"

	"cli/internal/ui"
)

type answerStreamer struct {
	mu          sync.Mutex
	buf         strings.Builder
	detected    bool
	printing    bool
	printed     bool
	escaped     bool
	startedLine bool
	spinner     *ui.Spinner
	jsonOut     bool
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
				s.emit("\n")
			case 't':
				s.emit("\t")
			case '"':
				s.emit("\"")
			case '\\':
				s.emit("\\")
			default:
				s.emit(string(ch))
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
		s.emit(string(ch))
	}
}

func (s *answerStreamer) emit(text string) {
	if !s.startedLine {
		fmt.Println()
		s.startedLine = true
	}
	fmt.Print(text)
}

func (s *answerStreamer) DidStream() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.printed
}

func (s *answerStreamer) Finish() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.printed && s.startedLine {
		fmt.Println()
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
