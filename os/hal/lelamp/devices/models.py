import os
from enum import StrEnum
from typing import Any, ClassVar

import numpy as np
import numpy.typing as npt
from pydantic import BaseModel, ConfigDict, field_serializer, model_validator


class DeviceModelType(StrEnum):
    VIDEO = "video"
    OTHER = "other"


class IDeviceInfo(BaseModel):
    model_type: DeviceModelType
    device_id: int | str
    device_name: str = ""

    @model_validator(mode="after")
    def validate_device_id(self):
        if self.device_name == "":
            self.device_name = f"{self.model_type}_{os.urandom(4).hex()}"

        return self


class IDeviceResponse(BaseModel):
    success: bool = True


class VideoCaptureDeviceInfo(IDeviceInfo):
    model_type: DeviceModelType = DeviceModelType.VIDEO
    fps: int | None = None
    max_width: int | None = None
    max_height: int | None = None
    rotate: float | None = None


class VideoCaptureDeviceResponse(IDeviceResponse):
    frame: npt.NDArray[np.uint8] | None = None
    frame_description: str | None = None
    frame_objects: list[dict[str, Any]] | None = None

    @field_serializer("frame")
    def serialize_frame(self, frame: npt.NDArray[np.uint8]):
        self.model_config
        return frame.tolist()

    model_config: ClassVar[ConfigDict] = ConfigDict(arbitrary_types_allowed=True)
