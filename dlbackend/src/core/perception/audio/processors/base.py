"""Audio processor base class."""

from abc import abstractmethod

from core.models.media import Audio
from core.perception.base.processor import InputProcessorBase


class AudioProcessorBase(InputProcessorBase[Audio, Audio]):
    """Base for audio processors. Input and output are both Audio.

    Default lifecycle is no-op (ready immediately). Subclasses that load
    resources (e.g. VAD model) override _start_impl/_stop_impl/_is_ready_impl.
    """

    def __init__(self) -> None:
        super().__init__()
        self._running: bool = False

    def _start_impl(self) -> None:
        self._running = True
        self._logger.info("Processor started")

    def _stop_impl(self) -> None:
        self._running = False
        self._logger.info("Processor stopped")

    def _is_ready_impl(self) -> bool:
        return self._running

    @abstractmethod
    def process(self, input: Audio) -> Audio:
        """Process audio, return processed audio."""
