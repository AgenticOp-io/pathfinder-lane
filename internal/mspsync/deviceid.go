package mspsync

import "strings"

// DeviceIDKey scopes external device ids by integration source so Domotz "42"
// does not collide with Ninja "42" on the same customer tree.
func DeviceIDKey(source, id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		return id
	}
	return source + ":" + id
}
