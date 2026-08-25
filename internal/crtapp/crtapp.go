// Package crtapp is the standalone SecureCRT companion identity (not Pathfinder).
package crtapp

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	Name         = "Lane"
	InstallDir   = "Lane"
	ShortcutBase = "Lane"
	HomeDirName  = ".lane"
)

// Root is %LOCALAPPDATA%\Lane on Windows.
func Root() string {
	if runtime.GOOS == "windows" {
		if base := os.Getenv("LOCALAPPDATA"); base != "" {
			return filepath.Join(base, InstallDir)
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return InstallDir
	}
	return filepath.Join(home, ".lane-app")
}

func BinDir() string { return filepath.Join(Root(), "bin") }

func ExeName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func AgentExe() string { return filepath.Join(BinDir(), ExeName("lane-crt")) }

func LaneExe() string { return filepath.Join(BinDir(), ExeName("lane")) }

func InstallerExe() string { return filepath.Join(BinDir(), ExeName("lane-install")) }

// Home is ~/.lane (config, backups, logs). Independent of Pathfinder.
func Home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return HomeDirName
	}
	dir := filepath.Join(h, HomeDirName)
	_ = os.MkdirAll(dir, 0o755)
	_ = os.MkdirAll(filepath.Join(dir, "logs"), 0o755)
	return dir
}

func LogsDir() string { return filepath.Join(Home(), "logs") }

func LegacyPathfinderHome() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(h, ".pathfinderssh")
}
