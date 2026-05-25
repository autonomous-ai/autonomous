"""Emotion2Vec+ Large audio emotion recognizer."""

from pathlib import Path

from core.perception.audio_emotion.constants import RESOURCES_DIR
from core.perception.audio_emotion.predictors.base import AudioEmotionRecognizer


class Emotion2VecPlusLargeRecognizer(AudioEmotionRecognizer):
    """Emotion2Vec+ Large — 9 emotion classes, 16kHz raw waveform input."""

    DEFAULT_MODEL_PATH: Path | None = RESOURCES_DIR / "emotion2vec" / "emotion2vec.onnx"
    DEFAULT_LABELS_PATH: Path | None = RESOURCES_DIR / "emotion2vec" / "labels.txt"
    DEFAULT_SAMPLE_RATE: int = 16000
    DEFAULT_INPUT_NAME: str = "input"
    DEFAULT_OUTPUT_NAME: str = "logits"
