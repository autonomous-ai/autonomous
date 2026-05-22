"""HTTP endpoints for audio embedding service."""

from __future__ import annotations

from fastapi import APIRouter, HTTPException

from core.models.media import Audio
from core.perception.audio.predictors.base import AudioEmbedder
from dlserver.models.audio import (
    EmbedAudioRequest,
    EmbedAudioResponse,
)
from dlserver.utils.audio import decode_b64_wav
from dlserver.utils.state import get_audio_embedder

router = APIRouter(tags=["audio-recognizer"])


def _require_audio_embedder() -> AudioEmbedder:
    embedder = get_audio_embedder()
    if embedder is None:
        raise HTTPException(status_code=503, detail="Audio embedder is unavailable")
    return embedder


@router.post("/audio-recognizer/embed", response_model=EmbedAudioResponse)
async def embed_audio(req: EmbedAudioRequest):
    """Return per-chunk and/or aggregated L2-normalized embeddings.

    Stateless — does NOT touch the speaker DB.
    """
    embedder = _require_audio_embedder()
    try:
        audios: list[Audio] = []
        for item in req.audios_b64:
            try:
                audios.append(decode_b64_wav(item))
            except Exception as exc:
                raise ValueError(f"invalid base64 payload: {exc}") from exc

        if not audios:
            raise ValueError("no audio extracted from inputs")

        results = embedder.predict(audios)
        return EmbedAudioResponse.from_raw_embedding(results[0], return_chunks=req.return_chunks)
    except HTTPException:
        raise
    except Exception as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
