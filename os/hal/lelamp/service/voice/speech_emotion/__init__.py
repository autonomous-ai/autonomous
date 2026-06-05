"""Speech emotion recognition (SER) service.

Public surface:
    SpeechEmotionService — voice_service calls `submit(user, wav, dur)` per
    utterance; this service buffers per-user, dedups by polarity bucket,
    and POSTs sensing events to Lamp. Mirrors the face emotion processor's
    clustering/dedup architecture.

Engine layer:
    BaseSpeechEmotionRecognizer — ABC
    Emotion2VecRecognizer       — concrete, talks to dlbackend /api/dl/ser

All env-overridable defaults live in `lelamp.config.SPEECH_EMOTION_*`;
label vocabulary and bucket map live in `constants.py`.
"""

from lelamp.service.voice.speech_emotion.base import (
    BaseSpeechEmotionRecognizer,
    SpeechEmotionResult,
)
from lelamp.service.voice.speech_emotion.emotion2vec import Emotion2VecRecognizer
from lelamp.service.voice.speech_emotion.service import SpeechEmotionService

__all__ = [
    "BaseSpeechEmotionRecognizer",
    "Emotion2VecRecognizer",
    "SpeechEmotionResult",
    "SpeechEmotionService",
]
