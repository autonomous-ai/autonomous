"""System route handlers -- /system/reboot, /system/shutdown."""

import subprocess

from fastapi import APIRouter

from lelamp.models import StatusResponse

router = APIRouter(tags=["System"])


@router.post("/system/reboot", response_model=StatusResponse)
def reboot_os():
    """Reboot the operating system."""
    subprocess.Popen(["sudo", "reboot"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    return {"status": "rebooting"}


@router.post("/system/shutdown", response_model=StatusResponse)
def shutdown_os():
    """Shutdown the operating system."""
    subprocess.Popen(
        ["sudo", "shutdown", "-h", "now"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL
    )
    return {"status": "shutting down"}
