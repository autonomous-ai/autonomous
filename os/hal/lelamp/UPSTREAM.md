# LeLamp Runtime — Upstream Tracking

## Source

- **Repo**: https://github.com/humancomputerlab/lelamp_runtime
- **Commit**: `ee23699` (Update README.md)
- **Date copied**: 2026-03-25

## What we use

- `service/base.py` — ServiceBase (event-driven, priority dispatch)
- `service/motors/motors_service.py` — MotorsService
- `service/motors/animation_service.py` — AnimationService (smooth interpolation)
- `service/rgb/rgb_service.py` — RGBService (64x WS2812)
- `follower/` — LeLampFollower (Feetech servo bus via lerobot)
- `recordings/*.csv` — Pre-recorded animations

## What we ignore (dead code, not imported)

- `leader/` — LeLamp leader arm (not relevant)
- `livekit-agents`, `openai` dependencies — replaced by OpenClaw
- `calibrate.py`, `record.py`, `replay.py` — CLI tools, not imported by server

## What we changed (Lamp-only additions to animation_service.py)

- `bus_lock` — camera + servo share serial bus, need serialization
- `freeze/unfreeze` — pause servo writes during camera capture
- `_motor_positions_from_bus()` — read positions without triggering camera
- `_sync_state_from_hardware()` — prevent interpolation from assuming 0° for missing joints
- `_configure_servos_raw()` — configure servos even when connect() partially fails
- `move_to()` + `STARTUP_POSITION` — smooth interpolated boot position
- Music groove support — loop music_groove recording during music playback
- Auto-play idle on start — same as upstream behavior

## Rules for modifying upstream code

1. **Do NOT override upstream servo config** — P_Coefficient=16 is intentional, higher values cause jerky motion
2. **Do NOT add software joint limits** — servo firmware + calibration handle range safety at hardware level
3. **Do NOT regenerate upstream recordings** — teleop recordings are natural, math-generated ones are worse
4. **Do NOT add per-frame processing** (clamp, transform) to the playback loop — send CSV values directly
5. **Only add code Lamp actually needs** — bus_lock, freeze, move_to. Don't touch motion core logic

## How to sync

1. Check upstream for driver-level fixes (servo protocol, LED timing)
2. Cherry-pick relevant changes manually
3. Ignore upstream AI/LiveKit changes
4. Update commit hash above after sync
