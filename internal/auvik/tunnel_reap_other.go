//go:build !windows

package auvik

func killUntrackedAuvikTunnels(keep map[int]struct{}) {
	// No-op outside Windows; AuvikTunnel.exe singleton behavior is Windows-specific.
	_ = keep
}
