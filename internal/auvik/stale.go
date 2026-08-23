package auvik

import (
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/mspsync"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// StaleSession is an Auvik-sourced session no longer present in inventory.
type StaleSession struct {
	Folder string
	Label  string
	Host   string
}

// CollectStale lists Auvik sessions under the customer not in activeDeviceIDs.
func CollectStale(t sessions.Tree, customer, auvikFolder string, activeDeviceIDs map[string]bool) []StaleSession {
	var out []StaleSession
	prefix := sessions.CustomerPath(sessions.DefaultCustomersRoot, customer)
	t.WalkSessions(func(folder string, n sessions.Node) {
		if !isUnderFolder(folder, prefix) {
			return
		}
		id := strings.TrimSpace(n.AuvikDeviceID)
		if id == "" {
			id = strings.TrimSpace(n.ExternalDeviceID)
		}
		if id == "" {
			return
		}
		src := strings.TrimSpace(n.IntegrationSource)
		if src != "" && src != mspsync.SourceAuvik {
			return
		}
		key := mspsync.DeviceIDKey(mspsync.SourceAuvik, id)
		if activeDeviceIDs[key] || activeDeviceIDs[id] {
			return
		}
		out = append(out, StaleSession{
			Folder: folder,
			Label:  n.Label(),
			Host:   n.Host,
		})
	})
	return out
}

// PruneStale removes stale sessions from the tree (Auvik folder only).
func PruneStale(t *sessions.Tree, stale []StaleSession) int {
	removed := 0
	for _, s := range stale {
		if !strings.Contains(strings.ToLower(s.Folder), strings.ToLower(ImportFolder)) {
			continue
		}
		if err := t.Remove(s.Folder, s.Label); err == nil {
			removed++
		}
	}
	return removed
}

// DeviceIDSet builds a lookup set from synced Auvik devices.
func DeviceIDSet(devices []Device) map[string]bool {
	out := make(map[string]bool, len(devices))
	for _, d := range devices {
		id := strings.TrimSpace(d.ID)
		if id == "" {
			continue
		}
		out[id] = true
		out[mspsync.DeviceIDKey(mspsync.SourceAuvik, id)] = true
	}
	return out
}
