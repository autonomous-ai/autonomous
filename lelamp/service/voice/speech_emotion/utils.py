"""Pure helpers for the speech emotion pipeline.

Kept free of I/O and threading so they can be unit-tested without spinning
up the service or hitting dlbackend.
"""

from __future__ import annotations

from lelamp.service.voice.speech_emotion.constants import (
    CONFIDENCE_THRESHOLD_BY_LABEL,
    DEFAULT_CONFIDENCE_THRESHOLD,
    HEDGE_BY_BUCKET,
    LABEL_BUCKETS,
    NEUTRAL_LABELS,
    SpeechEmotionLabel,
)

def normalize_label(label: str) -> str:
    return (label or "").strip().lower()


def is_neutral(label: SpeechEmotionLabel | str) -> bool:
    return label in NEUTRAL_LABELS


def bucket_for(label: SpeechEmotionLabel | str) -> str:
    return LABEL_BUCKETS.get(label, "other")


def threshold_for(label: SpeechEmotionLabel | str) -> float:
    return CONFIDENCE_THRESHOLD_BY_LABEL.get(
        label, DEFAULT_CONFIDENCE_THRESHOLD,
    )


def hedge_for(bucket: str) -> str:
    return HEDGE_BY_BUCKET.get(bucket, "do not over-react")


def format_message(label: SpeechEmotionLabel, confidence: float, bucket: str) -> str:
    """Hedged sensing message — symmetric with face emotion processor.

    Skill parsers on Lamp extract the raw label via regex on the
    "Speech emotion detected: <Label>." prefix; everything inside the
    parentheses is human-readable hint for the agent.
    """
    nice = label.value.capitalize() or "Unknown"
    return (
        f"Speech emotion detected: {nice}. "
        f"(weak voice cue; confidence={confidence:.2f}; "
        f"bucket={bucket}; treat as uncertain, {hedge_for(bucket)}.)"
    )
