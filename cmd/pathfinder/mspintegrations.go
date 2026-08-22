package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2/dialog"

	"github.com/scottpeterman/pathfinderssh/internal/automate"
	"github.com/scottpeterman/pathfinderssh/internal/autotask"
	"github.com/scottpeterman/pathfinderssh/internal/connectwise"
	"github.com/scottpeterman/pathfinderssh/internal/docvault"
	"github.com/scottpeterman/pathfinderssh/internal/dattormm"
	"github.com/scottpeterman/pathfinderssh/internal/domotz"
	"github.com/scottpeterman/pathfinderssh/internal/halo"
	"github.com/scottpeterman/pathfinderssh/internal/hudu"
	"github.com/scottpeterman/pathfinderssh/internal/invsync"
	"github.com/scottpeterman/pathfinderssh/internal/mspsync"
	"github.com/scottpeterman/pathfinderssh/internal/ninja"
	"github.com/scottpeterman/pathfinderssh/internal/ncentral"
	"github.com/scottpeterman/pathfinderssh/internal/passportal"
	"github.com/scottpeterman/pathfinderssh/internal/product"
	"github.com/scottpeterman/pathfinderssh/internal/psasync"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
)

func (h *host) mspIntegrationsEnabled() bool {
	return h.mspEnrollment.Provider.RequiresCloudLogin()
}

func (h *host) mspInventoryDefaults() (user, cred string) {
	user = strings.TrimSpace(h.base.MSPInventoryDefUsername)
	if user == "" {
		user = strings.TrimSpace(h.base.AuvikDefaultUsername)
	}
	cred = strings.TrimSpace(h.base.MSPInventoryDefCredential)
	if cred == "" {
		cred = strings.TrimSpace(h.base.AuvikDefaultCredential)
	}
	return user, cred
}

func (h *host) mspIntegrationActions() *ui.MSPIntegrationActions {
	if !h.mspIntegrationsEnabled() {
		return nil
	}
	return &ui.MSPIntegrationActions{
		OnImportAuvik:      h.importFromAuvik,
		OnSyncAuvik:        h.syncAuvikNow,
		OnImportITGlue:     h.importFromITGlue,
		OnImportHudu:       h.importFromHudu,
		OnImportPassportal: h.importFromPassportal,
		OnSyncConnectWise:  h.syncConnectWiseCustomers,
		OnSyncAutotask:     h.syncAutotaskCustomers,
		OnSyncHalo:         h.syncHaloCustomers,
		OnSyncCustomers:    h.syncPSACustomers,
		OnImportDomotz:     h.importFromDomotz,
		OnImportNinja:      h.importFromNinja,
		OnImportDatto:      h.importFromDattoRMM,
		OnImportAutomate:   h.importFromAutomate,
		OnImportNcentral:   h.importFromNcentral,
		OnBindWorkContext:  h.bindWorkContext,
		OnClearWorkContext: h.clearWorkContext,
		OnDocumentWork:     h.documentWorkToIncident,
	}
}

func (h *host) mspCustomerNames() []string {
	return h.tree.Tree().ListCustomers(product.CustomersRoot)
}

func (h *host) resolveMSPCustomer(external string) string {
	return mspsync.ResolveCustomerName(h.mspCustomerNames(), external)
}

func (h *host) syncInventoryDevices(customer, folder, source string, devices []invsync.Device) (string, error) {
	customer = h.resolveMSPCustomer(customer)
	user, cred := h.mspInventoryDefaults()
	tr := h.tree.Tree()
	res := invsync.SyncCustomerTree(&tr, devices, invsync.Options{
		CustomerName:   customer,
		ImportFolder:   folder,
		IntegrationSrc: source,
		DefaultUser:    user,
		DefaultCred:    cred,
	})
	h.tree.SetTree(tr)
	h.saveTree(tr)
	msg := res.Summary()
	if len(res.Errors) > 0 {
		msg += "\nErrors: " + strings.Join(res.Errors, "; ")
	}
	return msg, nil
}

func (h *host) syncPSAFromSource(src psasync.Source) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	list, err := src.ListCustomers(ctx)
	if err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	tr := h.tree.Tree()
	root := product.CustomersRoot
	res := psasync.SyncFolderNames(root, list, func(name string) (string, error) {
		path, err := (&tr).CreateCustomer(root, name)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") {
				return "", nil
			}
			return "", err
		}
		return path, nil
	})
	res.Source = src.Name()
	h.tree.SetTree(tr)
	h.saveTree(tr)
	msg := fmt.Sprintf("Source: %s\nCreated: %d\nAlready present: %d", res.Source, len(res.Created), len(res.Existing))
	if len(res.Errors) > 0 {
		msg += "\nErrors: " + strings.Join(res.Errors, "; ")
	}
	dialog.ShowInformation("Customer sync", msg, h.win)
}

func (h *host) syncConnectWiseCustomers() {
	cli := connectwise.New(
		h.base.ConnectWiseCompanyID,
		h.base.ConnectWisePublicKey,
		h.base.ConnectWisePrivateKey,
		h.base.ConnectWiseClientID,
		h.base.ConnectWiseBaseURL,
	)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := cli.Verify(ctx); err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	h.syncPSAFromSource(connectwise.PSA{Client: cli})
}

func (h *host) syncAutotaskCustomers() {
	cli := autotask.New(
		h.base.AutotaskUsername,
		h.base.AutotaskSecret,
		h.base.AutotaskAPIIntegrationCode,
		h.base.AutotaskBaseURL,
	)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := cli.Verify(ctx); err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	h.syncPSAFromSource(autotask.PSA{Client: cli})
}

func (h *host) syncHaloCustomers() {
	cli := halo.New(
		h.base.HaloClientID,
		h.base.HaloClientSecret,
		h.base.HaloTenant,
		h.base.HaloBaseURL,
	)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := cli.Verify(ctx); err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	h.syncPSAFromSource(halo.PSA{Client: cli})
}

func (h *host) importFromDomotz() {
	cli := domotz.New(h.base.DomotzAPIKey, h.base.DomotzBaseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := cli.Verify(ctx); err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	ui.ShowInventorySyncDialog(h.win, ui.InventorySyncOptions{
		Title:          "Domotz",
		Help:           "Sync accepted Domotz devices into Customers/<client>/Domotz/. IPs update on re-sync.",
		CustomerNames:  h.mspCustomerNames(),
		OnSync: func(customer string) (string, error) {
			ctx2, cancel2 := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel2()
			devs, err := cli.ListDevices(ctx2)
			if err != nil {
				return "", err
			}
			return h.syncInventoryDevices(customer, mspsync.FolderDomotz, invsync.SourceDomotz, devs)
		},
	})
}

func (h *host) importFromNinja() {
	cli := ninja.New(h.base.NinjaClientID, h.base.NinjaClientSecret, h.base.NinjaBaseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := cli.Verify(ctx); err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	ui.ShowInventorySyncDialog(h.win, ui.InventorySyncOptions{
		Title: "NinjaOne",
		Help:  "Sync NinjaOne devices into Customers/<client>/Ninja/.",
		CustomerNames: h.mspCustomerNames(),
		OnSync: func(customer string) (string, error) {
			ctx2, cancel2 := context.WithTimeout(context.Background(), 180*time.Second)
			defer cancel2()
			devs, err := cli.ListDevices(ctx2, "")
			if err != nil {
				return "", err
			}
			return h.syncInventoryDevices(customer, mspsync.FolderNinja, invsync.SourceNinja, devs)
		},
	})
}

func (h *host) importFromDattoRMM() {
	cli := dattormm.New(h.base.DattoAPIKey, h.base.DattoSecret, h.base.DattoBaseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := cli.Verify(ctx); err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	ui.ShowInventorySyncDialog(h.win, ui.InventorySyncOptions{
		Title: "Datto RMM",
		Help:  "Sync Datto RMM devices into Customers/<client>/DattoRMM/.",
		CustomerNames: h.mspCustomerNames(),
		OnSync: func(customer string) (string, error) {
			ctx2, cancel2 := context.WithTimeout(context.Background(), 180*time.Second)
			defer cancel2()
			devs, err := cli.ListDevices(ctx2, "")
			if err != nil {
				return "", err
			}
			return h.syncInventoryDevices(customer, mspsync.FolderDattoRMM, invsync.SourceDattoRMM, devs)
		},
	})
}

func (h *host) importFromAutomate() {
	cli := automate.New(h.base.AutomateUsername, h.base.AutomatePassword, h.base.AutomateServerURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := cli.Verify(ctx); err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	ui.ShowInventorySyncDialog(h.win, ui.InventorySyncOptions{
		Title:         "ConnectWise Automate",
		Help:          "Sync Automate computers into Customers/<client>/Automate/.",
		CustomerNames: h.mspCustomerNames(),
		OnSync: func(customer string) (string, error) {
			ctx2, cancel2 := context.WithTimeout(context.Background(), 180*time.Second)
			defer cancel2()
			devs, err := cli.ListDevices(ctx2)
			if err != nil {
				return "", err
			}
			return h.syncInventoryDevices(customer, mspsync.FolderAutomate, invsync.SourceAutomate, devs)
		},
	})
}

func (h *host) importFromNcentral() {
	cli := ncentral.New(h.base.NcentralJWT, h.base.NcentralServerURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := cli.Verify(ctx); err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	ui.ShowInventorySyncDialog(h.win, ui.InventorySyncOptions{
		Title:         "N-able N-central",
		Help:          "Sync N-central devices into Customers/<client>/N-central/.",
		CustomerNames: h.mspCustomerNames(),
		OnSync: func(customer string) (string, error) {
			ctx2, cancel2 := context.WithTimeout(context.Background(), 180*time.Second)
			defer cancel2()
			devs, err := cli.ListDevices(ctx2)
			if err != nil {
				return "", err
			}
			return h.syncInventoryDevices(customer, mspsync.FolderNcentral, invsync.SourceNcentral, devs)
		},
	})
}

func (h *host) importFromHudu() {
	if h.vault == nil {
		dialog.ShowInformation("Hudu", "Unlock or create the credential vault first.", h.win)
		return
	}
	cli := hudu.New(h.base.HuduAPIKey, h.base.HuduBaseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := cli.Verify(ctx); err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	companies, err := cli.ListCompanies(ctx)
	if err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	labels := make([]string, len(companies))
	idByLabel := map[string]string{}
	for i, c := range companies {
		labels[i] = c.Name
		idByLabel[c.Name] = c.ID
	}
	ui.ShowDocVaultImportDialog(h.win, ui.DocVaultImportOptions{
		Title:         "Hudu",
		Organizations: labels,
		CustomerNames: h.mspCustomerNames(),
		OnImport: func(label, customerFolder string, link bool) (string, error) {
			id := idByLabel[label]
			ctx2, cancel2 := context.WithTimeout(context.Background(), 180*time.Second)
			defer cancel2()
			raw, err := cli.ListPasswords(ctx2, id)
			if err != nil {
				return "", err
			}
			for i := range raw {
				raw[i].OrganizationName = label
			}
			vst, err := docvault.SyncPasswordsToVault(h.vault, raw, docvault.VaultSyncOptions{
				SourceTag:      mspsync.TagHudu,
				IDTagPrefix:    mspsync.TagHuduID,
				UpdateExisting: true,
			})
			if err != nil {
				return "", err
			}
			msg := fmt.Sprintf("vault: added %d, updated %d, skipped %d", vst.Added, vst.Updated, vst.Skipped)
			if link {
				customer := h.resolveMSPCustomer(customerFolder)
				if customer == "" {
					customer = h.resolveMSPCustomer(label)
				}
				credNames, err := docvault.CredNamesFromVault(h.vault, mspsync.TagHuduID)
				if err != nil {
					return msg, err
				}
				tr := h.tree.Tree()
				linkSt := docvault.LinkSessions(&tr, raw, credNames, docvault.LinkOptions{
					CustomerName: customer,
					OnlyEmpty:    true,
				})
				h.tree.SetTree(tr)
				h.saveTree(tr)
				msg += "; linked " + fmt.Sprintf("%d", linkSt.Linked) + " sessions"
			}
			h.refreshVault()
			return msg, nil
		},
	})
}

func (h *host) importFromPassportal() {
	if h.vault == nil {
		dialog.ShowInformation("Passportal", "Unlock or create the credential vault first.", h.win)
		return
	}
	cli := passportal.New(h.base.PassportalAPIKey, h.base.PassportalTenant, h.base.PassportalBaseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := cli.Verify(ctx); err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	ui.ShowPassportalImportDialog(h.win, ui.PassportalImportOptions{
		CustomerNames: h.mspCustomerNames(),
		OnImport: func(customer string, link bool) (string, error) {
			customer = h.resolveMSPCustomer(customer)
			ctx2, cancel2 := context.WithTimeout(context.Background(), 180*time.Second)
			defer cancel2()
			raw, err := cli.ListPasswords(ctx2)
			if err != nil {
				return "", err
			}
			for i := range raw {
				raw[i].OrganizationName = customer
			}
			vst, err := docvault.SyncPasswordsToVault(h.vault, raw, docvault.VaultSyncOptions{
				SourceTag:        mspsync.TagPassportal,
				IDTagPrefix:      mspsync.TagPassportalID,
				UpdateExisting:   true,
			})
			if err != nil {
				return "", err
			}
			msg := fmt.Sprintf("vault: added %d, updated %d, skipped %d", vst.Added, vst.Updated, vst.Skipped)
			if link && strings.TrimSpace(customer) != "" {
				credNames, err := docvault.CredNamesFromVault(h.vault, mspsync.TagPassportalID)
				if err != nil {
					return msg, err
				}
				tr := h.tree.Tree()
				linkSt := docvault.LinkSessions(&tr, raw, credNames, docvault.LinkOptions{
					CustomerName: customer,
					OnlyEmpty:    true,
				})
				h.tree.SetTree(tr)
				h.saveTree(tr)
				msg += "; linked " + fmt.Sprintf("%d", linkSt.Linked) + " sessions"
			}
			h.refreshVault()
			return msg, nil
		},
	})
}
