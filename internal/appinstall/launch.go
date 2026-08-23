package appinstall

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/scottpeterman/pathfinderssh/internal/winexec"
)

// ToolPath returns the installed path for a bundled tool name (e.g. pathfinder, pfsetup-apis).
func ToolPath(name string) string {
	return filepath.Join(BinDir(), exeName(name))
}

// LaunchTool starts an installed bundled exe when present.
func LaunchTool(name string) error {
	exe := ToolPath(name)
	if st, err := os.Stat(exe); err != nil || st.IsDir() {
		return fmt.Errorf("%s not installed in %s", exeName(name), BinDir())
	}
	cmd := winexec.Command(exe)
	cmd.Dir = Root()
	return cmd.Start()
}
