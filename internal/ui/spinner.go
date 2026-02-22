package ui

import (
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

type Spinner struct {
	mu      sync.Mutex
	active  bool
	done    chan struct{}
	exited  chan struct{}
	message string
}

func NewSpinner(message string) *Spinner {
	return &Spinner{message: message}
}

func (s *Spinner) Start() {
	if !term.IsTerminal(int(os.Stderr.Fd())) && !term.IsTerminal(int(os.Stdout.Fd())) {
		return
	}
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.done = make(chan struct{})
	s.exited = make(chan struct{})
	s.mu.Unlock()

	go func() {
		defer close(s.exited)
		frames := []string{"|", "/", "-", "\\"}
		i := 0
		for {
			select {
			case <-s.done:
				fmt.Fprint(os.Stderr, "\r\033[K")
				return
			default:
				label := Muted(fmt.Sprintf("\r  %s %s", frames[i%len(frames)], s.message))
				fmt.Fprint(os.Stderr, label)
				i++
				time.Sleep(120 * time.Millisecond)
			}
		}
	}()
}

func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	close(s.done)
	exited := s.exited
	s.mu.Unlock()
	<-exited
}
