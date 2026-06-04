"""Display route handlers -- /display/* endpoints."""

from fastapi import APIRouter, HTTPException
from fastapi.responses import Response

import lelamp.app_state as state
from lelamp.models import (
    DisplayEyesRequest,
    DisplayInfoRequest,
    DisplayStateResponse,
    StatusResponse,
)

router = APIRouter(tags=["Display"])


@router.get("/display", response_model=DisplayStateResponse)
def get_display_state():
    """Get current display state."""
    if not state.display_service:
        return {"mode": "unavailable", "hardware": False, "available_expressions": []}
    return state.display_service.get_state()


@router.post("/display/eyes", response_model=StatusResponse)
def set_display_eyes(req: DisplayEyesRequest):
    """Set eye expression on the round LCD display."""
    if not state.display_service:
        raise HTTPException(503, "Display not available")
    state.display_service.set_expression(req.expression, req.pupil_x, req.pupil_y)
    return {"status": "ok"}


@router.post("/display/info", response_model=StatusResponse)
def set_display_info(req: DisplayInfoRequest):
    """Switch display to info mode with text content."""
    if not state.display_service:
        raise HTTPException(503, "Display not available")
    state.display_service.set_info(req.text, req.subtitle)
    return {"status": "ok"}


@router.post("/display/eyes-mode", response_model=StatusResponse)
def switch_to_eyes_mode():
    """Switch display back to eyes mode (default)."""
    if not state.display_service:
        raise HTTPException(503, "Display not available")
    state.display_service.set_eyes_mode()
    return {"status": "ok"}


@router.get("/display/snapshot")
def display_snapshot():
    """Get current display frame as JPEG."""
    if not state.display_service:
        raise HTTPException(503, "Display not available")
    data = state.display_service.get_snapshot_bytes()
    if not data:
        raise HTTPException(404, "No frame rendered yet")
    return Response(content=data, media_type="image/jpeg")
