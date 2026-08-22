package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// noteInputDebug appends one line to ~/.pathfinderssh/input-debug.log so we can
// tell whether keystrokes reach writeOverride and whether the transport Write
// succeeds. Cheap and best-effort; never blocks the UI for long.
var (
	inputDebugMu   sync.Mutex
	inputDebugPath string
)

func noteInputDebug(kind string, data []byte, connected bool) {
	inputDebugMu.Lock()
	defer inputDebugMu.Unlock()
	if inputDebugPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		dir := filepath.Join(home, AppHomeDir)
		_ = os.MkdirAll(dir, 0o755)
		inputDebugPath = filepath.Join(dir, "input-debug.log")
	}
	f, err := os.OpenFile(inputDebugPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	preview := data
	if len(preview) > 32 {
		preview = preview[:32]
	}
	_, _ = fmt.Fprintf(f, "%s kind=%s connected=%v n=%d data=%q\n",
		time.Now().Format("15:04:05.000"), kind, connected, len(data), preview)
}
