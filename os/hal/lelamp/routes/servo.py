"""Servo route handlers — all /servo/* endpoints."""

import csv
import io
import os
import re
import threading
import time
from pathlib import Path
from typing import Optional

from fastapi import APIRouter, HTTPException, File, Form, UploadFile

import lelamp.app_state as state
from lelamp.models import (
    ServoAimRequest,
    ServoAimResponse,
    ServoNudgeRequest,
    ServoMoveRequest,
    ServoMoveResponse,
    ServoPositionResponse,
    ServoRequest,
    ServoStateResponse,
    ServoStatusResponse,
    ServoTrackRequest,
    ServoTrackResponse,
    StatusResponse,
)
from lelamp.presets import (
    AIM_LEFT,
    AIM_PRESETS,
    AIM_RIGHT,
    SERVO_CMD_PLAY,
)

router = APIRouter(tags=["Servo"])

# --- Constants ---

_SERVO_JOINT_FIELD_RE = re.compile(r"^[A-Za-z0-9_]+\.pos$")
_MAX_SERVO_RECORDING_UPLOAD_BYTES = 2 * 1024 * 1024  # 2MB
_MAX_SERVO_RECORDING_ROWS = 20000


def _sanitize_recording_name(name: str) -> str:
    name = (name or "").strip()
    name = re.sub(r"[^a-zA-Z0-9_-]+", "_", name)
    name = name.strip("_- ")
    if not name:
        raise ValueError("empty recording name")
    return name[:64]


# --- Endpoints ---


@router.get("/servo", response_model=ServoStateResponse)
def get_servo_state():
    """Get available recordings and current animation state."""
    if not state.animation_service:
        raise HTTPException(503, "Servo not available")
    return {
        "available_recordings": state.animation_service.get_available_recordings(),
        "current": state.animation_service._current_recording,
    }


@router.post("/servo/upload", response_model=StatusResponse)
async def upload_servo_recording(
    file: UploadFile = File(...),
    recording_name: Optional[str] = Form(None),
):
    """Upload a servo recording CSV and make it available in GET /servo."""
    if not state.animation_service:
        raise HTTPException(503, "Servo not available")

    orig_filename = file.filename or "recording.csv"
    if orig_filename.lower().endswith(".csv") is False:
        raise HTTPException(400, "upload must be a .csv file")

    rec_name = recording_name or Path(orig_filename).stem
    try:
        rec_name = _sanitize_recording_name(rec_name)
    except ValueError as e:
        raise HTTPException(400, str(e))

    content = await file.read()
    if len(content) == 0:
        raise HTTPException(400, "empty csv")
    if len(content) > _MAX_SERVO_RECORDING_UPLOAD_BYTES:
        raise HTTPException(
            413, f"csv too large (max {_MAX_SERVO_RECORDING_UPLOAD_BYTES} bytes)"
        )

    try:
        text = content.decode("utf-8")
    except UnicodeDecodeError:
        raise HTTPException(400, "csv must be utf-8 text")

    reader = csv.DictReader(io.StringIO(text))
    fieldnames = reader.fieldnames or []

    if "timestamp" not in fieldnames:
        raise HTTPException(400, 'missing required column "timestamp"')

    joint_fields = [f for f in fieldnames if f != "timestamp"]
    if not joint_fields:
        raise HTTPException(400, "missing joint columns (expected *.pos fields)")

    invalid_joint_fields = [f for f in joint_fields if not _SERVO_JOINT_FIELD_RE.match(f)]
    if invalid_joint_fields:
        raise HTTPException(
            400, f"invalid joint columns: {invalid_joint_fields}. Expected <name>.pos"
        )

    valid_joints = None
    try:
        if (
            state.animation_service.robot
            and state.animation_service.robot.bus
            and state.animation_service.robot.bus.motors
        ):
            valid_joints = {f"{m}.pos" for m in state.animation_service.robot.bus.motors}
    except Exception:
        valid_joints = None

    if valid_joints is not None:
        unknown = [j for j in joint_fields if j not in valid_joints]
        if unknown:
            raise HTTPException(
                400,
                f"unknown joint columns: {unknown}. Valid: {sorted(valid_joints)}",
            )

    actions: list[dict[str, float]] = []
    for row_idx, row in enumerate(reader):
        if len(actions) >= _MAX_SERVO_RECORDING_ROWS:
            raise HTTPException(
                400, f"too many rows (max {_MAX_SERVO_RECORDING_ROWS})"
            )

        ts_val = row.get("timestamp")
        try:
            _ = float(ts_val)
        except Exception:
            raise HTTPException(400, f"invalid timestamp at row {row_idx + 2}")

        action: dict[str, float] = {}
        for joint in joint_fields:
            v = row.get(joint)
            if v is None or v == "":
                raise HTTPException(400, f"missing value for {joint} at row {row_idx + 2}")
            try:
                action[joint] = float(v)
            except Exception:
                raise HTTPException(400, f"invalid float for {joint} at row {row_idx + 2}")

        actions.append(action)

    recordings_dir = os.path.join(os.path.dirname(os.path.dirname(__file__)), "recordings")
    Path(recordings_dir).mkdir(parents=True, exist_ok=True)
    csv_path = os.path.join(recordings_dir, f"{rec_name}.csv")

    with open(csv_path, "w", newline="") as f:
        f.write(text if text.endswith("\n") else text + "\n")

    try:
        state.animation_service._recording_cache[rec_name] = actions
    except Exception:
        pass

    return {"status": "ok"}


@router.post("/servo/play", response_model=StatusResponse)
def play_recording(req: ServoRequest):
    """Play a pre-recorded servo animation by name."""
    state.logger.debug("POST /servo/play recording=%s", req.recording)
    if not state.animation_service:
        raise HTTPException(503, "Servo not available")
    if getattr(state.animation_service, "_zero_mode", False) or getattr(state.animation_service, "_hold_mode", False):
        state.logger.debug("servo/play blocked: %s mode active",
                           "zero-hold" if state.animation_service._zero_mode else "hold")
        return {"status": "ok"}
    if not state.animation_service._running.is_set():
        state.animation_service._running.set()
        state.animation_service._event_thread = threading.Thread(
            target=state.animation_service._event_loop, daemon=True
        )
        state.animation_service._event_thread.start()
        state.logger.info("Animation event loop restarted via /servo/play")
    t0 = time.perf_counter()
    state.animation_service.dispatch(SERVO_CMD_PLAY, req.recording)
    state.logger.debug("servo dispatch took %.1fms", (time.perf_counter() - t0) * 1000)
    return {"status": "ok"}


@router.post("/servo/resume", response_model=StatusResponse)
def resume_servos():
    """Exit zero-hold mode and resume normal animation loop (plays idle)."""
    if not state.animation_service:
        raise HTTPException(503, "Servo not available")
    state.animation_service._zero_mode = False
    state.animation_service._hold_mode = False
    if not state.animation_service._running.is_set():
        state.animation_service._running.set()
        state.animation_service._event_thread = threading.Thread(
            target=state.animation_service._event_loop, daemon=True
        )
        state.animation_service._event_thread.start()
        state.logger.info("Animation event loop restarted via /servo/resume")
    state.animation_service.dispatch(SERVO_CMD_PLAY, state.animation_service.idle_recording)
    state.logger.info("Servo resumed from zero-hold mode")
    return {"status": "ok"}


@router.post("/servo/hold", response_model=StatusResponse)
def hold_servos():
    """Hold current pose -- suppress idle/ambient animations, torque stays ON."""
    if not state.animation_service:
        raise HTTPException(503, "Servo not available")
    state.animation_service._hold_mode = True
    state.logger.info("Servo hold mode activated -- idle suppressed, emotions still allowed")
    return {"status": "ok"}


@router.post("/servo/move", response_model=ServoMoveResponse)
def move_servo(req: ServoMoveRequest):
    """Send joint positions to servo motors with smooth interpolation."""
    if not state.animation_service:
        raise HTTPException(503, "Servo not available")
    if not state.animation_service.robot:
        raise HTTPException(503, "Servo robot not connected")
    valid_joints = {f"{m}.pos" for m in state.animation_service.robot.bus.motors}
    unknown = [j for j in req.positions if j not in valid_joints]
    if unknown:
        raise HTTPException(
            400, f"Unknown joints: {unknown}. Valid: {sorted(valid_joints)}"
        )

    errors = {}

    try:
        if req.duration > 0:
            state.animation_service.move_to(req.positions, duration=req.duration)
        else:
            with state.animation_service.bus_lock:
                state.animation_service.robot.send_action(req.positions)
    except Exception as e:
        errors["move"] = str(e)

    try:
        with state.animation_service.bus_lock:
            obs = state.animation_service.robot.get_observation()
        for joint, target in req.positions.items():
            actual = obs.get(joint)
            if actual is not None:
                error = abs(actual - target)
                if error > 5.0:
                    errors[joint] = (
                        f"position error {error:.1f} deg (target={target:.1f}, actual={actual:.1f})"
                    )
    except Exception as e:
        errors["read_position"] = str(e)

    return {
        "status": "error" if "move" in errors else "ok",
        "requested": req.positions,
        "clamped": req.positions,
        "duration": req.duration,
        "errors": errors if errors else None,
    }


@router.post("/servo/zero", response_model=StatusResponse)
def zero_servos():
    """Move all servos to 0 deg and hold (torque stays ON)."""
    if not state.animation_service:
        raise HTTPException(503, "Servo not available")
    if not state.animation_service.robot:
        raise HTTPException(503, "Servo robot not connected")
    state.animation_service._zero_mode = True
    state.animation_service._running.clear()
    if state.animation_service._event_thread and state.animation_service._event_thread.is_alive():
        state.animation_service._event_thread.join(timeout=3.0)
    zero_pos = {f"{m}.pos": 0.0 for m in state.animation_service.robot.bus.motors}
    zero_pos["elbow_pitch.pos"] = 0.0
    try:
        state.animation_service.move_to(zero_pos, duration=2.0)
    except Exception as e:
        state.logger.warning(f"Could not move to zero: {e}")
    state.animation_service._current_state = {k: 0.0 for k in zero_pos}
    return {"status": "ok"}


@router.post("/servo/release", response_model=StatusResponse)
def release_servos():
    """Move servos to idle position then disable torque (safe release)."""
    if not state.animation_service:
        raise HTTPException(503, "Servo not available")
    if not state.animation_service.robot:
        raise HTTPException(503, "Servo robot not connected")
    state.animation_service._running.clear()
    if state.animation_service._event_thread and state.animation_service._event_thread.is_alive():
        state.animation_service._event_thread.join(timeout=3.0)
    # Fully folded pose (elbow at max 90°) so the body is already at the
    # mechanical floor when torque is cut — no remaining gap to drop.
    rest_pos = {
        "base_yaw.pos": 0.0,
        "base_pitch.pos": -90.0,
        "elbow_pitch.pos": 90.0,
        "wrist_roll.pos": 0.0,
        "wrist_pitch.pos": 0.0,
    }
    try:
        # move_to commands the ramp but does not verify the servo physically
        # arrived. Under load the motor lags the command, so poll
        # Present_Position until every joint is within tolerance of rest_pos
        # before cutting torque — otherwise the body drops the remaining gap.
        state.animation_service.move_to(rest_pos, duration=4.0)
        from lelamp.service.motors.animation_service import (
            _motor_positions_from_bus,
        )

        tol_deg = 2.0
        deadline = time.perf_counter() + 3.0
        while time.perf_counter() < deadline:
            with state.animation_service.bus_lock:
                actual = _motor_positions_from_bus(
                    state.animation_service.robot
                )
            if all(
                abs(actual.get(k, 0.0) - v) <= tol_deg
                for k, v in rest_pos.items()
            ):
                break
            time.sleep(0.05)
        else:
            state.logger.warning(
                "rest_pos not reached within 3s; releasing anyway"
            )
    except Exception as e:
        state.logger.warning(f"Could not move to rest before release: {e}")
    bus = state.animation_service.robot.bus
    errors = {}
    with state.animation_service.bus_lock:
        for motor_name in bus.motors:
            try:
                bus.write("Torque_Enable", motor_name, 0)
            except Exception as e:
                errors[motor_name] = str(e)
    if errors:
        state.logger.warning(f"Servo release errors (offline?): {errors}")
    return {"status": "ok"}


@router.get("/servo/position", response_model=ServoPositionResponse)
def get_servo_position():
    """Read current servo joint positions."""
    if not state.animation_service:
        raise HTTPException(503, "Servo not available")
    if not state.animation_service.robot:
        raise HTTPException(503, "Servo robot not connected")
    try:
        with state.animation_service.bus_lock:
            obs = state.animation_service.robot.get_observation()
        positions = {k: v for k, v in obs.items() if k.endswith(".pos")}
        return {"positions": positions}
    except Exception as e:
        raise HTTPException(500, f"Failed to read position: {e}")


@router.get("/servo/status", response_model=ServoStatusResponse)
def get_servo_status():
    """Ping each servo and return per-joint online/offline status with angle."""
    if not state.animation_service:
        raise HTTPException(503, "Servo not available")
    if not state.animation_service.robot:
        raise HTTPException(503, "Servo robot not connected")
    bus = state.animation_service.robot.bus
    ph = bus.port_handler
    pk = bus.packet_handler
    from scservo_sdk import COMM_SUCCESS

    servos = {}
    with state.animation_service.bus_lock:
        for motor_name, motor_obj in bus.motors.items():
            key = f"{motor_name}.pos"
            sid = motor_obj.id
            detail = {"id": sid, "angle": None, "online": False, "error": None}
            try:
                _, result, _ = pk.ping(ph, sid)
                if result != COMM_SUCCESS:
                    detail["error"] = "no status packet"
                else:
                    detail["online"] = True
                    try:
                        pos = bus.read("Present_Position", motor_name)
                        detail["angle"] = float(pos)
                    except Exception as e:
                        detail["error"] = f"read failed: {e}"
            except Exception as e:
                detail["error"] = str(e)
            servos[key] = detail
    return {"servos": servos}


@router.get("/servo/aim")
def list_aim_directions():
    """List available aim directions."""
    return {"directions": list(AIM_PRESETS.keys())}


@router.post("/servo/aim", response_model=ServoAimResponse)
def aim_servo(req: ServoAimRequest):
    """Aim the lamp head to a named direction."""
    if not state.animation_service:
        raise HTTPException(503, "Servo not available")
    if not state.animation_service.robot:
        raise HTTPException(503, "Servo robot not connected")

    preset = AIM_PRESETS.get(req.direction)
    if preset is None:
        available = list(AIM_PRESETS.keys())
        raise HTTPException(
            400, f"Unknown direction '{req.direction}'. Available: {available}"
        )

    was_running = state.animation_service._running.is_set()
    if was_running:
        state.animation_service._running.clear()
        if state.animation_service._event_thread and state.animation_service._event_thread.is_alive():
            state.animation_service._event_thread.join(timeout=2.0)

    try:
        with state.animation_service.bus_lock:
            obs = state.animation_service.robot.get_observation()
        current = {k: v for k, v in obs.items() if k.endswith(".pos")}

        if req.direction in (AIM_LEFT, AIM_RIGHT):
            positions = {**current, "base_yaw.pos": preset["base_yaw.pos"]}
        else:
            positions = {**preset, "base_yaw.pos": current.get("base_yaw.pos", preset["base_yaw.pos"])}

        if req.duration > 0:
            state.animation_service.move_to(positions, duration=req.duration)
        else:
            with state.animation_service.bus_lock:
                state.animation_service.robot.send_action(positions)
        return {"status": "ok", "direction": req.direction, "positions": positions}
    except Exception as e:
        raise HTTPException(500, f"Servo aim failed: {e}")
    finally:
        if was_running and not state.animation_service._running.is_set():
            hold_pos = state.animation_service._current_state
            if hold_pos:
                state.animation_service._current_recording = "__aim_hold__"
                state.animation_service._current_actions = [hold_pos]
                state.animation_service._current_frame_index = 0
                state.animation_service._hold_until = time.time() + 5.0
            state.animation_service._running.set()
            state.animation_service._event_thread = threading.Thread(
                target=state.animation_service._event_loop, daemon=True
            )
            state.animation_service._event_thread.start()
            if not hold_pos:
                state.animation_service.dispatch(SERVO_CMD_PLAY, state.animation_service.idle_recording)


@router.post("/servo/nudge", response_model=ServoAimResponse)
def nudge_servo(req: ServoNudgeRequest):
    """Move servo by relative degrees from current position."""
    if not state.animation_service:
        raise HTTPException(503, "Servo not available")
    if not state.animation_service.robot:
        raise HTTPException(503, "Servo robot not connected")

    try:
        with state.animation_service.bus_lock:
            obs = state.animation_service.robot.get_observation()
        current = {k: v for k, v in obs.items() if k.endswith(".pos")}

        positions = dict(current)
        if req.yaw != 0:
            positions["base_yaw.pos"] = current.get("base_yaw.pos", 0) + req.yaw
        if req.pitch != 0:
            positions["base_pitch.pos"] = current.get("base_pitch.pos", 0) + req.pitch

        if req.duration > 0:
            state.animation_service.move_to(positions, duration=req.duration)
        else:
            with state.animation_service.bus_lock:
                state.animation_service.robot.send_action(positions)

        return {"status": "ok", "direction": f"nudge yaw={req.yaw} pitch={req.pitch}", "positions": positions}
    except Exception as e:
        raise HTTPException(500, f"Servo nudge failed: {e}")


@router.post("/servo/track", response_model=ServoTrackResponse)
def start_tracking(req: ServoTrackRequest):
    """Start tracking an object by bounding box. Servo follows the object in real-time."""
    if not state.tracker_service:
        raise HTTPException(503, "Tracker service not available")
    if not state.animation_service:
        raise HTTPException(503, "Servo not available")
    if not state.camera_capture:
        raise HTTPException(503, "Camera not available")

    bbox = tuple(req.bbox) if req.bbox else None
    ok = state.tracker_service.start(
        bbox=bbox,
        target_label=req.target,
        camera_capture=state.camera_capture,
        animation_service=state.animation_service,
    )
    if not ok:
        raise HTTPException(400, state.tracker_service.last_error or "Failed to initialize tracker")

    s = state.tracker_service.status
    return {
        "status": "ok",
        "tracking": True,
        "target": s.get("target"),
        "bbox": s.get("bbox"),
        "confidence": s.get("confidence"),
    }


@router.post("/servo/track/stop", response_model=ServoTrackResponse)
def stop_tracking():
    """Stop the current tracking session."""
    if not state.tracker_service:
        raise HTTPException(503, "Tracker service not available")

    state.tracker_service.stop()
    return {"status": "ok", "tracking": False}


@router.get("/servo/track", response_model=ServoTrackResponse)
def get_tracking_status():
    """Get current tracking status."""
    if not state.tracker_service:
        raise HTTPException(503, "Tracker service not available")

    s = state.tracker_service.status
    return {
        "status": "ok",
        "tracking": s["tracking"],
        "target": s["target"],
        "bbox": s["bbox"],
        "confidence": s.get("confidence"),
    }


@router.post("/servo/track/update", response_model=ServoTrackResponse)
def update_tracking_bbox(req: ServoTrackRequest):
    """Re-initialize tracker with a new bounding box."""
    if not state.tracker_service:
        raise HTTPException(503, "Tracker service not available")
    if not state.tracker_service.is_tracking:
        raise HTTPException(400, "No active tracking session")

    bbox = tuple(req.bbox)
    ok = state.tracker_service.update_bbox(bbox, camera_capture=state.camera_capture)
    if not ok:
        raise HTTPException(400, "Failed to re-initialize tracker")

    s = state.tracker_service.status
    return {
        "status": "ok",
        "tracking": True,
        "target": s.get("target"),
        "bbox": list(bbox),
    }
