package ui

// mergeEngineerSettings applies MSP integration fields from admin setup without wiping user layout prefs.
func mergeEngineerSettings(base, staged Settings) Settings {
	out := base
	out.AuvikUsername = staged.AuvikUsername
	out.AuvikAPIKey = staged.AuvikAPIKey
	out.AuvikBaseURL = staged.AuvikBaseURL
	out.AuvikSyncEnabled = staged.AuvikSyncEnabled
	out.AuvikSyncIntervalMin = staged.AuvikSyncIntervalMin
	out.AuvikTunnelPath = staged.AuvikTunnelPath
	out.AuvikAutoTunnel = staged.AuvikAutoTunnel
	out.AuvikPruneStale = staged.AuvikPruneStale
	out.AuvikDefaultUsername = staged.AuvikDefaultUsername
	out.AuvikDefaultCredential = staged.AuvikDefaultCredential
	out.ITGlueAPIKey = staged.ITGlueAPIKey
	out.ITGlueBaseURL = staged.ITGlueBaseURL
	out.HuduAPIKey = staged.HuduAPIKey
	out.HuduBaseURL = staged.HuduBaseURL
	out.PassportalAPIKey = staged.PassportalAPIKey
	out.PassportalTenant = staged.PassportalTenant
	out.PassportalBaseURL = staged.PassportalBaseURL
	out.ConnectWiseCompanyID = staged.ConnectWiseCompanyID
	out.ConnectWisePublicKey = staged.ConnectWisePublicKey
	out.ConnectWisePrivateKey = staged.ConnectWisePrivateKey
	out.ConnectWiseClientID = staged.ConnectWiseClientID
	out.ConnectWiseBaseURL = staged.ConnectWiseBaseURL
	out.AutotaskUsername = staged.AutotaskUsername
	out.AutotaskSecret = staged.AutotaskSecret
	out.AutotaskAPIIntegrationCode = staged.AutotaskAPIIntegrationCode
	out.AutotaskBaseURL = staged.AutotaskBaseURL
	out.HaloClientID = staged.HaloClientID
	out.HaloClientSecret = staged.HaloClientSecret
	out.HaloTenant = staged.HaloTenant
	out.HaloBaseURL = staged.HaloBaseURL
	out.DomotzAPIKey = staged.DomotzAPIKey
	out.DomotzBaseURL = staged.DomotzBaseURL
	out.NinjaClientID = staged.NinjaClientID
	out.NinjaClientSecret = staged.NinjaClientSecret
	out.NinjaBaseURL = staged.NinjaBaseURL
	out.DattoAPIKey = staged.DattoAPIKey
	out.DattoSecret = staged.DattoSecret
	out.DattoBaseURL = staged.DattoBaseURL
	out.AutomateUsername = staged.AutomateUsername
	out.AutomatePassword = staged.AutomatePassword
	out.AutomateServerURL = staged.AutomateServerURL
	out.NcentralJWT = staged.NcentralJWT
	out.NcentralServerURL = staged.NcentralServerURL
	out.PagerDutyAPIKey = staged.PagerDutyAPIKey
	out.PagerDutyBaseURL = staged.PagerDutyBaseURL
	out.OpsgenieAPIKey = staged.OpsgenieAPIKey
	out.OpsgenieBaseURL = staged.OpsgenieBaseURL
	out.MSPInventoryDefUsername = staged.MSPInventoryDefUsername
	out.MSPInventoryDefCredential = staged.MSPInventoryDefCredential
	out.CursorAPIKey = staged.CursorAPIKey
	out.TroubleshootAddon = staged.TroubleshootAddon
	return out.Normalized()
}
