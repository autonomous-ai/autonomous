"""Factory for audio embedder models."""

from pathlib import Path

from core.enums.audio import AudioEmbedderEnum
from core.perception.audio.predictors.base import AudioEmbedder
from core.perception.audio.processors.utils import AudioProcessorFactory
from core.perception.base import PredictorFactory


class AudioEmbedderFactory(PredictorFactory[AudioEmbedder]):
    """Factory that creates AudioEmbedder instances from config."""

    def __init__(
        self,
        model_name: AudioEmbedderEnum,
        model_path: Path | None = None,
        processor_factory: AudioProcessorFactory | None = None,
        window_frames: int | None = None,
        hop_frames: int | None = None,
        sample_rate: int | None = None,
        num_mel_bins: int | None = None,
        batch_size: int | None = None,
    ) -> None:
        self._model_name = model_name
        self._model_path = model_path
        self._processor_factory = processor_factory
        self._window_frames = window_frames
        self._hop_frames = hop_frames
        self._sample_rate = sample_rate
        self._num_mel_bins = num_mel_bins
        self._batch_size = batch_size

    def create(self) -> AudioEmbedder:
        return create_embedder(
            self._model_name,
            self._model_path,
            processor_factory=self._processor_factory,
            window_frames=self._window_frames,
            hop_frames=self._hop_frames,
            sample_rate=self._sample_rate,
            num_mel_bins=self._num_mel_bins,
            batch_size=self._batch_size,
        )


def create_embedder(
    model_name: AudioEmbedderEnum,
    model_path: Path | None,
    processor_factory: AudioProcessorFactory | None = None,
    window_frames: int | None = None,
    hop_frames: int | None = None,
    sample_rate: int | None = None,
    num_mel_bins: int | None = None,
    batch_size: int | None = None,
) -> AudioEmbedder:
    """Instantiate the correct audio embedder model."""
    if model_name == AudioEmbedderEnum.RESNET34:
        from core.perception.audio.predictors.resnet34 import ResNet34Embedder as cls
    elif model_name == AudioEmbedderEnum.ECAPA_TDNN_1024:
        from core.perception.audio.predictors.ecapa_tdnn import EcapaTdnn1024Embedder as cls
    elif model_name == AudioEmbedderEnum.CAMPPLUS:
        from core.perception.audio.predictors.campplus import CamPPlusEmbedder as cls
    else:
        raise ValueError(f"Unknown audio embedder model: {model_name}")

    return cls(
        model_path,
        processor_factory=processor_factory,
        window_frames=window_frames,
        hop_frames=hop_frames,
        sample_rate=sample_rate,
        num_mel_bins=num_mel_bins,
        batch_size=batch_size,
    )
