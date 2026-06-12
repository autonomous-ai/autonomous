# Servo Calibration

The arm uses 5x Feetech STS3215 servos controlled via the `lerobot` library. Each servo requires calibration to map raw encoder positions to degrees. Calibration data is stored as JSON and loaded at startup.

## Calibration files (repo-local, shared by the whole fleet)

Calibration is **version-controlled in this repo** and shared across all devices
(one calibration for identical arm hardware). The follower/leader configs override
lerobot's per-user default (`~/.cache/huggingface/lerobot/calibration`) and pin
`calibration_dir` to the repo via `__post_init__`:

- `os/hal/follower/config_hal_follower.py` → `os/hal/calibration/robots/hal_follower`
- `os/hal/leader/config_hal_leader.py` → `os/hal/calibration/teleoperators/hal_leader`

> **Why the override:** the per-user `~/.cache` path breaks when the service user
> differs from the user that ran calibration. `hal.service` runs as **root**, so it
> would look under `/root/.cache/...` and miss a calibration saved under another
> user's home — surfacing as `FeetechMotorsBus(...) has no calibration registered`.
> The repo-local path is independent of the service user.

lerobot loads `calibration_dir / f"{id}.json"`, where `id` is `HAL_DEVICE_ID`
(`os/hal/config.py`, default `"hal"`; `os/hal/.env.example` sets `HAL_DEVICE_ID=hal`).
With the default id the files are:

```
os/hal/calibration/
├── robots/hal_follower/hal.json
└── teleoperators/hal_leader/hal.json
```

On a deployed device these ship under `/opt/hal/calibration/...`.

> **Shared-calibration assumption:** the whole fleet reads the same committed
> `hal.json`. Keep `HAL_DEVICE_ID=hal` (or unset) on every device — if a device
> sets a different `HAL_DEVICE_ID`, lerobot looks for `<id>.json` and won't find
> the shared file, so the servos start **uncalibrated** (wrong ranges, possible
> over-travel). For per-device calibration, set a distinct `HAL_DEVICE_ID` and
> commit a matching `<id>.json` instead.

Each JSON contains per-servo values:

| Field | Description |
|-------|-------------|
| `id` | Servo ID (1–5) |
| `drive_mode` | Direction (0 = normal) |
| `homing_offset` | Encoder offset for center position |
| `range_min` | Minimum encoder value (physical limit) |
| `range_max` | Maximum encoder value (physical limit) |

## Servos

| ID | Name | Function |
|----|------|----------|
| 1 | `base_yaw` | Left/right rotation |
| 2 | `base_pitch` | Forward/backward tilt |
| 3 | `elbow_pitch` | Elbow bend |
| 4 | `wrist_roll` | Wrist rotation |
| 5 | `wrist_pitch` | Wrist tilt |

## Deploy to a device

No copy-to-`~/.cache` step is needed: the calibration ships with the repo checkout
under `/opt/hal/calibration/...` and is read directly. After deploying new code or a
new calibration, restart the HAL service:

```bash
sudo systemctl restart hal
```

## Run fresh calibration on a Pi

Use this when the arm hardware changes (e.g. after replacing servos). Calibration
writes back into the repo-local dir via the same `calibration_dir` override, so use
the same `id` the runtime uses (`hal`).

```bash
# Follower only
sudo /opt/hal/.venv/bin/python3 -m hal.calibrate \
  --id hal --port /dev/ttyACM0 --follower-only

# Leader only
sudo /opt/hal/.venv/bin/python3 -m hal.calibrate \
  --id hal --port /dev/ttyACM0 --leader-only

# Both follower and leader
sudo /opt/hal/.venv/bin/python3 -m hal.calibrate \
  --id hal --port /dev/ttyACM0
```

### Calibration steps (interactive)

1. **Torque is disabled** — servos go limp so you can move them by hand.
2. **Move all joints to the middle** of their range of motion, then press ENTER. This sets the homing offset.
3. **Move each joint through its full range** (min to max). The script records encoder positions. Press ENTER when done.
4. Calibration is saved to `os/hal/calibration/robots/hal_follower/hal.json`.

### After calibration

- Commit the updated calibration file(s) under `os/hal/calibration/` so the fleet picks them up.
- Restart the HAL service: `sudo systemctl restart hal` (or reboot).
