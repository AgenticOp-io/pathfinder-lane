//go:build windows

package auvik

// killUntrackedAuvikTunnels is intentionally a no-op.
// AuvikTunnel's release supervisor exits after spawning a child with a new PID;
// killing "untracked" processes destroys live tunnels when opening a second one.
func killUntrackedAuvikTunnels(keep map[int]struct{}) {
	_ = keep
}
