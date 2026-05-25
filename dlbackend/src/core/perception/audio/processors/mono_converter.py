"""Convert stereo/multichannel audio to mono."""

import numpy as np

from core.models.media import Audio

from .base import AudioProcessorBase


class MonoConverter(AudioProcessorBase):
    """Convert multichannel audio to mono by averaging channels."""

    def __init__(self) -> None:
        super().__init__()

    def process(self, input: Audio) -> Audio:
        if input.waveform.ndim == 1:
            return input
        if input.waveform.ndim == 2:
            return Audio(
                waveform=input.waveform.mean(axis=1).astype(np.float32),
                sample_rate=input.sample_rate,
            )
        return input
