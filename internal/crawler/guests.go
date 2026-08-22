// SSH guest inventory for hypervisors: list VMs/containers and best-effort IPs
// so they appear as peers under the hypervisor on the topology map.
package crawler

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/netexec"
	"github.com/scottpeterman/pathfinderssh/internal/topo"
)

var (
	reQMList = regexp.MustCompile(`(?m)^\s*(\d+)\s+(\S+)\s+(\S+)`)
	rePCTList = regexp.MustCompile(`(?m)^\s*(\d+)\s+(\S+)\s+(\S+)`)
	reVirshList = regexp.MustCompile(`(?m)^\s*(\d+|-)\s+(\S+)\s+(\S+)`)
	reESXiVM = regexp.MustCompile(`(?m)^(\d+)\s+(.+?)\s+\[`)
	reIPv4 = regexp.MustCompile(`\b((?:25[0-5]|2[0-4]\d|[01]?\d\d?)(?:\.(?:25[0-5]|2[0-4]\d|[01]?\d\d?)){3})\b`)
	reQMIPConfig = regexp.MustCompile(`(?i)ip=([0-9.]+)`)
)

// collectGuests runs platform-specific SSH inventory and returns containment
// neighbors (Protocol "guest"). Best-effort: failures yield an empty list.
func collectGuests(ctx context.Context, sess *netexec.Session, platform string) []topo.Neighbor {
	if sess == nil {
		return nil
	}
	switch platform {
	case "proxmox":
		return collectProxmoxGuests(ctx, sess)
	case "linux_kvm":
		return collectVirshGuests(ctx, sess)
	case "vmware_esxi":
		return collectESXiGuests(ctx, sess)
	default:
		return nil
	}
}

func collectProxmoxGuests(ctx context.Context, sess *netexec.Session) []topo.Neighbor {
	var out []topo.Neighbor
	if raw, err := sess.Run(ctx, "qm list 2>/dev/null || true"); err == nil {
		for _, line := range strings.Split(raw, "\n") {
			m := reQMList.FindStringSubmatch(line)
			if m == nil || m[1] == "VMID" {
				continue
			}
			vmid, name, status := m[1], m[2], m[3]
			if strings.EqualFold(name, "NAME") {
				continue
			}
			ip := proxmoxQMUIP(ctx, sess, vmid, status)
			out = append(out, guestNeighbor(name, ip, "linux", "qemu:"+vmid))
		}
	}
	if raw, err := sess.Run(ctx, "pct list 2>/dev/null || true"); err == nil {
		for _, line := range strings.Split(raw, "\n") {
			m := rePCTList.FindStringSubmatch(line)
			if m == nil || m[1] == "VMID" {
				continue
			}
			vmid, name := m[1], m[2]
			if strings.EqualFold(name, "NAME") {
				continue
			}
			ip := proxmoxPCTIP(ctx, sess, vmid)
			out = append(out, guestNeighbor(name, ip, "linux", "lxc:"+vmid))
		}
	}
	return dedupeGuests(out)
}

func proxmoxQMUIP(ctx context.Context, sess *netexec.Session, vmid, status string) string {
	// Guest agent (when installed).
	if strings.EqualFold(status, "running") {
		if raw, err := sess.Run(ctx, "qm guest cmd "+vmid+" network-get-interfaces 2>/dev/null || true"); err == nil {
			if ip := firstPublicishIPv4FromJSON(raw); ip != "" {
				return ip
			}
			if ip := firstPublicishIPv4(raw); ip != "" {
				return ip
			}
		}
	}
	// Cloud-init / static ipconfig in VM config.
	if raw, err := sess.Run(ctx, "qm config "+vmid+" 2>/dev/null || true"); err == nil {
		if m := reQMIPConfig.FindStringSubmatch(raw); m != nil {
			return m[1]
		}
	}
	return ""
}

func proxmoxPCTIP(ctx context.Context, sess *netexec.Session, vmid string) string {
	if raw, err := sess.Run(ctx, "pct config "+vmid+" 2>/dev/null || true"); err == nil {
		if m := reQMIPConfig.FindStringSubmatch(raw); m != nil {
			return m[1]
		}
		if m := reIPv4.FindStringSubmatch(raw); m != nil && !strings.HasPrefix(m[1], "127.") {
			return m[1]
		}
	}
	if raw, err := sess.Run(ctx, "pct exec "+vmid+" -- ip -4 -o addr show 2>/dev/null || true"); err == nil {
		return firstPublicishIPv4(raw)
	}
	return ""
}

func collectVirshGuests(ctx context.Context, sess *netexec.Session) []topo.Neighbor {
	raw, err := sess.Run(ctx, "virsh list --all 2>/dev/null || true")
	if err != nil {
		return nil
	}
	var out []topo.Neighbor
	for _, line := range strings.Split(raw, "\n") {
		m := reVirshList.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[2]
		if strings.EqualFold(name, "Name") || name == "-" {
			continue
		}
		ip := ""
		if addr, err := sess.Run(ctx, "virsh domifaddr "+shellQuote(name)+" 2>/dev/null || true"); err == nil {
			ip = firstPublicishIPv4(addr)
		}
		out = append(out, guestNeighbor(name, ip, "linux", "kvm"))
	}
	return dedupeGuests(out)
}

func collectESXiGuests(ctx context.Context, sess *netexec.Session) []topo.Neighbor {
	raw, err := sess.Run(ctx, "vim-cmd vmsvc/getallvms 2>/dev/null || true")
	if err != nil {
		return nil
	}
	var out []topo.Neighbor
	for _, line := range strings.Split(raw, "\n") {
		m := reESXiVM.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		vmid, name := m[1], strings.TrimSpace(m[2])
		if strings.EqualFold(vmid, "Vmid") {
			continue
		}
		ip := ""
		if g, err := sess.Run(ctx, "vim-cmd vmsvc/get.guest "+vmid+" 2>/dev/null || true"); err == nil {
			ip = firstPublicishIPv4(g)
		}
		out = append(out, guestNeighbor(name, ip, "linux", "esxi:"+vmid))
	}
	return dedupeGuests(out)
}

func guestNeighbor(name, ip, platform, localIf string) topo.Neighbor {
	name = strings.TrimSpace(name)
	if name == "" {
		name = ip
	}
	if name == "" {
		name = "guest"
	}
	return topo.Neighbor{
		LocalInterface:  localIf,
		RemoteDevice:    name,
		RemoteInterface: "guest",
		RemoteIP:        ip,
		RemotePlatform:  platform,
		RemoteDescr:     "SSH guest inventory",
		Protocol:        "guest",
	}
}

func dedupeGuests(in []topo.Neighbor) []topo.Neighbor {
	seen := map[string]bool{}
	var out []topo.Neighbor
	for _, n := range in {
		key := strings.ToLower(n.RemoteDevice) + "|" + n.RemoteIP
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, n)
	}
	return out
}

func firstPublicishIPv4(s string) string {
	for _, m := range reIPv4.FindAllStringSubmatch(s, -1) {
		ip := m[1]
		if strings.HasPrefix(ip, "127.") || strings.HasPrefix(ip, "169.254.") {
			continue
		}
		return ip
	}
	return ""
}

func firstPublicishIPv4FromJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if s[0] != '[' && s[0] != '{' {
		return firstPublicishIPv4(s)
	}
	return firstPublicishIPv4(s) // agent JSON still embeds dotted-quad strings
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if regexp.MustCompile(`^[A-Za-z0-9._-]+$`).MatchString(s) {
		return s
	}
	return strconv.Quote(s)
}
