package appinstall

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// AuvikTunnelExe is Auvik's official tunnel client. Pathfinder launches it as
// a sidecar; we do not reimplement the protocol.
func AuvikTunnelExe() string {
	if runtime.GOOS == "windows" {
		return "AuvikTunnel.exe"
	}
	return "AuvikTunnel"
}

// FindAuvikTunnelBinary locates the official AuvikTunnel executable.
func FindAuvikTunnelBinary(searchDirs ...string) string {
	name := AuvikTunnelExe()
	var candidates []string
	for _, d := range searchDirs {
		d = filepath.Clean(d)
		if d == "" || d == "." {
			continue
		}
		candidates = append(candidates, filepath.Join(d, name))
	}
	home, _ := os.UserHomeDir()
	local := os.Getenv("LOCALAPPDATA")
	candidates = append(candidates,
		filepath.Join(BinDir(), name),
		filepath.Join(local, "PathfinderSSH-MSP", "bin", name),
		filepath.Join(home, "auvik", "Auvik Tunnel", name),
		filepath.Join(local, "PathfinderSSH", "bin", name),
		filepath.Join(local, "Programs", "Auvik", name),
		filepath.Join(local, "Auvik", name),
	)
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

// CopyAuvikTunnelSidecar installs AuvikTunnel.exe next to Pathfinder so the
// MSP package does not depend on a separate Auvik UI install.
func CopyAuvikTunnelSidecar(destDir string, searchDirs ...string) (copied bool, err error) {
	if destDir == "" {
		destDir = BinDir()
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return false, err
	}
	src := FindAuvikTunnelBinary(searchDirs...)
	if src == "" {
		return false, nil
	}
	dst := filepath.Join(destDir, AuvikTunnelExe())
	if SameFile(src, dst) {
		return false, nil
	}
	stS, err := os.Stat(src)
	if err != nil {
		return false, err
	}
	if stD, err := os.Stat(dst); err == nil {
		if stS.Size() == stD.Size() && !stS.ModTime().After(stD.ModTime()) {
			return false, nil
		}
	}
	if err := copyFile(src, dst); err != nil {
		return false, fmt.Errorf("install AuvikTunnel: %w", err)
	}
	return true, nil
}
