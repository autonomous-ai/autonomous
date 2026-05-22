"""CAM++ speaker embedding model."""

from pathlib import Path

from core.perception.audio.constants import MODELS_DIR
from core.perception.audio.predictors.base import AudioEmbedder


class CamPPlusEmbedder(AudioEmbedder):
    """WeSpeaker CAM++ speaker embedder."""

    DEFAULT_MODEL_PATH: Path | None = MODELS_DIR / "wespeaker_campplus.onnx"
