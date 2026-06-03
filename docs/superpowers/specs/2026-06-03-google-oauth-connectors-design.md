# Google OAuth Connectors (Gmail / Google Calendar / Google Drive)

**Date:** 2026-06-03
**Status:** Approved — implementing
**Area:** lamp-server · MQTT connector flow

## Goal

Add three first-class connectors — `gmail`, `google_calendar`, `google_drive` — that
clone the **Google OAuth** credential model rather than the Notion/MCP model. They are
driven through the existing connector dispatch (`connector.set.<code>` /
`connector.remove.<code>`) and refreshed by the existing connector refresh loop, but
they do **not** write an `mcp.servers.<code>` entry into `openclaw.json`.

## Why not the Notion/MCP path

Investigation (and a context7 check of `/google/mcp` and
`/taylorwilsdon/google_workspace_mcp`) confirmed there is **no hosted per-service Google
MCP URL** for Gmail/Drive/Calendar (Google publishes hosted MCP only for Cloud services
like BigQuery/Maps; Workspace MCP is self-hosted and single-server). The existing MCP
connectors (`notion`, …) require a hardcoded `mcpURL` and skip registration without one.
So Google services cannot be wired as remote-MCP connectors today.

The existing **Google OAuth flow** already stores + refreshes an OAuth2 credential with
no MCP side-effect — that is the correct template. We reuse the connector *flow* (so
they appear as connectors with separate per-connector config) but the OAuth *auth shape*.

## Design

### Identity & storage
- Connector codes (must match what the backend publishes): `gmail`, `google_calendar`,
  `google_drive`.
- Each connector persists to its **own** config file under
  `<OpenclawConfigDir>/workspace/configs/`, following the connector convention used by
  MCP connectors (`notion_access_tokens.json`): `gmail_access_tokens.json`,
  `google_calendar_access_tokens.json`, `google_drive_access_tokens.json`.
- File schema is the existing `domain.ConnectorsFile` / `ConnectorEntry` — no new schema.
- The shared generic `connectors.json` and the OAuth `access_tokens.json` are untouched.
  The orphaned `google` provider in `access_tokens.json` is left as-is.

### New component: `oauthConnectorWriter`
A new `ConnectorWriter` implementation in
`lamp/server/device/delivery/mqtt/oauth_connector_writer.go`. It is the
`mcpConnectorWriter` with the `openclaw.json` side-effect removed:

- `Write(ctx, creds)` — persist the credential to `<code>_access_tokens.json`. No
  `gateway.WriteMCPEntry`.
- `Remove(ctx, connector)` — delete the entry from `<code>_access_tokens.json`. No
  `gateway.RemoveMCPEntry`.
- `RefreshableEntries()` — same rule as every writer: entry is eligible only with BOTH
  `refresh_token` and `refresh:true` (backend owns the flag).
- `loadEntry(connector)` — satisfies `entryLoader` so the refresh loop preserves
  fields the BE refresh response doesn't re-send (scopes, client_id, credentials).

It reuses the package-internal helpers `loadConnectorsFile` / `writeConnectorsFile` /
`connectorEntryFromCreds` / `credsFromEntry` — no duplication of file/atomic-write logic.

### Registration
In `handler.go`:
- Add `var googleOAuthConnectors = []string{"gmail", "google_calendar", "google_drive"}`.
- In `buildConnectorWriters`, after the MCP loop, register one `oauthConnectorWriter`
  per code: `reg[name] = newOAuthConnectorWriter(name, configsDir)`.

No changes to `dispatchData` — `connector.set.<code>` / `connector.remove.<code>` already
route by prefix, and `connectorWriters.get(code)` now returns the OAuth writer for these
three (instead of falling back to `default`).

### Refresh
These flow through the **existing** connector refresh loop
(`StartConnectorRefreshLoop` → `/connector/refresh-token`), unchanged. Because the new
writer implements `entryLoader`, rotation preserves the full entry. No new endpoint, no
new loop. (The Google-specific `/oauth/refresh` path and its `oauthRefreshableProviders`
map are not involved.)

### Data flow
```
backend → MQTT connector.set.gmail {auth_type:"oauth2", access_token, refresh_token,
          expires_in, scopes, refresh:true, ...}
        → handleConnectorSet → runConnectorSet
        → connectorWriters.get("gmail") = oauthConnectorWriter
        → writes gmail_access_tokens.json   (NO openclaw.json)
        → publishDataResult success

refresh loop (every 3m) → oauthConnectorWriter.RefreshableEntries()
        → POST /connector/refresh-token → re-Write gmail_access_tokens.json
```

## Out of scope
- No `openclaw.json` / MCP wiring (no Google MCP URLs exist).
- No backend changes (backend must publish these codes and support them in
  `/connector/refresh-token`; device side is endpoint-agnostic).
- No change to the existing `oauth.set`/`access_tokens.json` `google` provider.

## Testing
- Unit test `oauthConnectorWriter`: Write creates the per-connector file with the entry;
  Remove deletes it and returns removed=true/false correctly; RefreshableEntries gates on
  refresh+refresh_token; loadEntry round-trips.
- Verify registration: `buildConnectorWriters` returns the OAuth writer (not `default`)
  for the three codes, and each writer targets a distinct file.
- `go build ./...` and `go test ./server/device/delivery/mqtt/...`.

## Files
- New: `lamp/server/device/delivery/mqtt/oauth_connector_writer.go`
- New: `lamp/server/device/delivery/mqtt/oauth_connector_writer_test.go`
- Edit: `lamp/server/device/delivery/mqtt/handler.go` (spec list + registration)
- Docs: `docs/mqtt.md` + `docs/vi/mqtt_vi.md` (new "Google OAuth connectors" subsection)
