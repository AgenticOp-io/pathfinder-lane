package auvik

import "strings"
type Tenant struct {
	ID   string
	Name string // domainPrefix when present
}

// Device is inventory row used for Pathfinder session import.
type Device struct {
	ID           string
	Name         string
	IPs          []string
	DeviceType   string
	Vendor       string
	TenantID     string
	OnlineStatus string
	LoginStatus  string // authorized / notAuthorized / … when discovery status included
}

type tenantsEnvelope struct {
	Data []tenantResource `json:"data"`
}

type tenantResource struct {
	ID         string `json:"id"`
	Attributes struct {
		DomainPrefix string `json:"domainPrefix"`
	} `json:"attributes"`
}

type devicesEnvelope struct {
	Data     []deviceResource `json:"data"`
	Included []includedResource `json:"included"`
	Links    struct {
		Next string `json:"next"`
	} `json:"links"`
}

type deviceResource struct {
	ID         string `json:"id"`
	Attributes struct {
		IPAddresses  []string `json:"ipAddresses"`
		DeviceName   string   `json:"deviceName"`
		DeviceType   string   `json:"deviceType"`
		VendorName   string   `json:"vendorName"`
		OnlineStatus string   `json:"onlineStatus"`
	} `json:"attributes"`
	Relationships struct {
		Tenant struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"tenant"`
		DeviceDiscoveryStatus struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"deviceDiscoveryStatus"`
	} `json:"relationships"`
}

type includedResource struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Attributes struct {
		Login string `json:"login"`
	} `json:"attributes"`
}

func decodeDevice(r deviceResource, included []includedResource) Device {
	d := Device{
		ID:           r.ID,
		Name:         stringsTrim(r.Attributes.DeviceName),
		IPs:          append([]string(nil), r.Attributes.IPAddresses...),
		DeviceType:   r.Attributes.DeviceType,
		Vendor:       r.Attributes.VendorName,
		TenantID:     r.Relationships.Tenant.Data.ID,
		OnlineStatus: r.Attributes.OnlineStatus,
	}
	statusID := r.Relationships.DeviceDiscoveryStatus.Data.ID
	for _, inc := range included {
		if inc.Type == "deviceDiscoveryStatus" && inc.ID == statusID {
			d.LoginStatus = inc.Attributes.Login
			break
		}
	}
	if d.Name == "" && len(d.IPs) > 0 {
		d.Name = d.IPs[0]
	}
	return d
}

func stringsTrim(s string) string {
	return strings.TrimSpace(s)
}
