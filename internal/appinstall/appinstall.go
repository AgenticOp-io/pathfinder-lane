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
		// Still refresh LICENSE/NOTICE when already installed.
		if err := InstallLegalDocs(BinDir()); err != nil {
			return destExe, false, fmt.Errorf("install legal docs: %w", err)
		}
		return destExe, false, nil
	}
	if err := os.MkdirAll(BinDir(), 0o755); err != nil {
		return "", false, err
	}
	srcDir := filepath.Dir(src)
	bundleCopied, err := CopyToolBundle(srcDir, BinDir())
	if err != nil {
		return "", false, err
	}

	needCopy := true
	if stD, err := os.Stat(destExe); err == nil {
		mainSrc := filepath.Join(srcDir, exeName("pathfinder"))
		if stS, err := os.Stat(mainSrc); err == nil {
			needCopy = stS.Size() != stD.Size() || stS.ModTime().After(stD.ModTime())
		} else if isPathfinderExe(src) {
			if stS, err := os.Stat(src); err == nil {
				needCopy = stS.Size() != stD.Size() || stS.ModTime().After(stD.ModTime())
			}
		} else {
			needCopy = false
		}
	}
	if needCopy {
		mainSrc := filepath.Join(srcDir, exeName("pathfinder"))
		if st, err := os.Stat(mainSrc); err == nil && !st.IsDir() {
			if err := copyFile(mainSrc, destExe); err != nil {
				return "", false, fmt.Errorf("install pathfinder: %w", err)
			}
			copied = true
		} else if isPathfinderExe(src) {
			if err := copyFile(src, destExe); err != nil {
				return "", false, fmt.Errorf("install pathfinder: %w", err)
			}
			copied = true
		} else if _, err := os.Stat(destExe); err != nil {
			return "", false, fmt.Errorf("pathfinder.exe not found beside %s — build the app bundle first", src)
		}
	}
	if bundleCopied {
		copied = true
	}
	// Legacy: pfseed beside portable folder (handled by CopyToolBundle now).
	// Always refresh LICENSE/NOTICE beside the installed exe (GPL attribution).
	if err := InstallLegalDocs(BinDir()); err != nil {
		return destExe, copied, fmt.Errorf("install legal docs: %w", err)
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
