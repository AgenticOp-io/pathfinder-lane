package crtapp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/appinstall"
)

func TunnelExe() string {
	return filepath.Join(BinDir(), ExeName("AuvikTunnel"))
}

// CopyBundle installs pathfinder-crt, pfcrt-install, pflane, and AuvikTunnel into
// %LOCALAPPDATA%\PathfinderCRT-Bridge\bin.
func CopyBundle(srcDir string) error {
	if err := os.MkdirAll(BinDir(), 0o755); err != nil {
		return err
	}
	if srcDir == "" {
		if exe, err := os.Executable(); err == nil {
			srcDir = filepath.Dir(exe)
		}
	}
	for _, name := range []string{ExeName("pathfinder-crt"), ExeName("pfcrt-install"), ExeName("pflane")} {
		src := filepath.Join(srcDir, name)
		if !fileExists(src) {
			continue
		}
		dst := filepath.Join(BinDir(), name)
		if appinstall.SameFile(src, dst) {
			continue
		}
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("copy %s: %w", name, err)
		}
	}
	if !fileExists(AgentExe()) {
		for _, dir := range []string{srcDir, appinstall.BinDir()} {
			src := filepath.Join(dir, ExeName("pathfinder-crt"))
			if fileExists(src) && !appinstall.SameFile(src, AgentExe()) {
				if err := copyFile(src, AgentExe()); err != nil {
					return fmt.Errorf("copy pathfinder-crt: %w", err)
				}
				break
			}
		}
	}
	if _, err := appinstall.CopyAuvikTunnelSidecar(BinDir(), srcDir, appinstall.BinDir()); err != nil {
		return err
	}
	if !fileExists(AgentExe()) {
		if self, err := os.Executable(); err == nil && fileExists(self) {
			if strings.EqualFold(filepath.Base(self), ExeName("pathfinder-crt")) {
				if err := copyFile(self, AgentExe()); err != nil {
					return err
				}
			}
		}
	}
	if !fileExists(AgentExe()) {
		return fmt.Errorf("pathfinder-crt.exe not found beside the installer")
	}
	return nil
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
