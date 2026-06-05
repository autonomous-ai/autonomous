# Designing Autonomous — an OS for physical AI agents

Autonomous is an open-source OS that thousands of physical AI devices — different sensors,
actuators, and boards — can be built on without forking. This document defines the
architecture and the rules that make that possible. It is grounded in two battle-tested
systems: **Linux** (one kernel, every piece of hardware) and **Android** (one OS, thousands
of OEM devices).

## Three load-bearing principles

1. **Organize by interface, not by vendor.** A driver lives in the *subsystem* whose
   capability it provides (`motion`, `vision`, `audio`), never in a vendor folder. Adding the
   300th device changes one leaf, not the core. *(Linux `drivers/`.)*
2. **Two stability policies, by audience.** The contract facing skills and devices is a
   **frozen ABI**; the driver API behind it is **deliberately unstable and in-tree**, so when
   the core changes, every driver is fixed in the same commit. Out-of-tree is your problem;
   in-tree is ours — the flywheel that maximizes hardware support. *(Linux uapi + in-tree
   drivers; Android Treble vendor interface.)*
3. **Devices are declarative overlays, not forks; conformance is a test, not a promise.** A
   new device is a `DEVICE.md` inheriting a base + any missing drivers. "Autonomous-compatible"
   is defined by a spec and enforced by a test suite. *(Android `device/`, CDD + CTS.)*

## The layered stack

`Skills → Agentic Runtime → System Services → HAL (Capabilities) → Drivers → Board → Linux → Safety`.
See [overview.md](overview.md). This document is about the *contracts between layers* and the
*folder structure* that expresses them.

## Target repository structure

Organized by subsystem (Linux) with declarative device overlays (Android):

```
contract/              the frozen ABI — what skills + devices build against
  capabilities.md      the capability vocabulary (the "feature flags")
  DEVICE-SPEC.md       the device-manifest format + versioning
  COMPATIBILITY.md     the CDD — what a compliant device MUST do
skills/                the apps (SKILL.md), by domain
os/
  services/            System Services (Go): intent, network, OTA, sensing routing
  runtime/             the agentic-runtime bridge (gateway client, skill loader)
  hal/
    contract/          the capability interface (the routes/IDL)
    drivers/           BY SUBSYSTEM: audio/ vision/ motion/ light/ display/ sensing/
    board/             BY BOARD: rpi5/ rpi4/ orangepi/   (a new board is one entry)
    runtime/           the capability host — mounts what DEVICE.md declares
  web/                 on-device setup + monitor UI
devices/               declarative overlays: _base/, lamp/, intern/, examples/
cts/                   conformance test suite — "is this an Autonomous device?"
tools/                 imager, scripts (build + SBC images)
companions/            off-device apps (lamp-buddy, desktop-buddy)
backend/               off-device cloud inference (dlbackend)
docs/  MAINTAINERS  GOVERNANCE.md  ...
```

The `Drivers` and `Board` layers are surfaced as `os/hal/drivers/<subsystem>` and
`os/hal/board/<board>` — the Linux split made physical. This requires renaming today's
`lelamp` package and its deployed identifiers; see [Migration](#migration).

## The device contract (our "VINTF")

A device declares what it **provides**; the OS version declares what it **requires**; the two
are matched before the OS boots or updates — Android's Treble check, simplified.

- **Provides** — `devices/<id>/DEVICE.md` front matter: the capability groups this body has,
  the schema version, the board.
- **Requires** — the OS targets a `contract` schema version (`autonomous.device.v1`). A device
  declaring an unknown major version is refused, not crashed.
- **Match** — at boot the runtime mounts only declared capabilities (declaration-driven
  mounting); a declared-but-absent *required* driver fails loud; an undeclared one is skipped.

This is why **Intern is Lamp minus `motion` + `display`** — one image, two manifests, no fork.

## Capabilities are feature flags

Every ability is a namespaced capability (`motion.move`, `vision.snapshot`) in
`contract/capabilities.md` — Android's `uses-feature`. A `SKILL.md` declares the capabilities
it **requires** and **optionally** uses; the skill manager filters skills off devices that
lack a required capability and degrades gracefully for optional ones. One skill catalog spans
very different robots.

## Conformance — spec + test

- **`contract/COMPATIBILITY.md`** is the spec (the CDD): what a device MUST, SHOULD, and MAY do
  to call itself Autonomous-compatible.
- **`cts/`** is the test (the CTS): it validates a `DEVICE.md` against the spec, checks the
  capability contract, and verifies safety invariants. Passing `cts` is what lets a third party
  ship an "Autonomous-compatible" device. A test is the only thing that stops a hundreds-of-OEM
  ecosystem from fragmenting.

## Open core, licensed services

The OS is Apache-2.0 and fully open. The differentiated layer ships separately: premium
character packs (`soul_ref`), memory continuity, Grid inference, the skill store. Anyone can
build a device; the best experience comes from Autonomous. *(Android's AOSP/GMS split.)* See
[GOVERNANCE.md](../../GOVERNANCE.md).

## Governance — federated maintainers

A `MAINTAINERS` file maps each subsystem and device class to an owner; third parties submit
upstream to that owner, not to one core team. A BDFL breaks ties and owns the frozen
`contract/`. *(Linux.)* Docs live in-tree beside the code they describe.

## Migration

The architecture is ~85% in place. The remaining work is surfacing the lower layers and adding
the conformance machinery, staged so nothing breaks:

1. **Conformance + base** (additive, safe): `contract/COMPATIBILITY.md`, `cts/`, `devices/_base`.
2. **Surface the Go layers** (CI-validated): `os/core` → `os/services` + `os/runtime`.
3. **Rename the HAL package** (`lelamp` → neutral) and split it into `os/hal/{drivers,board,runtime}`
   — the one change that touches deployed identifiers (`/opt/lelamp`, systemd units, `LELAMP_*`
   env, GCS `lamp/ota/lelamp`). Done as a **back-compat field migration**: add the new names,
   alias the old, cut devices over an OTA cycle, then drop the old.
4. **Adapt Lamp + Intern** onto the final structure; both stay one image, two manifests.

Stages 1–2 are safe and CI-gated. Stage 3 is coordinated with anyone shipping from this tree.
