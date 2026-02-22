package app

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
)

var (
	signalOnce    sync.Once
	cleanupFuncs  []func()
	cleanupMu     sync.Mutex
)

func setupSignalHandler() {
	signalOnce.Do(func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		go func() {
			<-sigCh
			fmt.Fprintln(os.Stderr, "\nInterrupted. Cleaning up...")
			runCleanup()
			os.Exit(130)
		}()
	})
}

func runCleanup() {
	cleanupMu.Lock()
	fns := make([]func(), len(cleanupFuncs))
	copy(fns, cleanupFuncs)
	cleanupMu.Unlock()

	for _, fn := range fns {
		fn()
	}
	cleanTempPluginFiles()
}

func cleanTempPluginFiles() {
	tmpDir := os.TempDir()
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "dm-plugin-") && strings.HasSuffix(name, ".ps1") {
			_ = os.Remove(filepath.Join(tmpDir, name))
		}
	}
}
