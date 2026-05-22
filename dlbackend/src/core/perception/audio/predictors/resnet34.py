"""ResNet34 speaker embedding model."""

from pathlib import Path

from core.perception.audio.constants import MODELS_DIR
from core.perception.audio.predictors.base import AudioEmbedder


class ResNet34Embedder(AudioEmbedder):
    """WeSpeaker ResNet34-LM speaker embedder (256-dim)."""

    DEFAULT_MODEL_PATH: Path | None = MODELS_DIR / "wespeaker_resnet34.onnx"
