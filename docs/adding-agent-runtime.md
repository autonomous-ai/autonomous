# Adding an Agentic Runtime Backend

How to add a new agent "brain" (like OpenClaw, Hermes, PicoClaw) to the OS, and
**everything it must wire up** so it reaches parity. Written from the lessons of
the Hermes port, where too many pieces were left as silent no-ops and bit us
later (config that didn't survive a factory reset, skills that never came back,
identity rename that did nothing, no skill auto-update).

> **The one rule to remember:** the OS (`os-server`) is the platform; the backend
> is a swappable brain. Anything user-visible that OpenClaw does, a new backend
> must either do too or **consciously decide to skip — and say why in a comment.**
> A no-op is a decision, never a default.

Source of truth for the contract: `os/services/domain/agent.go` (the
`AgentGateway` interface). This doc explains *which* parts matter and *how* to
wire the switch, install, migration, skills, hooks, and reset.

---

## 0. Mental model

- `config.agent_runtime` (`/root/config/config.json`) selects the active backend.
- `internal/agent/factory.go` `ProvideGateway` resolves it at boot via Wire DI:
  `config.agent_runtime` > DEVICE.md `gateway.default` > openclaw.
- Switching at runtime goes through one core — `device.Service.UpdateAgentRuntime`
  — fired by 3 triggers (MQTT `agent_runtime.set`, HTTP `/api/device/agent-runtime`,
  web Runtime section). See `docs/hermes.md` §10–§11.

---

## 1. The contract — implement `domain.AgentGateway`

Your backend lives in `internal/<name>/` and its `*Service` must satisfy the
**whole** `AgentGateway` interface. The methods fall into groups:

| Group | Examples | New-backend stance |
|-------|----------|--------------------|
| **Core turn** | `SendChatMessage`, `SendSystemChatMessage`, `*WithImage`, `NextChatRunID`, `*WithRun`, `StartWS` | **MUST** — this is the agent. |
| **Readiness / busy** | `IsReady`, `ConnectedAt`, `AgentUptime`, `IsBusy`, `SetBusy`, `QueuePendingEvent` | **MUST** — os-server gates sensing on these. |
| **Identity** | `Name`, `Version`, `GetSessionKey`/`SetSessionKey` | **MUST** — surfaces in web Status. |
| **HAL passthrough** | `SendToHALTTS*`, `StopTTS`, `SetVolume`, `StartHALVoice` | Usually identical across backends — share or copy. |
| **Run markers** | `MarkGuardRun`/`Consume*`, `MarkBroadcastRun`, `MarkPoseBucketRun`, `MarkWebChatRun`, `MarkSilentRun`, `*PendingChatTrace*` | **MUST track** — os-server tags turns by runID; these are runtime-shaped but the OS depends on them. |
| **Channels** | `AddChannel`, `RefreshChannelConfig`, `PairWhatsapp`, `HasWhatsappSession`, `GetTelegram*`, `Broadcast`, `SendToUser*` | No-op only if the backend genuinely can't (e.g. WhatsApp needs a Baileys plugin). Telegram usually still works via Lumi config. |
| **Lifecycle / onboarding** | `SetupAgent`, `EnsureOnboarding`, `ResetAgent`, `RestartAgent`, `RefreshModelsConfig` | Decide per backend; document the no-op. |
| **Migration-adjacent** | `UpdateIdentityName`, `StartSkillWatcher`, `WatchIdentity`, `StartModelSync`, `UpdatePrimaryModel`, `StartPrimaryModelWatch`, `CompactSession`, `NewSession`, `FetchChatHistory`, `WriteMCPEntry`/`RemoveMCPEntry`, `GetConfigJSON` | **The danger zone** — easy to no-op, expensive to discover missing. See §4–§6. |

**Lesson (Hermes):** ~15 of these were stubbed no-op in `internal/hermes/stubs.go`.
Some are legitimately N/A (`WriteMCPEntry` — no `openclaw.json`; `PairWhatsapp` —
no plugin). But `StartSkillWatcher`, `UpdateIdentityName`, and the config-sync path
were **functional gaps**, not N/A — they shipped as no-ops and we only noticed when
skills went stale / rename did nothing / config broke after a reset. Audit every
stub: write `// no-op because <reason>` or `// TODO(<backend>-<feature>)`, never a
bare empty body.

---

## 2. Register + wire the switch

1. `domain/device.go`: add `AgentRuntime<Name>` const + the entry in `AgentRuntimes`.
2. `internal/agent/factory.go`: add the `case` in `ProvideGateway`.
3. **Embedded installer**: `internal/<name>/install.sh` + `install.go`
   (`//go:embed install.sh` → `runtimereg.Register(name, InstallScript)`).
4. `switch_runtime.sh` is **generic** — it knows no backend names. Do **not** edit
   it, the imager, or os-server's switch core to add a backend.

The installer contract (`switch_runtime.sh` expects):
- create a systemd unit; declare its name in `/usr/local/lib/os-runtimes/<name>/service`
  if it isn't `<name>.service`.
- optionally drop a `verify` hook at `/usr/local/lib/os-runtimes/<name>/verify`
  (exit 0 = "installed & usable"). Keep it **cheap** — see §3 for why it must not
  over-check.

---

## 3. The golden rule: install-once vs every-switch (the *activation gap*)

`install.sh` runs **once** — `switch_runtime.sh` only runs it on a first install or
when `verify` fails. On every later switch it is **skipped**.

> **Therefore: anything that must survive a factory reset, or must refresh on a
> plain `os-server` OTA, MUST NOT be written only by `install.sh`. It must be
> materialized by `os-server` on every switch.**

This is the **activation gap** and we hit it twice on Hermes:
- A fix shipped inside `install.sh` (or any file `install.sh` writes — the
  `verify` hook, the presync hook) **never reaches an already-installed device**
  via OTA: the old on-disk copy keeps passing `verify`, so `install.sh` never
  re-runs, so the new copy never lands.

The fix pattern (use it for everything stateful):
- Put the logic in a **presync hook** (`runtime-<name>-presync`).
- Embed it: `//go:embed presync.sh` → `runtimereg.RegisterPresync(name, PresyncScript)`.
- `os-server` materializes it every switch (`internal/device/runtime_installers.go`
  `materializePresync`, called from the switch flow next to `materializeInstaller`).
- `switch_runtime.sh` runs `runtime-<name>-presync` right before the backend starts.

Hermes's presync (`internal/hermes/presync.sh`) now owns **both** the
`config.yaml` model wiring (idempotent — coerces a reset-blanked `model: ''` back
to a map, asserts `provider`/`custom_providers` structure, syncs `llm_*`/secrets)
**and** the skill restore (re-runs `claw migrate` when `skills/openclaw-imports`
is empty). Keep `verify` CLI-only (`command -v <bin>`) — a structure-check in
`verify` would force a heavy full reinstall when presync alone heals it.

---

## 4. Persona + memory migration (Go, runs every switch)

Add `internal/agent/migrate_persona/openclaw_to_<name>.go` and the reverse. It runs
at os-server boot after a real switch (`Reconcile`, when `agent_state.json`
prev ≠ current). What to carry:

- **SOUL.md** → backend's identity file. If the backend has **no separate
  IDENTITY.md slot** (Hermes doesn't), inline the owner's filled IDENTITY fields
  as a `## Your identity card` block in SOUL (see `buildIdentityBlock`).
- **MEMORY.md + `memory/*.md` daily + KNOWLEDGE.md** → merged into the backend's
  long-term memory file. **First check which files the backend LOADS BY NAME** —
  Hermes loads only `MEMORY.md` + `USER.md` (no `memories/*.md` glob), so a
  separate `KNOWLEDGE.md` would be ignored; we fold it into `MEMORY.md` instead.
- **USER.md** → backend's user-profile file.
- Set **`Overwrite = true`** for the soul copy on a switch: a switch means "adopt
  the persona I was just using." `copyPersona` backs up first (`.bak-<nano>`).
- The reverse direction must **strip backend-only artifacts** it added (e.g. the
  identity card — OpenClaw keeps the name in its own IDENTITY.md).

Do **NOT** carry runtime-specific files: `AGENTS.md`, `TOOLS.md`, `HEARTBEAT.md`,
`hooks/` — they belong to the source runtime. The backend's **deep memory engine**
(episodic/semantic DB, dream-diary, grounded-short-term) is **not portable** —
the distilled `MEMORY.md`/`USER.md` is the portable form.

---

## 5. Skills

- Skills reach the backend by being **copied** (verify: copy vs convert! Hermes's
  `claw migrate` is `shutil.copytree`, no transform) into the backend's skill dir.
- **Restore-after-reset** belongs in **presync**, guarded on the dir being empty
  (so a normal switch is a no-op — no churn). See §3.
- **Skill watcher** (auto-update from CDN, capability-gated): the generic
  fetch/extract/hash plumbing is shared in `internal/skills/skillzip.go`
  (`FetchSkillVersions`/`DownloadToTempFile`/`FolderHash`/`ExtractSkillZip`). Add a
  thin `internal/<name>/skill_watcher.go` parallel to `internal/openclaw/skill_watcher.go`
  — only the **target dir** and the **notify path** differ. Gate with
  `skills.Supported(device.Capabilities(...))`. Notify the agent with
  `SendSystemChatMessage`.

---

## 6. Hooks — STILL OPEN for Hermes (a worked example of a gap)

OpenClaw hooks (`hooks/<name>/{HOOK.md, handler.ts}`) are TypeScript handlers that
fire on OpenClaw's `message:preprocessed` event — `emotion-acknowledge` (instant
"thinking" face on message arrival) and `turn-gate` (set busy for channel turns).
They are **runtime-specific** and not portable.

A new backend does **not** inherit them. Options:
- **Backend-native hooks** — if the backend has a hook system (Hermes has Python
  plugin hooks on `pre_gateway_dispatch` / gateway `agent:start`, discovered from
  `~/.hermes/plugins`; there is **no** drop-in `~/.hermes/hooks/HOOK.yaml` loader
  in the shipped build — verify your backend's actual loader before assuming).
- **OS-side handling** — for a voice device the highest-value behavior (thinking
  face on turn start) is best done in `os-server` (Go), where it covers
  os-server-initiated voice/sensing turns regardless of backend. Channel-initiated
  turns (Telegram) still need a backend-native hook.

> Status: Hermes hooks are **not yet implemented**. This is the remaining parity
> gap after skills/config/identity/persona were closed.

---

## 7. Factory reset

- Add the backend wipe in `server/system/reset_<name>.go` (`wipe<Name>State`) and
  the `case` in `factoryreset.go`.
- **Wipe `/root/config/agent_state.json` in lockstep with `config.json`** — they
  are a pair (current runtime + switch history). Leaving `agent_state.json` while
  `config.json` resets makes a stale `prev` diverge from the reset `current` and
  triggers a **spurious persona migration** that propagates wiped/stub state.
- Keep what must survive (`bootstrap.json` = OTA state).
- A wipe path that removes migrated content (skills, config) must have a
  **restore path that runs after the reset** (presync, §3/§5) — or the content is
  gone for good once `install.sh` stops re-running.

---

## 8. Capability gating

Use the runtime-agnostic platform metadata in `internal/skills`:
- `skills.Supported(deviceCaps)` for skills, `skills.SupportedHooks(deviceCaps)`
  for hooks, where `deviceCaps = device.Capabilities(config.DeviceTypeOrDefault())`.
- Never hardcode a skill/hook list per backend — gate the same way OpenClaw does.

---

## Checklist for a new backend

- [ ] `internal/<name>/` package; `*Service` implements **all** of `AgentGateway`.
- [ ] Every stub is `// no-op because …` or `// TODO(<name>-…)` — no bare bodies.
- [ ] `domain.AgentRuntime<Name>` + `AgentRuntimes` entry; `factory.go` case.
- [ ] `install.sh` + `install.go` (`//go:embed` + `runtimereg.Register`).
- [ ] **Stateful setup → `presync.sh`** (`//go:embed` + `runtimereg.RegisterPresync`),
      materialized by os-server every switch. Nothing reset-fragile lives only in
      `install.sh`.
- [ ] `verify` hook is cheap (CLI presence), not a structure check.
- [ ] `migrate_persona/openclaw_to_<name>.go` + reverse: SOUL(+identity inline),
      MEMORY+daily+KNOWLEDGE (folded into a file the backend LOADS BY NAME), USER;
      `Overwrite=true`; reverse strips backend-only artifacts.
- [ ] Skills: copy-import + **restore-in-presync** (guarded) + `skill_watcher.go`
      (parallel to openclaw, shared `internal/skills/skillzip.go`).
- [ ] Hooks: backend-native or OS-side — decided & documented (not silently absent).
- [ ] `reset_<name>.go` + `factoryreset.go` case; **`agent_state.json` wiped with
      `config.json`**.
- [ ] Capability gating via `skills.Supported` / `SupportedHooks`.
- [ ] Notify the agent on skill change via `SendSystemChatMessage`.
- [ ] Docs: update `docs/hermes.md`-style backend doc + this checklist if the
      contract changed.

---

## Hermes parity status (honest ledger)

**Done:** switch wiring, embedded install + presync (config self-heal, skill
restore), persona/memory migration (SOUL + inline identity, MEMORY + daily +
KNOWLEDGE, USER), `UpdateIdentityName`, skill watcher, factory-reset
`agent_state.json` lockstep.

**Still open / no-op:** hooks (`emotion-acknowledge`, `turn-gate` — §6),
`WriteMCPEntry`/`RemoveMCPEntry` (`TODO(hermes-mcp)`), `CompactSession`,
`FetchChatHistory`, and the model-sync group (`StartModelSync`,
`UpdatePrimaryModel`, `StartPrimaryModelWatch`, `RefreshModelsConfig` — largely
N/A because os-server sends a fixed request model to the campaign-api custom
provider).
