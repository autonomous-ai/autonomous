#!/usr/bin/env python

# Copyright 2025 The HuggingFace Inc. team. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from dataclasses import dataclass, field
from pathlib import Path

from lerobot.cameras import CameraConfig

from lerobot.robots import RobotConfig

# Repo-local, version-controlled calibration dir (os/hal/calibration/robots/hal_follower)
# instead of lerobot's per-user default (~/.cache/huggingface/lerobot/calibration), which
# breaks when the service user differs (e.g. hal.service runs as root, not orangepi).
# lerobot loads `calibration_dir / f"{id}.json"` (id = HAL_DEVICE_ID, default "hal").
CALIBRATION_DIR = Path(__file__).resolve().parent.parent / "calibration" / "robots" / "hal_follower"


@RobotConfig.register_subclass("hal_follower")
@dataclass
class LeLampFollowerConfig(RobotConfig):
    # Port to connect to the arm
    port: str

    disable_torque_on_disconnect: bool = True

    # `max_relative_target` limits the magnitude of the relative positional target vector for safety purposes.
    # Set this to a positive scalar to have the same value for all motors, or a list that is the same length as
    # the number of motors in your follower arms.
    max_relative_target: int | None = None

    # cameras
    cameras: dict[str, CameraConfig] = field(default_factory=dict)

    # Set to `True` for backward compatibility with previous policies/dataset
    use_degrees: bool = False

    def __post_init__(self):
        super().__post_init__()
        # Pin calibration to the repo-local dir unless a caller overrides it.
        if self.calibration_dir is None:
            self.calibration_dir = CALIBRATION_DIR
