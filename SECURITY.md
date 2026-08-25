# Security policy — PathfinderSSH MSP

This repository is the **AgenticOps MSP fork** of [PathfinderSSH](https://github.com/scottpeterman/pathfinderssh) (Scott Peterman), licensed under **GPL-3.0**. Upstream PathfinderSSH is not an AgenticOps product; this edition is.

## Reporting a vulnerability

Please report security issues **privately first** — do not open a public GitHub issue for credential leaks, host-key bypasses, or anything that could put production gear at risk.

- Email: **opensource@agenticop.io**
- Subject line: `PathfinderSSH MSP security`

Include what you observed, how to reproduce it, and which binary/commit you used (`pathfinder.exe -version` / `git describe` if building from source). We will acknowledge receipt and work with you on a fix before any public disclosure.

For non-security bugs, use the normal issue tracker on https://github.com/AgenticOp-io/pathfinder-lane.

## Security stance (non-negotiable)

These match upstream PathfinderSSH’s documented perimeter. Automated crawl/capture paths must not relax them; MSP UI features (button bar, scripts, fan-out) are **operator-driven** and sit outside the automated read-only engine.

1. **Credential storage done right** — OS-backed vault, no plaintext password path in session files.
2. **Host-key verification fails closed** — a key mismatch is never overridden for convenience; an unanswered trust-on-first-use prompt resolves to *no*.
3. **Crawler and capture are provably read-only** — every automated command is on an exact-string allowlist, checked in code and tests.
4. **Never hang a device** — per-command timeouts, byte bounds, and limited concurrency for expensive commands.
5. **Map viewer is the only socket listener** — loopback only; per-run token, `Origin` / `Host` checks, opaque node IDs; click confirms, never auto-connects.
6. **Session files are not secrets** — they store vault *references*, never passwords or passphrases; imports drop any password fields.

## GPL and attribution

Distributed binaries must remain free software under GPL-3.0. Corresponding source is this MSP repository. Installed AppData copies include `LICENSE` and `NOTICE` beside `pathfinder.exe`. See `NOTICE` for upstream attribution.

## Out of scope for this policy

- Compromised OS accounts or keyrings
- Devices that already accept weak credentials
- Operator misuse of interactive send / scripts against production without change control
