package crawler

import (
	"regexp"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/topo"
)

// parseMikroTikNeighbors parses `/ip neighbor print detail` (RouterOS).
// Blocks are separated by blank lines; fields look like "identity=foo".
func parseMikroTikNeighbors(output string) []topo.Neighbor {
	blocks := regexp.MustCompile(`\n\s*\n`).Split(output, -1)
	var out []topo.Neighbor
	for _, b := range blocks {
		b = strings.TrimSpace(b)
		if b == "" || strings.HasPrefix(b, "Flags:") {
			continue
		}
		identity := mikrotikField(b, "identity")
		address := mikrotikField(b, "address")
		platform := mikrotikField(b, "platform")
		iface := mikrotikField(b, "interface")
		mac := mikrotikField(b, "mac-address")
		if identity == "" {
			identity = mikrotikField(b, "system-description")
		}
		if identity == "" {
			identity = address
		}
		if identity == "" {
			identity = mac
		}
		if identity == "" {
			continue
		}
		if platform == "" {
			platform = "unknown"
		}
		out = append(out, topo.Neighbor{
			LocalInterface:  iface,
			RemoteDevice:    identity,
			RemoteInterface: mac,
			RemoteIP:        address,
			RemotePlatform:  platform,
			RemoteDescr:     mikrotikField(b, "system-description"),
			Protocol:        "discovery",
		})
	}
	return out
}

func mikrotikField(block, key string) string {
	// "identity=foo bar" or "identity="foo""
	re := regexp.MustCompile(`(?m)` + regexp.QuoteMeta(key) + `=(?:"([^"]*)"|(\S+))`)
	m := re.FindStringSubmatch(block)
	if m == nil {
		return ""
	}
	if m[1] != "" {
		return m[1]
	}
	return m[2]
}
