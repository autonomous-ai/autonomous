"""ALSA arecord-backed input stream.

Drop-in replacement for sounddevice.InputStream that uses arecord directly.
Needed on devices where PortAudio's hw: interface bypasses ALSA's sample-rate
conversion and produces corrupted audio.
"""

import subprocess


class ArecordStream:
    """Drop-in replacement for sd.InputStream using arecord subprocess.

    Records directly via ALSA plughw which handles sample-rate conversion
    natively — the same path as `arecord -D plughw:X,0`. sounddevice uses
    PortAudio's hw: interface which bypasses ALSA SRC, producing corrupted
    audio at rates the hardware doesn't natively support.
    """

    def __init__(self, alsa_device: str, rate: int, channels: int, blocksize: int, np):
        self._device = alsa_device
        self._rate = rate
        self._channels = channels
        self._blocksize = blocksize
        self._np = np
        self._proc = None
        self._bytes_per_frame = 2 * channels  # int16 = 2 bytes

    def __enter__(self):
        self._proc = subprocess.Popen(
            ["arecord", "-D", self._device, "-f", "S16_LE",
             "-r", str(self._rate), "-c", str(self._channels),
             "-t", "raw", "-q"],
            stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
        )
        return self

    def __exit__(self, *args):
        if self._proc:
            self._proc.terminate()
            try:
                self._proc.wait(timeout=2)
            except Exception:
                self._proc.kill()
            self._proc = None

    def read(self, frames):
        n_bytes = frames * self._bytes_per_frame
        raw = self._proc.stdout.read(n_bytes)
        if not raw:
            # arecord process died — raise so the main loop can restart it
            raise IOError("arecord process exited (stdout EOF)")
        if len(raw) < n_bytes:
            raw = raw + b"\x00" * (n_bytes - len(raw))
        data = self._np.frombuffer(raw, dtype=self._np.int16).reshape(frames, self._channels)
        return data, False
