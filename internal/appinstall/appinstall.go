// Package appinstall copies Pathfinder into the per-user AppData install
// location and creates Start Menu / Desktop shortcuts from the executable.
package appinstall

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/scottpeterman/pathfinderssh/internal/product"
)

// Root is %LOCALAPPDATA%\PathfinderSSH-MSP on Windows, or ~/.pathfinderssh-msp-app elsewhere.
func Root() string {
	if runtime.GOOS == "windows" {
		if base := os.Getenv("LOCALAPPDATA"); base != "" {
			return filepath.Join(base, product.InstallDir)
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return product.InstallDir
	}
	return filepath.Join(home, ".pathfinderssh-msp-app")
}

// BinDir holds pathfinder.exe / pfseed.exe.
func BinDir() string { return filepath.Join(Root(), "bin") }

// ExePath is the installed pathfinder binary path.
func ExePath() string {
	name := "pathfinder"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(BinDir(), name)
}

func osExecutable() (string, error) { return os.Executable() }

// RunningInstalled reports whether this process is already the AppData copy.
func RunningInstalled() bool {
	exe, err := osExecutable()
	if err != nil {
		return false
	}
	return SameFile(exe, ExePath())
}

// Ensure copies this executable (and sibling pfseed, if present) into BinDir.
// It does nothing when already running from the install location.
// copied is true when files were written.
func Ensure() (destExe string, copied bool, err error) {
	destExe = ExePath()
	src, err := osExecutable()
	if err != nil {
		return "", false, err
	}
	if SameFile(src, destExe) {
		return destExe, false, nil
	}
	if err := os.MkdirAll(BinDir(), 0o755); err != nil {
		return "", false, err
	}
	needCopy := true
	if stD, err := os.Stat(destExe); err == nil {
		if stS, err := os.Stat(src); err == nil {
			needCopy = stS.Size() != stD.Size() || stS.ModTime().After(stD.ModTime())
		}
	}
	if needCopy {
		if err := copyFile(src, destExe); err != nil {
			return "", false, fmt.Errorf("install pathfinder: %w", err)
		}
		copied = true
	}
	// Sibling tools from the same portable folder.
	srcDir := filepath.Dir(src)
	seedName := "pfseed"
	if runtime.GOOS == "windows" {
		seedName += ".exe"
	}
	srcSeed := filepath.Join(srcDir, seedName)
	if st, err := os.Stat(srcSeed); err == nil && !st.IsDir() {
		dstSeed := filepath.Join(BinDir(), seedName)
		seedNeed := true
		if stD, err := os.Stat(dstSeed); err == nil {
			seedNeed = st.Size() != stD.Size() || st.ModTime().After(stD.ModTime())
		}
		if seedNeed {
			_ = copyFile(srcSeed, dstSeed)
		}
	}
	return destExe, copied, nil
}

func copyFile(src, dst string) error {
	if SameFile(src, dst) {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return closeErr
	}
	_ = os.Remove(dst)
	return os.Rename(tmp, dst)
}
