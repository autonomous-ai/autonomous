"""Audio processing utilities for endpoints."""

from __future__ import annotations

import base64
import io

import numpy as np
import soundfile as sf

from core.models.media import Audio


def decode_b64_wav(b64: str) -> Audio:
    """Decode a base64-encoded WAV into an Audio dataclass."""
    raw = base64.b64decode(b64)
    if not raw:
        raise ValueError("empty audio payload")
    waveform, sample_rate = sf.read(io.BytesIO(raw), dtype="float32")
    arr = np.asarray(waveform, dtype=np.float32)
    if arr.ndim == 2:
        arr = arr.mean(axis=1)
    elif arr.ndim != 1:
        raise ValueError("Uploaded wav must be mono/stereo waveform.")
    return Audio(waveform=arr, sample_rate=int(sample_rate))
