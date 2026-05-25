"""Internal audio emotion models — dataclasses for core logic, not HTTP."""

from dataclasses import dataclass, field

import numpy as np
import numpy.typing as npt


@dataclass
class RawAudioEmotionDetection:
    """Raw recognizer output for a single audio utterance."""

    expression_probs: npt.NDArray[np.float32]
    """Shape: (C,) — softmaxed expression probabilities."""


@dataclass
class AudioEmotion:
    """Single classified emotion for one utterance."""

    emotion: str
    confidence: float


@dataclass
class AudioEmotionDetection:
    """Session output: filtered emotions for a single utterance."""

    emotions: list[AudioEmotion] = field(default_factory=list)


@dataclass
class AudioEmotionPerceptionSessionConfig:
    confidence_threshold: float = 0.0
