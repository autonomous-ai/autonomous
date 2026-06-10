"""OpenAI Realtime-specific enumerations."""

from enum import StrEnum


class OpenAITurnDetectionType(StrEnum):
    SERVER_VAD = "server_vad"
    SEMANTIC_VAD = "semantic_vad"


class OpenAIReasoningEffort(StrEnum):
    MINIMAL = "minimal"
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"
    XHIGH = "xhigh"


class OpenAITruncationType(StrEnum):
    AUTO = "auto"
    DISABLED = "disabled"
    RETENTION_RATIO = "retention_ratio"


class OpenAIVoice(StrEnum):
    ALLOY = "alloy"
    ASH = "ash"
    BALLAD = "ballad"
    CORAL = "coral"
    ECHO = "echo"
    SAGE = "sage"
    SHIMMER = "shimmer"
    VERSE = "verse"
    MARIN = "marin"
    CEDAR = "cedar"
