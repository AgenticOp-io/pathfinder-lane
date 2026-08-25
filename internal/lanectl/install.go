package lanectl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/appinstall"
	"github.com/scottpeterman/pathfinderssh/internal/crtapp"
)

func isGoRunCache(p string) bool {
	s := filepath.ToSlash(p)
	return strings.Contains(s, "/go-build/")
}

// InstallSelf copies this pflane binary to the stable AppData/home bin
// so OpenSSH ProxyCommand and PuTTY local proxy keep working after a rebuild.
func InstallSelf() (string, error) {
	src, err := os.Executable()
	if err != nil {
		return "", err
	}
	src, err = filepath.Abs(src)
	if err != nil {
		return src, err
	}
	if isGoRunCache(src) {
		return src, fmt.Errorf("running from go run — build pflane and run that binary so ssh/PuTTY have a stable path")
	}
	dest := crtapp.LaneExe()
	if err := copyExe(src, dest); err != nil {
		return "", err
	}
	if st, err := os.Stat(appinstall.BinDir()); err == nil && st.IsDir() {
		_ = copyExe(src, filepath.Join(appinstall.BinDir(), crtapp.ExeName("pflane")))
	}
	_, _ = InstallOnPATH(dest)
	return dest, nil
}

func copyExe(src, dest string) error {
	if appinstall.SameFile(src, dest) {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dest, 0o755)
}
