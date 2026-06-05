"""
One-shot test of the speech-emotion engine against a hosted dlbackend.

Records a fixed-length clip from the default mic, sends it to
`/api/dl/ser/recognize`, prints `{label, confidence}`. No buffering,
no Lamp POST — just verifies the engine + network path.

Usage (from repo root):

    # Point at your hosted dlbackend
    export DL_BACKEND_URL="https://<host>"            # no trailing slash
    export DL_API_KEY="<your shared secret>"          # if your backend requires it

    # Record 5 seconds and classify
    python -m hal.test.test_speech_emotion_engine

    # Tweak
    python -m hal.test.test_speech_emotion_engine \\
        --dl-backend-url "$DL_BACKEND_URL" \\
        --api-key "$DL_API_KEY" \\
        --duration 6 \\
        --device 1            # sounddevice input index (see `python -m sounddevice`)

The script exits non-zero on transport / parse failure so it's safe to
chain in a smoke-test pipeline.
"""

from __future__ import annotations

import argparse
import io
import logging
import os
import sys
import wave
from dotenv import load_dotenv

load_dotenv()

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("test.speech_emotion_engine")

SAMPLE_RATE = 16000
CHANNELS = 1
DEFAULT_DURATION_S = 5.0
# Hit FastAPI directly — the `/hal/` prefix only exists when dlbackend
# is fronted by the production nginx config (RunPod) that strips it before
# forwarding. Local dev runs uvicorn on its native port, no prefix.
DEFAULT_ENDPOINT = "/api/dl/ser/recognize"


def record_wav_bytes(duration_s: float, device: int | None) -> bytes:
    """Record `duration_s` of mono int16 audio @ 16 kHz, return WAV bytes.

    Uses sounddevice (PortAudio). Pass `device` as a sounddevice index
    when the OS default mic isn't right (run `python -m sounddevice` to list).
    """
    import numpy as np
    import sounddevice as sd

    frames = int(duration_s * SAMPLE_RATE)
    logger.info(
        "Recording %.1fs @ %d Hz (device=%s) — speak now...",
        duration_s, SAMPLE_RATE, device if device is not None else "default",
    )
    pcm = sd.rec(
        frames, samplerate=SAMPLE_RATE, channels=CHANNELS,
        dtype=np.int16, device=device,
    )
    sd.wait()
    logger.info("Recording done — %d samples captured", len(pcm))

    buf = io.BytesIO()
    with wave.open(buf, "wb") as wf:
        wf.setnchannels(CHANNELS)
        wf.setsampwidth(2)  # int16
        wf.setframerate(SAMPLE_RATE)
        wf.writeframes(pcm.tobytes())
    return buf.getvalue()


def main() -> int:
    parser = argparse.ArgumentParser(description="Test speech emotion engine")
    parser.add_argument(
        "--dl-backend-url",
        default=os.environ.get("DL_BACKEND_URL", "http://localhost:8008"),
        help="Base URL of the hosted dlbackend (no trailing slash).",
    )
    parser.add_argument(
        "--api-key",
        default=os.environ.get("DL_API_KEY", "")
    )
    parser.add_argument(
        "--endpoint", default=os.environ.get("DL_SER_ENDPOINT", "") or DEFAULT_ENDPOINT,
        help=f"SER endpoint path (default {DEFAULT_ENDPOINT}).",
    )
    parser.add_argument(
        "--duration", type=float, default=DEFAULT_DURATION_S,
        help="Seconds to record from the mic (default 5).",
    )
    parser.add_argument(
        "--device", type=int, default=None,
        help="sounddevice input index. Omit for system default.",
    )
    args = parser.parse_args()

    if not args.dl_backend_url:
        print(
            "ERROR: --dl-backend-url is required (or set DL_BACKEND_URL).",
            file=sys.stderr,
        )
        return 2

    # Compose URL once so logs show exactly what we're hitting.
    url = args.dl_backend_url.rstrip("/") + "/" + args.endpoint.strip("/")
    logger.info("Target: %s (api_key=%s)", url, "set" if args.api_key else "none")

    try:
        wav_bytes = record_wav_bytes(args.duration, args.device)
    except Exception as e:
        logger.exception("Recording failed: %s", e)
        return 3
    logger.info("WAV bytes: %d", len(wav_bytes))

    # Lazy import so missing requests doesn't break --help.
    from hal.drivers.voice.speech_emotion import Emotion2VecRecognizer

    rec = Emotion2VecRecognizer(url=url, api_key=args.api_key)
    if not rec.available:
        logger.error("Recognizer unavailable (empty URL?)")
        return 4

    result = rec.recognize(wav_bytes)
    if result is None:
        logger.error("Recognize returned None — see [speech_emotion.engine] logs above")
        return 5

    print()
    print(f"  label      = {result.label}")
    print(f"  confidence = {result.confidence:.4f}")
    print()
    return 0


if __name__ == "__main__":
    sys.exit(main())
