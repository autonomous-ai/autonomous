"""Factory for audio emotion recognizer models."""

from pathlib import Path

from core.enums import SpeechEmotionRecognizerEnum
from core.perception.audio_emotion.predictors.base import AudioEmotionRecognizer
from core.perception.base import PredictorFactory


class AudioEmotionRecognizerFactory(PredictorFactory[AudioEmotionRecognizer]):
    """Factory that creates AudioEmotionRecognizer instances from config."""

    def __init__(
        self,
        model_name: SpeechEmotionRecognizerEnum,
        model_path: Path | None = None,
    ) -> None:
        self._model_name: SpeechEmotionRecognizerEnum = model_name
        self._model_path: Path | None = model_path

    def create(self) -> AudioEmotionRecognizer:
        return create_audio_emotion_recognizer(self._model_name, self._model_path)


def create_audio_emotion_recognizer(
    model_name: SpeechEmotionRecognizerEnum,
    model_path: Path | None = None,
) -> AudioEmotionRecognizer:
    """Create the audio emotion recognizer for the given model type."""
    if model_name == SpeechEmotionRecognizerEnum.EMOTION2VEC:
        from core.perception.audio_emotion.predictors.emotion2vec import Emotion2VecPlusLargeRecognizer

        return Emotion2VecPlusLargeRecognizer(model_path=model_path)
    else:
        raise ValueError(f"Unknown audio emotion model: {model_name}")
