"""Shared model state for protocol handlers.

Lifespan (server.py) calls setters during startup/shutdown.
Routers call getters to access the loaded models.

Thread-safety: these module globals are WRITTEN only by the lifespan hook
(single-threaded, before the server accepts traffic, and again at shutdown after
it stops). Request handlers only READ them. Because writes never race with reads,
no lock is needed; do NOT mutate these from within a request handler or that
invariant breaks. The perception objects themselves handle their own internal
concurrency (see PredictorBase locking).
"""

from core.perception.action.perception import ActionPerception
from core.perception.audio.predictors.base import AudioEmbedder
from core.perception.audio_emotion.perception import AudioEmotionPerception
from core.perception.facial_emotion.perception import EmotionPerception
from core.perception.object.perception import ObjectPerception
from core.perception.pose.perception import PosePerception

_action_model: ActionPerception | None = None
_emotion_model: EmotionPerception | None = None
_pose_model: PosePerception | None = None
_object_models: dict[str, ObjectPerception] = {}
_audio_embedder: AudioEmbedder | None = None


def get_action_model() -> ActionPerception | None:
    return _action_model


def set_action_model(model: ActionPerception | None) -> None:
    global _action_model
    _action_model = model


def get_emotion_model() -> EmotionPerception | None:
    return _emotion_model


def set_emotion_model(model: EmotionPerception | None) -> None:
    global _emotion_model
    _emotion_model = model


def get_pose_model() -> PosePerception | None:
    return _pose_model


def set_pose_model(model: PosePerception | None) -> None:
    global _pose_model
    _pose_model = model


def get_object_models() -> dict[str, ObjectPerception]:
    return _object_models


def get_object_model(name: str) -> ObjectPerception | None:
    return _object_models.get(name)


def set_object_models(models: dict[str, ObjectPerception]) -> None:
    global _object_models
    _object_models = models


def get_audio_embedder() -> AudioEmbedder | None:
    return _audio_embedder


def set_audio_embedder(embedder: AudioEmbedder | None) -> None:
    global _audio_embedder
    _audio_embedder = embedder


_audio_emotion_model: AudioEmotionPerception | None = None


def get_audio_emotion_model() -> AudioEmotionPerception | None:
    return _audio_emotion_model


def set_audio_emotion_model(model: AudioEmotionPerception | None) -> None:
    global _audio_emotion_model
    _audio_emotion_model = model
