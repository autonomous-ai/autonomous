from enum import StrEnum


class AudioEmbedderEnum(StrEnum):
    RESNET34 = "resnet34"
    ECAPA_TDNN_1024 = "ecapa-tdnn1024"
    CAMPPLUS = "campplus"
