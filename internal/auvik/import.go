package auvik

import (
	"fmt"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// ImportFolder is the default folder under each customer for synced devices.
const ImportFolder = "Auvik"

// SessionNodes maps Auvik devices to SSH session nodes.
//
// Passwords are never available from Auvik's API — nodes use agent auth by default.
// Assign vault credentials per customer folder after import, or set Username on
// nodes manually.
func SessionNodes(devices []Device, opts ImportOptions) ([]sessions.Node, ImportStats) {
	opts = opts.withDefaults()
	var out []sessions.Node
	st := ImportStats{}
	for _, d := range devices {
		if !opts.WantDevice(d) {
			st.Skipped++
			continue
		}
		host := d.PrimaryIP()
		if host == "" {
			st.NoIP++
			continue
		}
		name := d.Name
		if name == "" {
			name = host
		}
		n := sessions.Node{
			Name:          name,
			Transport:     sessions.TransportSSH,
			Host:          host,
			Port:          sessions.TransportSSH.DefaultPort(),
			Username:      strings.TrimSpace(opts.DefaultUsername),
			AuthType:      sessions.AuthAgent,
			HostKeyPolicy: sessions.HostKeyTOFU,
			Vendor:        strings.TrimSpace(d.Vendor),
			DeviceType:    strings.TrimSpace(d.DeviceType),
		}
		if opts.DefaultUsername != "" && opts.DefaultCredential != "" {
			n.AuthType = sessions.AuthPassword
			n.Credential = opts.DefaultCredential
		}
		out = append(out, n)
		st.Imported++
	}
	return out, st
}

// ImportOptions filters and defaults for session import.
type ImportOptions struct {
	// NetworkGearOnly keeps switches/routers/firewalls/APs plus servers/VMs/hypervisors;
	// skips end-user gear (workstations, printers, phones, cameras, etc.).
	NetworkGearOnly bool
	// RequireLoginAuthorized skips devices whose Auvik login status is not authorized/privileged.
	RequireLoginAuthorized bool
	DefaultUsername        string
	DefaultCredential      string // vault credential name (optional)
}

func (o ImportOptions) withDefaults() ImportOptions {
	return o
}

// WantDevice applies import filters.
func (o ImportOptions) WantDevice(d Device) bool {
	if o.RequireLoginAuthorized {
		switch strings.ToLower(strings.TrimSpace(d.LoginStatus)) {
		case "":
			// device/info include=deviceDetail does not supply login status — do not drop unknowns.
		case "authorized", "privileged":
		default:
			return false
		}
	}
	if o.NetworkGearOnly && !isSyncDeviceType(d.DeviceType) {
		return false
	}
	return true
}

// isSyncDeviceType matches Auvik deviceType values we import for MSP SSH inventory.
// Names are compared lowercased (Auvik uses camelCase in the API).
func isSyncDeviceType(deviceType string) bool {
	switch strings.ToLower(strings.TrimSpace(deviceType)) {
	// L2/L3 network fabric
	case "switch", "l3switch", "router", "accesspoint", "thinaccesspoint",
		"firewall", "securityappliance", "utm", "loadbalancer", "controller",
		"hub", "bridge", "modem", "stack", "voipswitch", "packetprocessor",
		"chassis", "backhaul", "telecommunications":
		return true
	// Virtual systems / hosts / guests / BMC
	case "hypervisor", "virtualmachine", "virtualappliance", "server", "ipmi", "storage":
		return true
	default:
		return false
	}
}

// PrimaryIP returns the first usable management address.
func (d Device) PrimaryIP() string {
	for _, ip := range d.IPs {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			return ip
		}
	}
	return ""
}

// ImportStats summarizes a sync pass.
type ImportStats struct {
	Imported int
	Skipped  int
	NoIP     int
	Errors   []string
	Notes    []string
}

func (s ImportStats) Summary() string {
	return fmt.Sprintf("imported %d, skipped %d, no IP %d", s.Imported, s.Skipped, s.NoIP)
}

// ErrorCount is the number of hard failures (not informational notes).
func (s ImportStats) ErrorCount() int {
	return len(s.Errors)
}
