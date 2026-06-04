"""Pure-function audio DSP helpers.

No class state, no I/O. Just transforms on numpy arrays. Easier to test and reuse.
"""

from math import gcd


def rms(audio_data, np) -> float:
    """RMS energy of a PCM frame (int16 → float32 internally)."""
    samples = audio_data.flatten().astype(np.float32)
    return float(np.sqrt(np.mean(samples ** 2)))


def resample_to_stt(data, device_rate: int, stt_rate: int, np) -> bytes:
    """Resample audio from device_rate to stt_rate using polyphase + anti-aliasing.

    Returns raw bytes at stt_rate. No-op (just .tobytes()) if rates already match.
    """
    if device_rate == stt_rate:
        return data.tobytes()
    import scipy.signal
    samples = data.flatten().astype(np.float32)
    g = gcd(stt_rate, device_rate)
    up, down = stt_rate // g, device_rate // g
    resampled = scipy.signal.resample_poly(samples, up, down).astype(np.int16)
    return resampled.tobytes()
