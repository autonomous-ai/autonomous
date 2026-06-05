# Security

## Reporting

Email security reports to the maintainer (see `MAINTAINERS`). Do not open public
issues for vulnerabilities.

## Secret model

Autonomous separates secrets by **purpose** — they must never be conflated. A
device-control credential is not a model-provider credential.

| Secret | Purpose | Source |
|--------|---------|--------|
| `device_auth_token` | the token a caller presents to reach the on-device HAL | `config.json::device_auth_token` / `LELAMP_DEVICE_AUTH_TOKEN` |
| `llm_api_key` | the LLM/model provider key (Deepgram, OpenAI, …) | `config.json::llm_api_key` |
| internal hardware secret | runtime-internal HAL ↔ system-server calls | (planned) |

### Why they're split

Historically the LeLamp HAL used the **LLM provider key as its device auth
token** (`DL_API_KEY` mirrored `llm_api_key`). That coupled a model-billing
credential with device control: leaking one leaked the other, and rotating the
LLM key silently changed who could drive the hardware.

The HAL now authenticates the bearer token against **`device_auth_token`**, a
secret distinct from `llm_api_key`. For backward compatibility with devices
provisioned before the split, `device_auth_token` **falls back to `llm_api_key`
when unset** — so existing deployments keep working, but new provisioning should
set a distinct `device_auth_token`.

## Known follow-ups (tracked)

- **Go system server (`os/core`)** carries the same coupling and a legacy
  `?token=` query-string auth path (tokens leak via logs/referrers/history).
  Apply the same split there and remove query-token auth where possible.
- **Internal hardware secret**: introduce a separate runtime-internal secret for
  HAL ↔ system-server calls, distinct from both of the above.
