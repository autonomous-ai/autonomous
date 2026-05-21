"""Defaults & label vocabulary for speech emotion recognition.

Bucketing mirrors the face emotion processor so downstream skills/dedup
share the same polarity language. emotion2vec_plus_large labels (from
dlbackend `/api/dl/ser/labels`) are enumerated by `SpeechEmotionLabel`
below.
"""

from __future__ import annotations

from enum import Enum

# --- API contract ---------------------------------------------------------

DEFAULT_DL_SER_ENDPOINT: str = "/lelamp/api/dl/ser/recognize"
DEFAULT_API_TIMEOUT_S: float = 15.0

# --- Label vocabulary ----------------------------------------------------
class SpeechEmotionLabel(str, Enum):
    ANGRY = "angry"
    DISGUSTED = "disgusted"
    FEARFUL = "fearful"
    HAPPY = "happy"
    NEUTRAL = "neutral"
    OTHER = "other"
    SAD = "sad"
    SURPRISED = "surprised"
    UNK = "<unk>"
    
    @classmethod
    def _missing_(cls, value: str) -> SpeechEmotionLabel:
        return cls.UNK

# --- Input gating ---------------------------------------------------------

DEFAULT_MIN_AUDIO_S: float = 3.0

CONFIDENCE_THRESHOLD_BY_LABEL: dict[str, float] = {
    SpeechEmotionLabel.HAPPY:     0.5,
    SpeechEmotionLabel.SURPRISED: 0.6,
    SpeechEmotionLabel.SAD:       0.7,
    SpeechEmotionLabel.ANGRY:     0.6,
    SpeechEmotionLabel.FEARFUL:   0.6,
    SpeechEmotionLabel.DISGUSTED: 0.6,
}
DEFAULT_CONFIDENCE_THRESHOLD: float = 0.5

# --- Buffering / dedup ---------------------------------------------------

DEFAULT_FLUSH_S: float = 10.0
DEFAULT_DEDUP_WINDOW_S: float = 300.0
DEFAULT_QUEUE_MAXSIZE: int = 32

# --- Polarity buckets -----------------------------------------------------
# Matches face emotion processor's EMOTION_BUCKETS shape so (user, bucket)
# dedup keys are interpretable across modalities.

LABEL_BUCKETS: dict[str, str] = {
    SpeechEmotionLabel.HAPPY:     "positive",
    SpeechEmotionLabel.SURPRISED: "positive",
    SpeechEmotionLabel.ANGRY:     "negative",
    SpeechEmotionLabel.DISGUSTED: "negative",
    SpeechEmotionLabel.FEARFUL:   "negative",
    SpeechEmotionLabel.SAD:       "negative",
    # Anything not in the map collapses to "other" via utils.bucket_for().
}

# Bare "unk" and "" stay as raw-string defensive fallbacks so is_neutral()
# still answers True if a caller bypasses the SpeechEmotionLabel coercion
# in _process_job and hands us a raw model string.
NEUTRAL_LABELS: frozenset = frozenset(
    {SpeechEmotionLabel.NEUTRAL, SpeechEmotionLabel.OTHER, SpeechEmotionLabel.UNK,
     "unk", ""}
)

HEDGE_BY_BUCKET: dict[str, str] = {
    "positive": "do not over-celebrate",
    "negative": "do not assume the user is distressed",
    "other": "do not over-react",
}

# --- Wire format ----------------------------------------------------------

SENSING_EVENT_TYPE: str = "speech_emotion.detected"
UNKNOWN_USER_LABEL: str = "unknown"
