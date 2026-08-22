// Linux fingerprint refinement: upgrade a generic "linux" hit to a more
// specific platform when Proxmox / ESXi / libvirt tools answer over SSH.
package netexec

import (
	"context"
	"regexp"
	"strings"
)

var (
	reProxmox = regexp.MustCompile(`(?i)\bpve\b|proxmox|pve-manager`)
	reESXi    = regexp.MustCompile(`(?i)VMware ESXi|VMkernel`)
	reKVM     = regexp.MustCompile(`(?i)\bQEMU\b|\bKVM\b|libvirt`)
)

// refineLinux upgrades fp when secondary SSH probes identify a hypervisor.
// Best-effort: any probe error or rejection leaves the original linux result.
func refineLinux(ctx context.Context, s *Session, fp *Platform) *Platform {
	if fp == nil || fp.Name != "linux" || s == nil {
		return fp
	}

	// Proxmox VE — pveversion is definitive when present.
	if out, err := s.Run(ctx, "pveversion 2>/dev/null || true"); err == nil && !isCLIError(out) {
		if reProxmox.MatchString(out) || strings.Contains(out, "pve-manager/") {
			fp.Name = "proxmox"
			fp.VersionCommand = "pveversion"
			fp.VersionOutput = out
			return fp
		}
	}
	if out, err := s.Run(ctx, "test -d /etc/pve && echo PVE_OK"); err == nil && strings.Contains(out, "PVE_OK") {
		fp.Name = "proxmox"
		fp.VersionCommand = "test -d /etc/pve"
		if vo, err := s.Run(ctx, "cat /etc/pve/.version 2>/dev/null || pveversion 2>/dev/null || true"); err == nil {
			fp.VersionOutput = vo
		}
		return fp
	}

	// VMware ESXi
	if out, err := s.Run(ctx, "vmware -v 2>/dev/null || true"); err == nil && reESXi.MatchString(out) {
		fp.Name = "vmware_esxi"
		fp.VersionCommand = "vmware -v"
		fp.VersionOutput = out
		return fp
	}
	if out, err := s.Run(ctx, "esxcli system version get 2>/dev/null || true"); err == nil && reESXi.MatchString(out) {
		fp.Name = "vmware_esxi"
		fp.VersionCommand = "esxcli system version get"
		fp.VersionOutput = out
		return fp
	}

	// KVM / libvirt host
	if out, err := s.Run(ctx, "virsh version 2>/dev/null || true"); err == nil && !isCLIError(out) && reKVM.MatchString(out) {
		fp.Name = "linux_kvm"
		fp.VersionCommand = "virsh version"
		fp.VersionOutput = out
		return fp
	}
	if out, err := s.Run(ctx, "virsh list --all 2>/dev/null || true"); err == nil && !isCLIError(out) &&
		(strings.Contains(out, "Id") || strings.Contains(out, "Name")) && !strings.Contains(strings.ToLower(out), "command not found") {
		fp.Name = "linux_kvm"
		fp.VersionCommand = "virsh list --all"
		fp.VersionOutput = out
		return fp
	}

	return fp
}

// IsHypervisor reports platforms that host VMs/containers over SSH.
func IsHypervisor(platform string) bool {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "proxmox", "vmware_esxi", "linux_kvm":
		return true
	default:
		return false
	}
}
