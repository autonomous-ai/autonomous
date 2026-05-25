"""POSTER V2 emotion predictor (7-class, RAF-DB).

Pure emotion classification from a face crop — no face detection.
Input: 224x224 RGB face crop, ImageNet-normalised.

Uses the default _postprocess_batch from base (single output: logits).
"""

from pathlib import Path

import numpy as np
import numpy.typing as npt

from core.perception.facial_emotion.constants import RESOURCES_DIR
from core.perception.facial_emotion.predictors.base import EmotionRecognizer


class PosterV2Recognizer(EmotionRecognizer):
    """POSTER V2 ONNX emotion predictor."""

    DEFAULT_MODEL_PATH: Path | None = RESOURCES_DIR / "posterv2_7cls.onnx"
    DEFAULT_CLASSES_PATH: Path = RESOURCES_DIR / "posterv2_classes.txt"
    DEFAULT_INPUT_SIZE: tuple[int, int] = (224, 224)

    MEAN: npt.NDArray[np.float32] = np.array([0.485, 0.456, 0.406], dtype=np.float32)
    STD: npt.NDArray[np.float32] = np.array([0.229, 0.224, 0.225], dtype=np.float32)
