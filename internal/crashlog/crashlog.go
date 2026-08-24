// Package crashlog writes recovered panics to disk so a windowsgui build
// (no console) still leaves a stack after a hard failure.
package crashlog

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
)

const fileName = "panic.log"

// Write records a recovered panic. Safe to call from any goroutine.
func Write(appHome, where string, recovered any) {
	stack := strings.TrimSpace(string(debug.Stack()))
	msg := fmt.Sprintf("%s panic in %s: %v\n%s\n",
		time.Now().Format(time.RFC3339), strings.TrimSpace(where), recovered, stack)
	log.Print(msg)

	if strings.TrimSpace(appHome) == "" {
		return
	}
	dir := filepath.Join(appHome, "logs")
	_ = os.MkdirAll(dir, 0o755)
	f, err := os.OpenFile(filepath.Join(dir, fileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, _ = f.WriteString(msg + "\n")
	_ = f.Close()
}

// Error converts a recovered panic into an operator-facing error.
func Error(where string, recovered any) error {
	return fmt.Errorf("internal error while %s — the app stayed up; try again. (%v)", where, recovered)
}
