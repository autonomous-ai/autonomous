"""HTTP models for audio emotion endpoints — Pydantic request/response types."""

from __future__ import annotations

from pydantic import BaseModel, Field

from core.models.audio_emotion import AudioEmotionDetection


class RecognizeEmotionRequest(BaseModel):
    """JSON body for ``POST /ser/recognize``."""

    audio_b64: str = Field(min_length=1)
    return_scores: bool = True


class RecognizeEmotionResponse(BaseModel):
    """HTTP response for audio emotion recognition."""

    label: str
    confidence: float
    scores: dict[str, float] | None = None

    @staticmethod
    def from_detection(
        detection: AudioEmotionDetection,
        return_scores: bool = True,
    ) -> "RecognizeEmotionResponse":
        if not detection.emotions:
            return RecognizeEmotionResponse(label="unknown", confidence=0.0, scores=None)

        top = detection.emotions[0]
        scores: dict[str, float] | None = None
        if return_scores:
            scores = {e.emotion: e.confidence for e in detection.emotions}

        return RecognizeEmotionResponse(
            label=top.emotion,
            confidence=top.confidence,
            scores=scores,
        )


class LabelsResponse(BaseModel):
    engine: str
    labels: list[str]
