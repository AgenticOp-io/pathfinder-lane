// Package mspsync defines how external MSP systems plug into PathfinderSSH MSP.
//
// Cohesive loop (only integrations that strengthen this ship):
//
//  1. Customers — PSA creates Customers/<name>/ folders (ConnectWise, Autotask, Halo, file JSON)
//  2. Inventory — RMM/monitoring adds SSH targets + updates IPs (Auvik, Domotz, Ninja, Datto RMM, Automate, N-central)
//  3. Credentials — documentation vault fills username/password (IT Glue, Hudu, Passportal)
//
// Session nodes merge on external_device_id, host, or name. Vault entries link by host/name.
// Systems that do not provide live IPs or SSH credentials (e.g. config-audit snapshots) are excluded.
package mspsync

// Inventory integration source names (session IntegrationSource field).
const (
	SourceAuvik    = "auvik"
	SourceDomotz   = "domotz"
	SourceNinja    = "ninja"
	SourceDattoRMM = "dattormm"
	SourceAutomate = "automate"
	SourceNcentral = "ncentral"
)

// Inventory folder names under Customers/<client>/.
const (
	FolderAuvik     = "Auvik"
	FolderDomotz    = "Domotz"
	FolderNinja     = "Ninja"
	FolderDattoRMM  = "DattoRMM"
	FolderAutomate  = "Automate"
	FolderNcentral  = "N-central"
)

// Doc vault tag prefixes for credential re-sync.
const (
	TagITGlue      = "itglue"
	TagITGlueID    = "itglue-id:"
	TagHudu        = "hudu"
	TagHuduID      = "hudu-id:"
	TagPassportal  = "passportal"
	TagPassportalID = "passportal-id:"
)
