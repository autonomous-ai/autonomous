from .base import AudioEmbedder
from .campplus import CamPPlusEmbedder
from .ecapa_tdnn import EcapaTdnn1024Embedder
from .resnet34 import ResNet34Embedder

__all__ = [
    "AudioEmbedder",
    "CamPPlusEmbedder",
    "EcapaTdnn1024Embedder",
    "ResNet34Embedder",
]
