# PathfinderSSH MSP — documentation index

AgenticOps fork of [PathfinderSSH](https://github.com/scottpeterman/pathfinderssh) (Scott Peterman).  
Repo: https://github.com/AgenticOp-io/pathfinderssh-msp · License: **GPL-3.0**

## Start here

| Doc | Audience | Contents |
| --- | --- | --- |
| [**MSP-SYNOPSIS.md**](MSP-SYNOPSIS.md) | Everyone | What we built — executive summary |
| [**MSP-FEATURES.md**](MSP-FEATURES.md) | Engineers / MSP staff | Feature catalog with UI paths |
| [**INSTALL.md**](INSTALL.md) | Installers | GUI installer, cloud sign-in, paths |

## Integrations (cohesive MSP stack)

| Doc | Contents |
| --- | --- |
| [**MSP-INTEGRATION-STACK.md**](MSP-INTEGRATION-STACK.md) | **Start here** — customers + inventory + credentials + incident doc |
| [**MSP-INTEGRATION-COMBINATIONS.md**](MSP-INTEGRATION-COMBINATIONS.md) | Every PSA × RMM × vault combination |
| [**MSP-INTEGRATION-SETTINGS.md**](MSP-INTEGRATION-SETTINGS.md) | Settings → Tools field reference |
| [**MSP-FEATURE-GAPS.md**](MSP-FEATURE-GAPS.md) | Engineer augment gaps vs CRT / PSA replacement |
| [**INTEGRATIONS.md**](INTEGRATIONS.md) | Architecture overview |
| [**AUVIK-API.md**](AUVIK-API.md) | Auvik inventory sync, tunnel, settings |
| [**ITGLUE-API.md**](ITGLUE-API.md) | IT Glue password import, vault, session linking |
| [**DOMOTZ-API.md**](DOMOTZ-API.md) · [**NINJA-API.md**](NINJA-API.md) · [**DATTO-RMM-API.md**](DATTO-RMM-API.md) | Inventory alternatives |
| [**AUTOMATE-API.md**](AUTOMATE-API.md) · [**NCENTRAL-API.md**](NCENTRAL-API.md) | RMM inventory |
| [**HUDU-API.md**](HUDU-API.md) · [**PASSPORTAL-API.md**](PASSPORTAL-API.md) | Vault alternatives |
| [**CONNECTWISE-API.md**](CONNECTWISE-API.md) · [**AUTOTASK-API.md**](AUTOTASK-API.md) · [**HALO-API.md**](HALO-API.md) | PSA customers |
| [**PAGERDUTY-API.md**](PAGERDUTY-API.md) | Incident bind + engineer documentation |
| [**OPSGENIE-API.md**](OPSGENIE-API.md) | Opsgenie alert notes (same augment lane) |
| [**CURSOR-AI.md**](CURSOR-AI.md) | Troubleshoot addon, side pane, Cloud Agents API |

## Access & security

| Doc | Contents |
| --- | --- |
| [**AUTH.md**](AUTH.md) | Solo, Microsoft 365, Google sign-in + roadmap |
| [**../SECURITY.md**](../SECURITY.md) | Vulnerability reporting, security stance |

## Reference

| Doc | Contents |
| --- | --- |
| [**ARCHITECTURE.md**](ARCHITECTURE.md) | Packages, data flow, file layout |
| [**CLI.md**](CLI.md) | Binaries and command-line flags |
| [**OPERATIONS.md**](OPERATIONS.md) | Day-to-day MSP workflows |
| [**CONTRIBUTING.md**](CONTRIBUTING.md) | Build and branch workflow |

## Samples

| File | Purpose |
| --- | --- |
| [customer-crawl-seeds.example.csv](customer-crawl-seeds.example.csv) | CSV template for crawl seeds |

## UI images

Anonymized mockups (no real customer names): [images/README.md](images/README.md)

## Roadmap & upstream

| Doc | Contents |
| --- | --- |
| [../README.md](../README.md) | MSP product entry |
| [../ROADMAP-FRONTEND.md](../ROADMAP-FRONTEND.md) | MSP UI roadmap (done vs open) |
| [UPSTREAM.md](UPSTREAM.md) | Upstream engine attribution |

## Packaging (outside this repo)

| Path | Contents |
| --- | --- |
| `products/pathfinder-msp/` | `Install.ps1`, `Update-Install.ps1`, Windows binaries |
| `products/pathfinder-msp/deploy/entra/` | Entra app registration script (optional) |
