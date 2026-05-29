"""Health check endpoint."""

from fastapi import APIRouter

from dlserver.utils.state import (
    get_action_model,
    get_audio_embedder,
    get_audio_emotion_model,
    get_emotion_model,
    get_object_models,
    get_pose_model,
)

router = APIRouter()


@router.get("/health")
async def health():
    """Health check endpoint — reports readiness of all loaded models."""
    action_model = get_action_model()
    emotion_model = get_emotion_model()
    pose_model = get_pose_model()
    audio_embedder = get_audio_embedder()
    audio_emotion_model = get_audio_emotion_model()
    object_models = get_object_models()

    return {
        "status": "ok",
        "models": {
            "action": action_model is not None and action_model.is_ready(),
            "emotion": emotion_model is not None and emotion_model.is_ready(),
            "ser": audio_emotion_model is not None and audio_emotion_model.is_ready(),
            "pose": pose_model is not None and pose_model.is_ready(),
            "audio_embedder": audio_embedder is not None and audio_embedder.is_ready(),
            "object_detectors": {
                name: model.is_ready() for name, model in object_models.items()
            },
        },
    }
