package invsync

// Device is a normalized inventory record from any RMM/monitoring API.
type Device struct {
	ID         string
	Name       string
	Host       string
	Vendor     string
	DeviceType string
}

// Source names the integration (domotz, ninja, dattormm, automate, ncentral, auvik).
const (
	SourceAuvik    = "auvik"
	SourceDomotz   = "domotz"
	SourceNinja    = "ninja"
	SourceDattoRMM = "dattormm"
	SourceAutomate = "automate"
	SourceNcentral = "ncentral"
)
