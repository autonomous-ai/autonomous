"""ECAPA-TDNN 1024 speaker embedding model."""

from pathlib import Path

from core.perception.audio.constants import MODELS_DIR
from core.perception.audio.predictors.base import AudioEmbedder


class EcapaTdnn1024Embedder(AudioEmbedder):
    """WeSpeaker ECAPA-TDNN-1024-LM speaker embedder."""

    DEFAULT_MODEL_PATH: Path | None = MODELS_DIR / "wespeaker_ecapa_tdnn1024.onnx"
