# Vision Tracking — Object Follow with Servo

Lamp can track and follow any object the user names. A detector finds the object by name and seeds a ViT tracker, then a fast vision loop follows it in real time while a decoupled servo worker glides the head smoothly toward the target.

All tracking code lives in `os/hal/drivers/tracking/tracker_service.py`.

## Architecture

```
User: "Lamp, follow the cup"
         |
    POST /servo/track {"target": "cup"}
         |
    1. Freeze servos 0.3s → grab a sharp frame
         |
    2. Detect the object (YuNet face | local YOLOv8n | remote YOLOWorld) → bbox
         |
    3. TrackerVit init on the bbox
         |
    4. Two decoupled threads:
         |   a. Vision loop @ FAST_LOOP_FPS (10):
         |        ViT update → alpha-beta centroid filter → soft dead zone
         |        → PID + velocity feedforward → publish an absolute servo goal
         |        (background YOLO re-detect every 1.5s corrects drift)
         |   b. Servo worker: SmoothDamp glide toward the latest goal
         |        (ease-in/ease-out, one bus write per ~30ms tick)
         |
    5. Lost / bloated / no-detect / timeout → auto-stop, hold or return to zero
```

The vision loop never blocks on motor motion: it publishes an *absolute* servo goal and moves on to the next frame. The servo worker owns the physical motion and continuously eases toward whatever the latest goal is. This is what keeps both the tracker fps high and the head motion smooth.

### Downscaled vision, original-resolution math

The camera runs **1280×720**. Every heavy vision component — the ViT tracker and all three detectors — runs on a frame downscaled to `VISION_MAX_WIDTH` (640 px wide, 0.5× → ¼ the pixels) for speed. Each bbox they produce is mapped **back to original 1280×720 coordinates** before any servo/PID math (`_downscale` / `_scale_bbox` / `_vit_init` / `_vit_update`, and `detect_object` is transparent). Because the coordinate contract downstream is always original resolution, none of the pixel-tuned constants (PID gains, gates, dead zones, feedforward thresholds) change when the downscale factor changes. Set `VISION_MAX_WIDTH = 0` to disable.

## Detection

`detect_object(frame, target)` returns a bbox `(x, y, w, h)` in original camera coords, trying three paths in order:

| Path | Detector | When | Speed (A523) |
|------|----------|------|--------------|
| 0 | **YuNet** face detector (`face_detection_yunet_2023mar.onnx`) | target ∈ {`face`, `human face`, `khuôn mặt`, `mặt`} | ~30 ms |
| 1 | **Local YOLOv8n** (COCO classes, `yolov8n.pt`, imgsz=320) | target maps to a COCO class | ~260–770 ms |
| 2 | **Remote YOLOWorld** open-vocab (`{DL_BACKEND_URL}/detect/yoloworld`) | non-COCO target, or local miss (fallback) | ~1.3–2.8 s |

- COCO has no hand/face class, so `hand`/`face` intentionally fall through to YuNet/YOLOWorld instead of mapping to `person` (which locked onto the whole body).
- On a local-YOLO miss the code falls back to remote YOLOWorld, **throttled** to at most one attempt per `REMOTE_FALLBACK_MIN_INTERVAL` (2.0 s) so a genuinely unseeable target doesn't hit remote every redetect.
- Detection quality filters: confidence ≥ `DETECT_MIN_CONFIDENCE` (0.15), area between `DETECT_MIN_AREA_RATIO` (0.3%) and `DETECT_MAX_AREA_RATIO` (80%) of frame.

Weights are checked into the repo (`os/hal/drivers/tracking/models/`) so deploy is one rsync and the Pi never needs internet at boot to start tracking.

## Tracker: TrackerVit

**Model:** `os/hal/drivers/tracking/models/vittrack.onnx` (checked into repo)

| Feature | Value |
|---------|-------|
| Speed | ~15–25 ms/frame on the downscaled frame |
| Confidence score | `getTrackingScore()` 0.0–1.0 per frame |
| Scale handling | Auto-adjusts bbox size |
| Loss detection | Returns `ok=False` + low score when object disappears |

**Fallback chain:** TrackerVit → CSRT → KCF → MIL. Only ViT exposes a confidence score (used for ghost-lock detection); the others return 1.0.

## Servo Control

Tracking drives 4 joints:

- **base_yaw** (ID 1) — left/right pan (100 % of yaw)
- **base_pitch** (ID 2) — up/down tilt, 10 % of pitch
- **elbow_pitch** (ID 3) — up/down tilt, 90 % of pitch
- **wrist_pitch** (ID 5) — up/down tilt, 0 %

Pitch is concentrated on the elbow (`PITCH_WEIGHT_ELBOW = 0.90`). Empirically only pure-rotation joints move the object toward center; base/wrist mostly translate the camera (kinematic coupling), so their weights are low/zero. The elbow motor's positive direction was reversed in hardware, so its contribution carries `ELBOW_PITCH_SIGN = -1.0`.

### Control law (vision loop → servo goal)

Each frame the loop turns the tracker bbox into an absolute servo goal:

1. **Alpha-beta filter on the centroid** (`AlphaBetaFilter2D`) — a constant-velocity steady-state Kalman. Smooths jitter, coasts through dropped/garbage frames on prediction, gates outlier teleports (`AB_GATE_PX`), and exposes a velocity estimate. A velocity lead (`AB_LEAD_S = 0.12 s`) aims slightly ahead of the target.
2. **Soft dead zone** (`_soft_deadband`) — the error is 0 inside the dead zone and ramps up from 0 at the edge (no value step). This removes the "kick out of center" jerk the old hard dead zone produced.
3. **PID + velocity feedforward** — a time-aware PID with anti-windup on the soft-deadbanded position error, **plus** a feedforward term proportional to the target's measured pixel velocity (`VFF_GAIN`). The feedforward pans the camera *at the target's speed* even at zero position error, so a steadily moving target is a continuous pan instead of catch-up bursts. A position-centered but moving target keeps panning (does not freeze in the dead zone). Combined output is clamped to `PID_OUTPUT_MAX_DEG` (5°).
4. **Publish goal** — the resulting absolute joint target is handed to the servo worker (non-blocking).

### Servo worker (SmoothDamp follower)

`_servo_worker` runs on its own thread and continuously eases the joints toward the latest goal using **SmoothDamp** (`_smooth_damp`, a critically-damped follower): each joint carries its own velocity, so every move accelerates smoothly and eases out into the target, and a fresh goal arriving mid-move retargets without a restart jerk — the cinematic "film camera" motion. It issues **one bus write per `SERVO_SUBSTEP_SLEEP` (30 ms) tick**, the same click cadence as the old fixed-substep ramp (the Feetech STS3215 clicks on each write, so the write rate must stay bounded — SmoothDamp changes *what* is commanded per tick, not *how often*).

Hardware motion limits during tracking: `TRACKING_GOAL_VELOCITY = 150` steps/s and `TRACKING_ACCELERATION = 30` (gentle ramp). Restored to snappy defaults when tracking ends.

### Drift correction & lock management

- **Background YOLO re-detect** every `YOLO_REDETECT_S` (1.5 s) on a worker thread (never blocks the fast loop; result delivered via a `maxsize=1` queue). Forced immediately when the object nears a frame edge (>25 %) or on the first CSRT miss.
- **Reinit gating (SORT/ByteTrack-style)** — a re-detect only reinitializes the tracker when it has clearly diverged, to avoid the reinit churn that whipsaws the servo:
  - **Area gate** `YOLO_AREA_GATE_MULT` (4.0) — reject a detection whose area is >4× or <¼ the median of the last 5; don't reinit to it.
  - **Reinit debounce** `REINIT_COOLDOWN_S` (0.5 s) — rate-limit reinits; bypassed only when the lock is clearly lost (`center_dist > frame_diag × LOST_CENTER_FRAC` = 0.5).
- **Bbox-trust guard (bloat hold)** — when the ViT lock dissolves into an oversized box the centroid is garbage, so the servo holds instead of chasing it:
  - `BBOX_FREEZE_RATIO` (1.0) — bbox ≥ full frame area ⇒ ViT dissolved.
  - `BLOAT_HOLD_MULT` (3.0) — bbox > 3× the last trusted lock area ⇒ hold and force a re-detect.
- **Detector-gated trust** — if no detector has confirmed for `TRUST_TRACKER_S` (2.5 s) and ViT confidence < `TRACKER_TRUST_CONF` (0.4), hold the servo (`WAIT-YOLO`) rather than chase a phantom; high ViT confidence keeps firing even without a fresh detector confirm.

### Pixel-to-Degree Conversion

```
deg_per_px = CAMERA_FOV_DEG / frame_width          (same on both axes for square pixels)

dx = filtered_lead_x - frame_width/2   (positive = right)
dy = filtered_lead_y - frame_height/2  (positive = below)

yaw_step         = clamp(PID(soft_deadband(dx)) + VFF·vx·deg_per_px·dt,  ±5°)
pitch_correction = clamp(PID(soft_deadband(dy)) + VFF·vy·deg_per_px·dt,  ±5°)
```

### Tuning Constants

| Constant | Value | Description |
|----------|-------|-------------|
| `VISION_MAX_WIDTH` | 640 | Downscale width for ViT + detectors (0 = off) |
| `FAST_LOOP_FPS` | 10 | Vision loop frequency |
| `CAMERA_FOV_DEG` | 60 | Horizontal FOV, for px→deg |
| `DEAD_ZONE_YAW_PCT` / `_PITCH_PCT` | 0.07 / 0.05 | Soft dead zone as fraction of frame |
| `PID_YAW_KP` / `PID_PITCH_KP` | 0.025 / 0.03 | PID proportional gains |
| `PID_OUTPUT_MAX_DEG` | 5.0 | Max degrees per fire (yaw & combined pitch) |
| `AB_ALPHA` / `AB_BETA` | 0.6 / 0.2 | Alpha-beta position/velocity gains |
| `AB_GATE_PX` | 200 | Reject a centroid teleport beyond this residual |
| `AB_LEAD_S` | 0.12 | Velocity lead (aim ahead of the target) |
| `VFF_GAIN` | 0.6 | Fraction of target velocity fed forward |
| `VFF_MAX_DT_S` | 0.20 | Cap on per-fire dt for feedforward |
| `VFF_MOVING_MIN_PXS` | 40 | Target speed above which a centered target keeps panning |
| `SERVO_SMOOTH_TIME` | 0.18 | SmoothDamp time constant (↓ snappier, ↑ smoother/laggier) |
| `SERVO_MAX_SPEED_DPS` | 60 | SmoothDamp peak pan speed cap |
| `SERVO_SUBSTEP_SLEEP` | 0.030 | Servo-worker tick / bus-write period |
| `TRACKING_GOAL_VELOCITY` | 150 | Hardware velocity limit (steps/s) |
| `TRACKING_ACCELERATION` | 30 | Hardware acceleration ramp |
| `PITCH_WEIGHT_BASE/ELBOW/WRIST` | 0.10 / 0.90 / 0.0 | Pitch distribution across joints |
| `ELBOW_PITCH_SIGN` | -1.0 | Elbow polarity (hardware reversed) |
| `YOLO_REDETECT_S` | 1.5 | Background re-detect interval |
| `YOLO_AREA_GATE_MULT` | 4.0 | Reject re-detect area outliers |
| `REINIT_COOLDOWN_S` | 0.5 | Min seconds between tracker reinits |
| `BBOX_FREEZE_RATIO` | 1.0 | Bbox ≥ frame ⇒ ViT dissolved (hold) |
| `BLOAT_HOLD_MULT` | 3.0 | Bbox > 3× trusted lock ⇒ hold |
| `CONFIDENCE_THRESHOLD` | 0.15 | Below this = low-confidence frame |
| `MAX_LOW_CONFIDENCE_FRAMES` | 10 | Consecutive low-confidence frames → stop |
| `YOLO_MAX_MISS` | 30 | Consecutive CSRT misses before retry |
| `MAX_TRACK_DURATION_S` | 300 | Auto-stop timeout (5 min) |
| `_LOCAL_IMGSZ` | 320 | Local YOLO inference size (640 → 1.3–2.9 s, too slow) |

> Legacy note: the `GIMBAL_GAIN` / `GIMBAL_MAX_STEP` / `EMA_ALPHA` proportional path (`_fire_gimbal` / `_send_gimbal_target`) is **dead** — live control is the PID + feedforward path (`_fire_pid`). Don't tune those for responsiveness.

### Servo Position Limits

| Joint | Min | Max |
|-------|-----|-----|
| base_yaw | -135 | 135 |
| base_pitch | -90 | 30 |
| elbow_pitch | -90 | 90 |
| wrist_pitch | -90 | 90 |

## Auto-Stop Conditions

| Condition | Action |
|-----------|--------|
| `confidence < 0.15` for 10 frames | Stop — lost target |
| Bbox shrinks below `DETECT_MIN_AREA_RATIO` | Stop — ghost-lock on a sliver |
| Bbox overflows frame + no detect for 3 s | Forced retry, then stop if unrecovered |
| No detector confirm for `STOP_NO_YOLO_S` (20 s) | Stop — ghost tracking |
| CSRT misses `YOLO_MAX_MISS` (30) after `MAX_TRACKING_RETRIES` (4) | Stop — object gone |
| Tracking duration > 5 minutes | Stop — timeout to save motor/CPU |

Note: a large bbox (e.g. a person filling the frame) is **not** a stop condition — PID drives off the centroid, not bbox size, so a close object still tracks. When tracking ends the arm glides back to zero at tracking speed (no snap).

### Auto-stop on gateway/network disconnect

Object tracking is driven by remote vision updates from the agent/cloud. When the gateway WebSocket disconnects (cloud or internet loss), the device auto-stops any in-flight servo tracking — `os/services/internal/openclaw/service_ws.go` calls `hal.StopServoTracking()` → HAL `POST /servo/track/stop` (best-effort, guarded by `SetUpCompleted`). Without fresh remote updates, continued tracking would keep aiming the body at a stale target it can no longer correct, so it is stopped as a safety reflex. Local idle animation continues (the device stays "alive", doesn't freeze) and recovery (`/servo/track/stop`, stop/release) stays available. See `devices/lamp/SAFETY.md` → `## fail-safe states` (Network/gateway loss row, enforced).

## API Endpoints

All under `/servo/track`.

### GET /servo/track/targets — List suggested targets

```json
{"targets": ["person", "cup", "bottle", "glass", "phone", "laptop", ...]}
```

Detection is open-vocabulary via YOLOWorld (and YuNet for faces) — any text works, this list is just suggestions.

### POST /servo/track — Start tracking

`target` accepts either a single string or a list of candidate labels. When a list is passed, the first non-empty label is used. Useful when the caller (e.g. an LLM skill) is unsure which exact label will match.

```json
// Auto-detect, single label
{"target": "cup"}

// Auto-detect, list of candidate labels (preferred from LLM skills)
{"target": ["cup", "mug", "coffee cup"]}

// Manual bbox (skip detection — target is for display only)
{"bbox": [190, 50, 170, 300], "target": "cup"}

// Response
{
  "status": "ok",
  "tracking": true,
  "target": "cup",
  "bbox": [190, 50, 170, 300],
  "confidence": 1.0
}
```

### POST /servo/track/stop — Stop tracking

```json
{"status": "ok", "tracking": false}
```

### GET /servo/track — Check status

```json
{
  "status": "ok",
  "tracking": true,
  "target": "cup",
  "bbox": [195, 55, 175, 295],
  "confidence": 0.612
}
```

### POST /servo/track/update — Re-initialize bbox

Manual re-init of the tracker with a new bbox without stopping the session (the background YOLO re-detect handles drift automatically; this is for callers that want explicit control).

```json
{"bbox": [250, 160, 75, 95], "target": "cup"}
```

## End-to-End Flow

### Happy path

```
1. User: "Lamp, follow the cup"
2. Agent calls POST /servo/track {"target": "cup"}
3. HAL internally:
   a. Freezes servos 0.3s and snapshots a sharp frame
   b. Detects "cup" (local YOLOv8n, or remote YOLOWorld) → bbox
   c. TrackerVit init uses the same frame + bbox (coordinates match)
   d. Starts the vision loop + servo worker
4. Servo pans smoothly to follow the cup, background YOLO corrects drift
5. User: "OK stop" → agent calls POST /servo/track/stop
6. Servo glides back to zero
```

### Auto-stop on lost

```
1. Object leaves frame or is occluded
2. TrackerVit confidence drops below 0.15 (or ViT lock dissolves)
3. Background YOLO can't re-find it → after the guards trip → auto-stop
4. Arm returns to zero
5. Agent can notify user or re-issue the follow command
```

## Camera Stream Overlay

When tracking is active, the MJPEG stream (`/camera/stream`) draws:
- Green bounding box around the tracked object
- Target label above the box

## Web UI

Camera section shows:
- **Vision Tracking card** — target input, bbox input, Start/Stop/Status buttons
- **Stream badge** — "LIVE" or "TRACKING: {target}"
- **Confidence** — shown in tracking info panel
- **Polling** — status refreshes every 3 seconds

## Dependencies

- `opencv-python>=4.8.0` (already in `pyproject.toml`)
- `ultralytics` — local YOLOv8n inference
- `vittrack.onnx`, `yolov8n.pt`, `face_detection_yunet_2023mar.onnx` — checked into `os/hal/drivers/tracking/models/`
- `requests` (already in project)
- **YOLOWorld API** — DL backend at `{DL_BACKEND_URL}/detect/yoloworld` (open-vocab fallback only)

## Interaction with Other Systems

| System | During tracking | After tracking |
|--------|----------------|----------------|
| Servo idle animation | Suppressed (`_hold_mode`) | Resumed |
| `/servo/play` | Blocked by `_hold_mode` | Resumed |
| Sensing (face, motion) | Continues — shares camera | Continues |
| Camera stream overlay | Green bbox drawn | Normal stream |
| TTS | Continues normally | Continues normally |

## Performance Notes

- Fast-loop CPU floor on the Allwinner A523 is ViT inference + detector cost; the frame downscale (`VISION_MAX_WIDTH`) and local imgsz=320 are the main levers.
- Motion smoothness comes from the decoupled servo worker + SmoothDamp + velocity feedforward; the alpha-beta filter + reinit gating keep the goal itself stable so the follower isn't chasing noise.
- Small/far objects (e.g. a cup across the room) can exceed both local and remote detector resolution — a perception limit, not a control bug.
