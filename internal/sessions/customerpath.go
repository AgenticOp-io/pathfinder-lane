// Customer path helpers for MSP folder layout (Customers/<name>/…).
package sessions

import "strings"

// CustomerOfFolder returns the customer leaf under Customers/, or "".
// Examples: "Customers/Acme" → "Acme", "Customers/Acme/Core" → "Acme".
func CustomerOfFolder(folder string) string {
	parts := SplitPath(folder)
	if len(parts) < 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], DefaultCustomersRoot) {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// CustomerTag is the vault tag used for per-customer credential sets.
func CustomerTag(customer string) string {
	customer = strings.TrimSpace(customer)
	if customer == "" {
		return ""
	}
	return "customer:" + customer
}
