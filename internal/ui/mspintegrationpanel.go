package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// MSPIntegrationActions are File-tab import/sync handlers (all MSP enrollments).
type MSPIntegrationActions struct {
	OnImportAuvik      func()
	OnSyncAuvik        func()
	OnImportITGlue     func()
	OnImportHudu       func()
	OnImportPassportal func()
	OnSyncConnectWise  func()
	OnSyncAutotask     func()
	OnSyncHalo         func()
	OnSyncCustomers    func()
	OnImportDomotz     func()
	OnImportNinja      func()
	OnImportDatto      func()
	OnImportAutomate   func()
	OnImportNcentral   func()

	OnBindWorkContext  func()
	OnClearWorkContext func()
	OnDocumentWork     func()
	OnExportHandoff    func()
	OnOpenNOCMap       func()
}

// MSPIntegrationPanel holds Tools-tab widgets for MSP integrations.
type MSPIntegrationPanel struct {
	defUser  *widget.Entry
	defCred  *widget.Entry

	auvikUser     *widget.Entry
	auvikKey      *widget.Entry
	auvikBase     *widget.Entry
	auvikSync     *widget.Check
	auvikInterval *widget.Entry
	auvikTunnel   *widget.Entry
	auvikAutoTun  *widget.Check
	auvikPrune    *widget.Check

	itglueKey  *widget.Entry
	itglueBase *widget.Entry

	huduKey  *widget.Entry
	huduBase *widget.Entry

	passKey    *widget.Entry
	passTenant *widget.Entry
	passBase   *widget.Entry

	cwCompany  *widget.Entry
	cwPublic   *widget.Entry
	cwPrivate  *widget.Entry
	cwClientID *widget.Entry
	cwBase     *widget.Entry

	atUser  *widget.Entry
	atSecret *widget.Entry
	atCode   *widget.Entry
	atBase   *widget.Entry

	haloClientID     *widget.Entry
	haloClientSecret *widget.Entry
	haloTenant       *widget.Entry
	haloBase         *widget.Entry

	domotzKey  *widget.Entry
	domotzBase *widget.Entry

	ninjaClientID     *widget.Entry
	ninjaClientSecret *widget.Entry
	ninjaBase         *widget.Entry

	dattoKey  *widget.Entry
	dattoSecret *widget.Entry
	dattoBase   *widget.Entry

	automateUser   *widget.Entry
	automatePass   *widget.Entry
	automateServer *widget.Entry

	ncentralJWT   *widget.Entry
	ncentralServer *widget.Entry

	pdKey  *widget.Entry
	pdBase *widget.Entry

	ogKey  *widget.Entry
	ogBase *widget.Entry
}

func newMSPIntegrationPanel() *MSPIntegrationPanel {
	p := &MSPIntegrationPanel{}
	p.defUser = entry("Default SSH username for synced devices")
	p.defCred = entry("Default vault credential name")

	p.auvikUser = entry("Auvik user email")
	p.auvikKey = entry("API key")
	p.auvikKey.Password = true
	p.auvikBase = entry("https://auvikapi.us1.my.auvik.com")
	p.auvikSync = widget.NewCheck("Periodic Auvik inventory sync (all clients)", nil)
	p.auvikInterval = entry("60")
	p.auvikTunnel = entry("Path to AuvikTunnel.exe (optional)")
	p.auvikAutoTun = widget.NewCheck("Try AuvikTunnel when direct SSH fails", nil)
	p.auvikPrune = widget.NewCheck("Prune stale Auvik sessions on sync", nil)

	p.itglueKey = entry("ITG.… API key")
	p.itglueKey.Password = true
	p.itglueBase = entry("https://api.itglue.com")

	p.huduKey = entry("API key")
	p.huduKey.Password = true
	p.huduBase = entry("https://api.hudu.com")

	p.passKey = entry("API key / bearer token")
	p.passKey.Password = true
	p.passTenant = entry("Tenant id (optional)")
	p.passBase = entry("https://api.passportal.com")

	p.cwCompany = entry("Company ID")
	p.cwPublic = entry("Public key")
	p.cwPrivate = entry("Private key")
	p.cwPrivate.Password = true
	p.cwClientID = entry("Client ID (optional)")
	p.cwBase = entry("https://na.myconnectwise.net/v4_6_release/apis/3.0")

	p.atUser = entry("API user")
	p.atSecret = entry("Secret")
	p.atSecret.Password = true
	p.atCode = entry("API integration code")
	p.atBase = entry("https://webservices.autotask.net/atservicesrest/v1.0")

	p.haloClientID = entry("Client ID")
	p.haloClientSecret = entry("Client secret")
	p.haloClientSecret.Password = true
	p.haloTenant = entry("Tenant subdomain")
	p.haloBase = entry("https://yourtenant.halopsa.com")

	p.domotzKey = entry("API key")
	p.domotzKey.Password = true
	p.domotzBase = entry("https://api.domotz.com/public-api/v1")

	p.ninjaClientID = entry("Client ID")
	p.ninjaClientSecret = entry("Client secret")
	p.ninjaClientSecret.Password = true
	p.ninjaBase = entry("https://api.ninjarmm.com/v2")

	p.dattoKey = entry("API key")
	p.dattoKey.Password = true
	p.dattoSecret = entry("Secret")
	p.dattoSecret.Password = true
	p.dattoBase = entry("https://zinfandel-api.centrastage.net/api/v2")

	p.automateUser = entry("Username")
	p.automatePass = entry("Password")
	p.automatePass.Password = true
	p.automateServer = entry("https://yourserver.hostedrmm.com")

	p.ncentralJWT = entry("JWT token")
	p.ncentralJWT.Password = true
	p.ncentralServer = entry("https://yourserver.n-able.com")

	p.pdKey = entry("REST API key")
	p.pdKey.Password = true
	p.pdBase = entry("https://api.pagerduty.com")

	p.ogKey = entry("REST API key")
	p.ogKey.Password = true
	p.ogBase = entry("https://api.opsgenie.com")

	return p
}

func (p *MSPIntegrationPanel) load(v SettingsFields) {
	p.defUser.SetText(v.MSPDefUsername)
	p.defCred.SetText(v.MSPDefCred)

	p.auvikUser.SetText(v.AuvikUsername)
	p.auvikKey.SetText(v.AuvikAPIKey)
	p.auvikBase.SetText(v.AuvikBaseURL)
	p.auvikSync.SetChecked(v.AuvikSyncEnabled)
	p.auvikInterval.SetText(v.AuvikSyncInterval)
	p.auvikTunnel.SetText(v.AuvikTunnelPath)
	p.auvikAutoTun.SetChecked(v.AuvikAutoTunnel)
	p.auvikPrune.SetChecked(v.AuvikPruneStale)

	p.itglueKey.SetText(v.ITGlueAPIKey)
	p.itglueBase.SetText(v.ITGlueBaseURL)

	p.huduKey.SetText(v.HuduAPIKey)
	p.huduBase.SetText(v.HuduBaseURL)

	p.passKey.SetText(v.PassportalAPIKey)
	p.passTenant.SetText(v.PassportalTenant)
	p.passBase.SetText(v.PassportalBaseURL)

	p.cwCompany.SetText(v.CWCompanyID)
	p.cwPublic.SetText(v.CWPublicKey)
	p.cwPrivate.SetText(v.CWPrivateKey)
	p.cwClientID.SetText(v.CWClientID)
	p.cwBase.SetText(v.CWBaseURL)

	p.atUser.SetText(v.AutotaskUser)
	p.atSecret.SetText(v.AutotaskSecret)
	p.atCode.SetText(v.AutotaskCode)
	p.atBase.SetText(v.AutotaskBase)

	p.haloClientID.SetText(v.HaloClientID)
	p.haloClientSecret.SetText(v.HaloClientSecret)
	p.haloTenant.SetText(v.HaloTenant)
	p.haloBase.SetText(v.HaloBaseURL)

	p.domotzKey.SetText(v.DomotzAPIKey)
	p.domotzBase.SetText(v.DomotzBaseURL)

	p.ninjaClientID.SetText(v.NinjaClientID)
	p.ninjaClientSecret.SetText(v.NinjaClientSecret)
	p.ninjaBase.SetText(v.NinjaBaseURL)

	p.dattoKey.SetText(v.DattoAPIKey)
	p.dattoSecret.SetText(v.DattoSecret)
	p.dattoBase.SetText(v.DattoBaseURL)

	p.automateUser.SetText(v.AutomateUser)
	p.automatePass.SetText(v.AutomatePass)
	p.automateServer.SetText(v.AutomateServer)

	p.ncentralJWT.SetText(v.NcentralJWT)
	p.ncentralServer.SetText(v.NcentralServer)

	p.pdKey.SetText(v.PagerDutyAPIKey)
	p.pdBase.SetText(v.PagerDutyBaseURL)

	p.ogKey.SetText(v.OpsgenieAPIKey)
	p.ogBase.SetText(v.OpsgenieBaseURL)
}

func (p *MSPIntegrationPanel) fields(base SettingsFields) SettingsFields {
	base.MSPDefUsername = p.defUser.Text
	base.MSPDefCred = p.defCred.Text

	base.AuvikUsername = p.auvikUser.Text
	base.AuvikAPIKey = p.auvikKey.Text
	base.AuvikBaseURL = p.auvikBase.Text
	base.AuvikSyncEnabled = p.auvikSync.Checked
	base.AuvikSyncInterval = p.auvikInterval.Text
	base.AuvikTunnelPath = p.auvikTunnel.Text
	base.AuvikAutoTunnel = p.auvikAutoTun.Checked
	base.AuvikPruneStale = p.auvikPrune.Checked

	base.ITGlueAPIKey = p.itglueKey.Text
	base.ITGlueBaseURL = p.itglueBase.Text

	base.HuduAPIKey = p.huduKey.Text
	base.HuduBaseURL = p.huduBase.Text

	base.PassportalAPIKey = p.passKey.Text
	base.PassportalTenant = p.passTenant.Text
	base.PassportalBaseURL = p.passBase.Text

	base.CWCompanyID = p.cwCompany.Text
	base.CWPublicKey = p.cwPublic.Text
	base.CWPrivateKey = p.cwPrivate.Text
	base.CWClientID = p.cwClientID.Text
	base.CWBaseURL = p.cwBase.Text

	base.AutotaskUser = p.atUser.Text
	base.AutotaskSecret = p.atSecret.Text
	base.AutotaskCode = p.atCode.Text
	base.AutotaskBase = p.atBase.Text

	base.HaloClientID = p.haloClientID.Text
	base.HaloClientSecret = p.haloClientSecret.Text
	base.HaloTenant = p.haloTenant.Text
	base.HaloBaseURL = p.haloBase.Text

	base.DomotzAPIKey = p.domotzKey.Text
	base.DomotzBaseURL = p.domotzBase.Text

	base.NinjaClientID = p.ninjaClientID.Text
	base.NinjaClientSecret = p.ninjaClientSecret.Text
	base.NinjaBaseURL = p.ninjaBase.Text

	base.DattoAPIKey = p.dattoKey.Text
	base.DattoSecret = p.dattoSecret.Text
	base.DattoBaseURL = p.dattoBase.Text

	base.AutomateUser = p.automateUser.Text
	base.AutomatePass = p.automatePass.Text
	base.AutomateServer = p.automateServer.Text

	base.NcentralJWT = p.ncentralJWT.Text
	base.NcentralServer = p.ncentralServer.Text

	base.PagerDutyAPIKey = p.pdKey.Text
	base.PagerDutyBaseURL = p.pdBase.Text

	base.OpsgenieAPIKey = p.ogKey.Text
	base.OpsgenieBaseURL = p.ogBase.Text

	return base
}

func (p *MSPIntegrationPanel) content() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabelWithStyle("MSP integration stack", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Three layers work together: PSA customers → inventory IPs → vault credentials.\n"+
			"Use File tab actions to sync after saving credentials here or in pfsetup-apis."),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Shared inventory defaults", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form(
			row("Default SSH user", p.defUser),
			row("Default vault credential", p.defCred),
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Inventory (IPs + new devices)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Sync creates SSH sessions under Customers/<client>/<source>/ and updates IPs on re-sync."),
		widget.NewLabelWithStyle("Auvik", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form(
			row("Username", p.auvikUser),
			row("API key", p.auvikKey),
			row("API base URL", p.auvikBase),
			row("Periodic sync", p.auvikSync),
			row("Sync every (min)", p.auvikInterval),
			row("AuvikTunnel path", p.auvikTunnel),
			row("Auto tunnel", p.auvikAutoTun),
			row("Prune stale on sync", p.auvikPrune),
		),
		widget.NewLabelWithStyle("Domotz", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form(row("API key", p.domotzKey), row("Base URL", p.domotzBase)),
		widget.NewLabelWithStyle("NinjaOne", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form(row("Client ID", p.ninjaClientID), row("Client secret", p.ninjaClientSecret), row("Base URL", p.ninjaBase)),
		widget.NewLabelWithStyle("Datto RMM", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form(row("API key", p.dattoKey), row("Secret", p.dattoSecret), row("Base URL", p.dattoBase)),
		widget.NewLabelWithStyle("ConnectWise Automate", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form(row("Username", p.automateUser), row("Password", p.automatePass), row("Server URL", p.automateServer)),
		widget.NewLabelWithStyle("N-able N-central", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form(row("JWT", p.ncentralJWT), row("Server URL", p.ncentralServer)),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Credentials (vault)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Import username/password into encrypted vault; link to sessions by host/name."),
		widget.NewLabelWithStyle("IT Glue", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form(row("API key", p.itglueKey), row("Base URL", p.itglueBase)),
		widget.NewLabelWithStyle("Hudu", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form(row("API key", p.huduKey), row("Base URL", p.huduBase)),
		widget.NewLabelWithStyle("Passportal", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form(row("API key", p.passKey), row("Tenant", p.passTenant), row("Base URL", p.passBase)),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("PSA customers", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Creates Customers/<name>/ folders. Use File tab to sync lists."),
		widget.NewLabelWithStyle("ConnectWise Manage", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form(row("Company ID", p.cwCompany), row("Public key", p.cwPublic), row("Private key", p.cwPrivate), row("Client ID", p.cwClientID), row("Base URL", p.cwBase)),
		widget.NewLabelWithStyle("Datto Autotask", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form(row("User", p.atUser), row("Secret", p.atSecret), row("Integration code", p.atCode), row("Base URL", p.atBase)),
		widget.NewLabelWithStyle("Halo PSA", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form(row("Client ID", p.haloClientID), row("Client secret", p.haloClientSecret), row("Tenant", p.haloTenant), row("Base URL", p.haloBase)),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Incident documentation (augment)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Post engineer work notes to PagerDuty. PSA/RMM/incident workflow stays in those apps."),
		widget.NewLabelWithStyle("PagerDuty", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form(row("API key", p.pdKey), row("Base URL", p.pdBase)),
		widget.NewLabelWithStyle("Opsgenie", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form(row("API key", p.ogKey), row("Base URL", p.ogBase)),
	)
}

func mspFileTabRows(actions *MSPIntegrationActions) []fyne.CanvasObject {
	if actions == nil {
		return nil
	}
	var rows []fyne.CanvasObject
	rows = append(rows,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("MSP sync", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Inventory adds SSH targets and updates IPs. Credentials fill the vault.\n"+
			"Customers folders come from PSA sync or file import."),
	)
	if actions.OnSyncCustomers != nil {
		rows = append(rows, widget.NewButton("Import customer list (JSON file)…", actions.OnSyncCustomers))
	}
	if actions.OnSyncConnectWise != nil {
		rows = append(rows, widget.NewButton("Sync customers from ConnectWise…", actions.OnSyncConnectWise))
	}
	if actions.OnSyncAutotask != nil {
		rows = append(rows, widget.NewButton("Sync customers from Autotask…", actions.OnSyncAutotask))
	}
	if actions.OnSyncHalo != nil {
		rows = append(rows, widget.NewButton("Sync customers from Halo…", actions.OnSyncHalo))
	}
	if actions.OnImportAuvik != nil {
		rows = append(rows, widget.NewButton("Sync devices from Auvik…", actions.OnImportAuvik))
	}
	if actions.OnSyncAuvik != nil {
		rows = append(rows, widget.NewButton("Sync all Auvik tenants now…", actions.OnSyncAuvik))
	}
	if actions.OnImportDomotz != nil {
		rows = append(rows, widget.NewButton("Sync devices from Domotz…", actions.OnImportDomotz))
	}
	if actions.OnImportNinja != nil {
		rows = append(rows, widget.NewButton("Sync devices from NinjaOne…", actions.OnImportNinja))
	}
	if actions.OnImportDatto != nil {
		rows = append(rows, widget.NewButton("Sync devices from Datto RMM…", actions.OnImportDatto))
	}
	if actions.OnImportAutomate != nil {
		rows = append(rows, widget.NewButton("Sync devices from Automate…", actions.OnImportAutomate))
	}
	if actions.OnImportNcentral != nil {
		rows = append(rows, widget.NewButton("Sync devices from N-central…", actions.OnImportNcentral))
	}
	if actions.OnImportITGlue != nil {
		rows = append(rows, widget.NewButton("Import credentials from IT Glue…", actions.OnImportITGlue))
	}
	if actions.OnImportHudu != nil {
		rows = append(rows, widget.NewButton("Import credentials from Hudu…", actions.OnImportHudu))
	}
	if actions.OnImportPassportal != nil {
		rows = append(rows, widget.NewButton("Import credentials from Passportal…", actions.OnImportPassportal))
	}
	rows = append(rows,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Engineer work documentation", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Document on-call/incident work from the engineer console (augments PagerDuty)."),
	)
	if actions.OnBindWorkContext != nil {
		rows = append(rows, widget.NewButton("Bind active incident…", actions.OnBindWorkContext))
	}
	if actions.OnClearWorkContext != nil {
		rows = append(rows, widget.NewButton("Clear active incident", actions.OnClearWorkContext))
	}
	if actions.OnDocumentWork != nil {
		rows = append(rows, widget.NewButton("Document work to incident…", actions.OnDocumentWork))
	}
	if actions.OnExportHandoff != nil {
		rows = append(rows, widget.NewButton("Export customer handoff package…", actions.OnExportHandoff))
	}
	if actions.OnOpenNOCMap != nil {
		rows = append(rows, widget.NewButton("Open NOC map view…", actions.OnOpenNOCMap))
	}
	return rows
}
