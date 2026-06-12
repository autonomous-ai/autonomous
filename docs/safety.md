# Safety Engine

The safety engine enforces a device's `SAFETY.md` bounds **deterministically in the
runtime**, below the agent. It is the mechanism behind the first principle in
`contract/SAFETY-SPEC.md`: *safety is below the brain.* The agent requests actions;
the engine decides — on every request, regardless of who issued it — whether the
hardware is allowed to honour them and within what limits.

> **Status:** design + incremental build. Slice 1 (brightness ceiling) is the first
> vertical slice; later slices reuse the same loader + gate. Each table row below is
> marked enforced / reserved. This document tracks the code, not the other way around.

## Why an engine, not prompting

Routing safety through the language model is unreliable — it can be argued out of a
refusal, it can hallucinate a limit, and it cannot guarantee an action *did not*
happen. Guard mode in this codebase was already rebuilt to deliver alerts
deterministically for exactly this reason. The safety engine generalises that: the
runtime, not the gateway, is the single point that clamps, blocks, or stops.

## Architecture

Three layers, mirroring the device layer (`DEVICE.md` → capability → route → driver):

```
SAFETY.md front matter        the declared bounds (machine contract; per capability group)
        │  resolved via DEVICE.md safety_ref (path or http), parsed at boot
        ▼
os/hal/safety/policy.py        pure SafetyPolicy + gate functions (no IO, unit-testable)
        │  clamp_brightness(requested) -> min(requested, ceiling)   [slice 1]
        ▼
HAL capability routes          call the gate BEFORE actuating (led, later servo/music)
        │  deterministic, in-process, cannot be bypassed by the agent
        ▼
hardware
```

- **`SAFETY.md` front matter** — the bounds, keyed by capability group. Schema and
  field table: `contract/SAFETY-SPEC.md`.
- **`os/hal/safety/policy.py`** — a pure loader (regex front-matter parse,
  dependency-free, same discipline as `os/hal/board/device.py`) producing a typed
  `SafetyPolicy`, plus pure gate functions. No hardware, no clock side effects, fully
  unit-testable off-hardware.
- **The routes** — each capability route consults the gate at the point of actuation.
  The LED route clamps brightness; later the servo route clamps speed/accel and
  guarantees `stop`. Because the gate is a plain function call in the request path,
  there is no path around it.

The policy is loaded once at boot (alongside the `DeviceProfile`) and exposed via the
device endpoint so the declared bounds are observable: `GET /device` already returns
`safety_ref`; the resolved bounds surface there too.

## Fail-safe semantics

Per-capability criticality (full rule in `contract/SAFETY-SPEC.md`):

| Capability | Bound absent / unloadable | Rationale |
|------------|---------------------------|-----------|
| light, audio | pass-through (log only) | a calm light/quiet speaker is not a hazard; do not invent a limit |
| motion | **fail-closed** (refuse actuation) | moving against unknown limits is a hardware fault |

`SAFETY.md` is optional. The schema tag is validated like `DEVICE.md`'s — a missing or
unknown-major `schema` aborts boot rather than enforce an ABI it cannot read.

## Slice roadmap

| Slice | Scope | Gate | Enforced where | Status |
|-------|-------|------|----------------|--------|
| 1 | `light.max_brightness` ceiling | `clamp_brightness` | LED route | building first |
| 2 | `quiet_hours` (light + audio) | `quiet_now` + reduced ceiling | LED + music routes | reserved |
| 3 | `motion.max_speed`/`max_accel`, `stop_always` (fail-closed) | `clamp_motion`, stop guarantee | servo route | reserved |
| 4 | fail-safe states (network loss → hold pose, board fault → disable capability) | state gate | lifespan + routes | reserved |

Each slice adds fields to the `SafetyPolicy` and gate functions and wires one or more
routes; the loader and the front-matter contract do not change shape between slices
(fields are only added — the `autonomous.safety.v1` ABI).

## Verifying enforcement

A safety bound is only real if you can *prove* it holds and that the agent cannot get
around it. Each slice is verified at three levels; a bound is not "done" until all
three pass. (This is distinct from `devices/lamp/docs/security-test.md`, which covers
network/access-control security — ports, RCE, CORS — not actuation bounds.)

1. **Unit (pure gate, off-hardware).** The gate function is pure, so its limit is a
   table test: a request above the ceiling clamps to it, a request below passes
   through unchanged, an absent bound behaves per the fail-safe rule. Runs in CI with
   no device.
2. **Runtime (on the device, through the real route).** Issue the actuation request
   over HTTP and observe the hardware-bound value, not the requested one. The declared
   bound is also observable at `GET /device`, so the test asserts *request vs. ceiling
   vs. observed output* line up.
3. **Bypass audit (the safety-critical check).** Confirm there is **no** path to the
   actuator that skips the gate — issue the same action through every route that can
   drive it (agent path, direct route, any raw/low-level endpoint) and confirm each is
   clamped. A bound enforced on one path but reachable on another is not enforced.

### Slice 1 — brightness ceiling (checklist)

- [ ] **Unit:** `clamp_brightness(255)` with `max_brightness: 180` → `180`;
      `clamp_brightness(120)` → `120`; no `max_brightness` declared → returns the
      request unchanged (pass-through) and logs that no ceiling is set.
- [ ] **Runtime:** `GET /device` reports the resolved `light.max_brightness`. A
      `POST` to the LED route at full white (255) lights the ring at the ceiling, not
      255; a request below the ceiling is untouched.
- [ ] **Bypass audit:** every code path in `os/hal/routes/led.py` (and any effect that
      sets pixels) goes through the gate — there is no direct `rgb_service` brightness
      write that skips `clamp_brightness`. Grep the route + effects module to confirm.
- [ ] **Determinism:** the clamp is identical whether the request came from the agent,
      the Web UI, or a raw `curl` — the gate does not consult who asked.

Later slices extend this checklist (quiet-hours: assert reduced ceiling inside the
window and normal outside, with an injected clock; motion: assert fail-closed when
bounds are absent, and that `stop` preempts an in-flight move).

## Relationship to existing ad-hoc enforcement

Some safety behaviour already exists, hardcoded and scattered: `motion.stop()` in the
motors/animation services, lerobot's mechanical position clamp, the LED brightness
scaling config. The engine does not rip these out at once — it *centralises* them
into the declared policy one slice at a time, so the bounds become data a device
declares rather than constants buried in drivers. Slice 1 introduces the engine with
a bound that has **no** prior enforcement (an agent-independent brightness ceiling),
proving the path end to end before migrating the existing pieces.
