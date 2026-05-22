from dataclasses import dataclass
from typing import NamedTuple

import cv2.typing as cv2t
import numpy as np
import numpy.typing as npt

from core.enums import GraphEnum


@dataclass
class Audio:
    waveform: npt.NDArray[np.float32]
    """Shape: (T,) — mono float32 waveform."""

    sample_rate: int


@dataclass
class Video:
    frames: list[cv2t.MatLike]
    fps: float | None = None


@dataclass
class Point2D(NamedTuple):
    x: float
    y: float


@dataclass
class Point3D(NamedTuple):
    x: float
    y: float
    z: float


@dataclass
class Pose2D:
    graph_type: GraphEnum
    joints: list[Point2D]
    confs: list[float]


@dataclass
class Pose3D:
    graph_type: GraphEnum
    joints: list[Point3D]
    confs: list[float]
