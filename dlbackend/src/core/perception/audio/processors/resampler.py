"""Resample audio to a target sample rate."""

import math

import numpy as np
from scipy.signal import resample_poly

from core.models.media import Audio

from .base import AudioProcessorBase


class Resampler(AudioProcessorBase):
    """Resample audio waveform to a target sample rate using polyphase filtering."""

    def __init__(self, target_sample_rate: int = 16000) -> None:
        super().__init__()
        self._target_sample_rate: int = target_sample_rate

    def process(self, input: Audio) -> Audio:
        if input.sample_rate == self._target_sample_rate:
            return input

        if input.waveform.shape[0] == 0:
            return Audio(waveform=input.waveform, sample_rate=self._target_sample_rate)

        gcd = math.gcd(self._target_sample_rate, input.sample_rate)
        up: int = self._target_sample_rate // gcd
        down: int = input.sample_rate // gcd

        resampled = resample_poly(input.waveform, up, down).astype(np.float32)
        return Audio(waveform=resampled, sample_rate=self._target_sample_rate)
