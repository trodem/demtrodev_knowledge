package ui

import (
	"fmt"
	"sync"
	"time"
)

type Spinner struct {
	mu      sync.Mutex
	active  bool
	done    chan struct{}
	message string
}

func NewSpinner(message string) *Spinner {
	return &Spinner{message: message}
}

func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.done = make(chan struct{})
	s.mu.Unlock()

	go func() {
		frames := []string{"|", "/", "-", "\\"}
		i := 0
		for {
			select {
			case <-s.done:
				fmt.Print("\r\033[K")
				return
			default:
				label := Muted(fmt.Sprintf("\r%s %s", frames[i%len(frames)], s.message))
				fmt.Print(label)
				i++
				time.Sleep(120 * time.Millisecond)
			}
		}
	}()
}

func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return
	}
	s.active = false
	close(s.done)
}
