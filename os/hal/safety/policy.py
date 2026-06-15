"""Safety policy layer — read a device's SAFETY.md bounds and expose pure,
deterministic gate functions the HAL routes/drivers call before actuating.

Mirrors os/hal/board/device.py: a dependency-free regex front-matter parser (no
pyyaml in the runtime) and pure functions, fully unit-testable off-hardware. This
is the mechanism behind the first principle in contract/SAFETY-SPEC.md — *safety
is below the brain*: the gate sits in the request path between the agent and the
hardware, runs on every request regardless of who issued it, and cannot be
bypassed by prompting.

Slice 1 enforces `light.max_brightness` (an LED brightness ceiling that has no
prior enforcement). Later slices add quiet_hours / motion bounds to the same
SafetyPolicy + gate shape — the autonomous.safety.v1 ABI only ever gains fields.
See contract/SAFETY-SPEC.md and docs/safety.md.
"""
from __future__ import annotations

import logging
import os
import re
import urllib.request
from dataclasses import dataclass
from typing import Optional, Tuple

logger = logging.getLogger("hal.safety")

# The SAFETY.md `schema:` is an ABI tag (SAFETY-SPEC.md §Versioning), identical in
# discipline to autonomous.device.v1: within a major version fields are only added,
# so a v1 file must keep enforcing on every later v1 runtime. A file that declares
# a major this runtime does not understand cannot be parsed safely → fail loud.
SCHEMA_NAMESPACE = "autonomous.safety"
SUPPORTED_SCHEMA_MAJORS = frozenset({1})

_RE_SCHEMA = re.compile(r"^schema:\s*(\S+)\s*$", re.MULTILINE)
_RE_SCHEMA_VERSION = re.compile(r"^" + re.escape(SCHEMA_NAMESPACE) + r"\.v(\d+)$")

MAX_CHANNEL = 255  # 8-bit per-channel RGB ceiling


@dataclass(frozen=True)
class SafetyPolicy:
    schema: str
    # light brightness ceiling (0–255). None = no ceiling declared → pass-through
    # (light fail-safe: a calm LED is not a hazard; never invent a limit).
    max_brightness: Optional[int] = None


def extract_front_matter(text: str) -> str:
    """Return the YAML front-matter block (between the first two '---' fences)."""
    m = re.match(r"^---\s*\n(.*?)\n---\s*\n", text, re.DOTALL)
    return m.group(1) if m else ""


def validate_schema(front_matter: str) -> str:
    """Parse + validate the `schema:` ABI tag. Returns the raw schema string.

    Raises ValueError if missing/malformed/unknown-major — a deploy fault that
    must fail boot rather than enforce a bounds ABI the runtime cannot read.
    """
    m = _RE_SCHEMA.search(front_matter)
    if not m:
        raise ValueError(
            f"SAFETY.md front matter is missing 'schema:' "
            f"(expected '{SCHEMA_NAMESPACE}.v<major>')"
        )
    schema = m.group(1)
    v = _RE_SCHEMA_VERSION.match(schema)
    if not v:
        raise ValueError(
            f"SAFETY.md schema '{schema}' is not a valid '{SCHEMA_NAMESPACE}.v<major>' tag"
        )
    major = int(v.group(1))
    if major not in SUPPORTED_SCHEMA_MAJORS:
        raise ValueError(
            f"SAFETY.md schema '{schema}' has major v{major}; this runtime supports "
            f"majors {sorted(SUPPORTED_SCHEMA_MAJORS)}"
        )
    return schema


def _parse_max_brightness(front_matter: str) -> Optional[int]:
    """`light.max_brightness` as an int, or None if absent. The field name is
    unique in the v1 contract, so a direct match is unambiguous (block or flow
    style: `light:\\n  max_brightness: 180` or `light: { max_brightness: 180 }`)."""
    m = re.search(r"\bmax_brightness:\s*(\d+)", front_matter)
    if not m:
        return None
    val = int(m.group(1))
    if not (0 <= val <= MAX_CHANNEL):
        raise ValueError(
            f"SAFETY.md light.max_brightness {val} out of range 0–{MAX_CHANNEL}"
        )
    return val


def parse_safety(text: str) -> SafetyPolicy:
    """Parse SAFETY.md text (which HAS front matter) into a SafetyPolicy.
    Validates the schema fail-loud; raises on an out-of-range bound."""
    fm = extract_front_matter(text)
    schema = validate_schema(fm)
    return SafetyPolicy(schema=schema, max_brightness=_parse_max_brightness(fm))


# ── gate functions — pure, deterministic, the single enforcement point ───────────

def clamp_brightness(policy: Optional[SafetyPolicy], value: int) -> int:
    """Clamp a 0–255 brightness scalar to the policy ceiling. No policy / no
    ceiling → pass-through unchanged (light fail-safe)."""
    if policy is None or policy.max_brightness is None:
        return value
    return min(value, policy.max_brightness)


def clamp_color(
    policy: Optional[SafetyPolicy], color: Tuple[int, int, int]
) -> Tuple[int, int, int]:
    """Scale an (r,g,b) tuple so its brightest channel respects the ceiling,
    preserving hue (full white 255 with ceiling 180 → 180,180,180; pure red
    255,0,0 → 180,0,0). Pass-through when no ceiling or already within it."""
    if policy is None or policy.max_brightness is None:
        return color
    r, g, b = color
    peak = max(r, g, b)
    ceiling = policy.max_brightness
    if peak <= ceiling:
        return color
    scale = ceiling / peak
    return (round(r * scale), round(g * scale), round(b * scale))


# ── loader ───────────────────────────────────────────────────────────────────

def _read_ref(device_dir: str, ref: str) -> str:
    """Resolve a *_ref to text, mirroring device._read_ref: an http(s) URL is
    downloaded, anything else is read as a path relative to the device dir."""
    if ref.startswith("http://") or ref.startswith("https://"):
        with urllib.request.urlopen(ref, timeout=30) as r:  # noqa: S310 (device-trusted ref)
            return r.read().decode("utf-8")
    with open(os.path.join(device_dir, ref), "r") as f:
        return f.read()


def load_safety(device_dir: str, safety_ref: str) -> Optional[SafetyPolicy]:
    """Resolve `safety_ref` (path/URL) and parse the bounds, or None when there
    are no enforceable bounds (pass-through; light fail-safe):

      - no safety_ref                          → None
      - safety_ref set but file unreadable     → None + WARN (declared-but-absent)
      - SAFETY.md present but no front matter   → None + WARN (legacy prose-only)

    A SAFETY.md that *does* carry front matter must have a valid schema — a
    missing/malformed/unknown-major tag (or an out-of-range bound) raises and
    aborts boot, since the runtime will not enforce an ABI it cannot read
    (contract/SAFETY-SPEC.md).
    """
    if not safety_ref:
        return None
    try:
        text = _read_ref(device_dir, safety_ref)
    except Exception as e:
        logger.warning(
            "[safety] cannot read safety_ref %r: %s — bounds not enforced", safety_ref, e
        )
        return None
    if not extract_front_matter(text):
        logger.warning(
            "[safety] %s has no machine front matter — bounds not enforced (prose only)",
            safety_ref,
        )
        return None
    return parse_safety(text)  # validates schema fail-loud
