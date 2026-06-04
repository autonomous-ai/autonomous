"""Abstract speech emotion recognizer.

One inference per utterance — given a mono 16kHz WAV blob, return the top
label + confidence. Concrete engines (see `emotion2vec.py`) talk to
dlbackend; in-process engines could be added the same way as the face
emotion recognizer registry on dlbackend.
"""

from __future__ import annotations

import abc
from dataclasses import dataclass


@dataclass(slots=True)
class SpeechEmotionResult:
    """One classifier output for one utterance."""

    label: str
    confidence: float


class BaseSpeechEmotionRecognizer(abc.ABC):
    """Engine interface — stateless, one call per utterance."""

    @property
    @abc.abstractmethod
    def available(self) -> bool:
        """False if the engine is not configured (missing URL, key, …)."""

    @abc.abstractmethod
    def recognize(self, wav_bytes: bytes) -> SpeechEmotionResult | None:
        """Classify one utterance. Return None on transport/parse failure."""
