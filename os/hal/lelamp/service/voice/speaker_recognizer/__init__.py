"""Speaker voice recognition package."""

from .speaker_recognizer import (
    EmbeddingAPIUnavailableError,
    SpeakerRecognizer,
    SpeakerRecognizerError,
)

__all__ = [
    "EmbeddingAPIUnavailableError",
    "SpeakerRecognizer",
    "SpeakerRecognizerError",
]
