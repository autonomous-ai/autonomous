"""Input data models sent to the realtime voice agent."""


import cv2.typing as cv2t
import numpy as np
import numpy.typing as npt
from pydantic import BaseModel, ConfigDict

from hal.drivers.realtime.enums import InputTypeEnum


class InputBase(BaseModel):
    model_config = ConfigDict(frozen=True, arbitrary_types_allowed=True)
    type: InputTypeEnum


class TextInput(InputBase):
    type: InputTypeEnum = InputTypeEnum.TEXT
    text: str


class AudioInput(InputBase):
    type: InputTypeEnum = InputTypeEnum.AUDIO
    audio: npt.NDArray[np.float32]


class ImageInput(InputBase):
    type: InputTypeEnum = InputTypeEnum.IMAGE
    image: cv2t.MatLike


class FunctionCallResultInput(InputBase):
    type: InputTypeEnum = InputTypeEnum.FUNCTION_CALL_RESULT
    call_id: str
    output: str  # JSON string
