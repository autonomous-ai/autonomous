"""Queue event models for the voice agent send/receive loops.

Send queue: AgentInputEvent subclasses.
Receive queue: AgentOutputEvent subclasses.
"""

from pydantic import BaseModel, ConfigDict

from hal.drivers.realtime.enums import InputEventTypeEnum, OutputEventTypeEnum
from hal.drivers.realtime.models.input import InputBase
from hal.drivers.realtime.models.output import OutputBase

# --- Send queue events ---


class AgentInputEvent(BaseModel):
    """Base for all send queue events."""

    model_config = ConfigDict(arbitrary_types_allowed=True)
    type: InputEventTypeEnum


class InputEvent(AgentInputEvent):
    """Send queue: a model input (audio, text, image, function result)."""

    type: InputEventTypeEnum = InputEventTypeEnum.INPUT
    input: InputBase


class AudioCommitEvent(AgentInputEvent):
    """Send queue: commit buffered audio (end of speech turn)."""

    type: InputEventTypeEnum = InputEventTypeEnum.AUDIO_COMMIT


# --- Receive queue events ---


class AgentOutputEvent(BaseModel):
    """Base for all receive queue events."""

    model_config = ConfigDict(arbitrary_types_allowed=True)
    type: OutputEventTypeEnum


class OutputEvent(AgentOutputEvent):
    """Receive queue: a model output (audio, text, function call)."""

    type: OutputEventTypeEnum = OutputEventTypeEnum.OUTPUT
    output: OutputBase


class TurnDoneEvent(AgentOutputEvent):
    """Receive queue: model finished its turn."""

    type: OutputEventTypeEnum = OutputEventTypeEnum.TURN_DONE
