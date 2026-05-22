"""Internal audio models — dataclasses for core logic, not HTTP."""

from dataclasses import dataclass

import numpy as np
import numpy.typing as npt


@dataclass
class RawAudioEmbedding:
    """Raw audio embedder output."""

    embedding: npt.NDArray[np.float32]
    """Shape: (D,) — aggregated L2-normalized embedding."""

    chunk_embeddings: npt.NDArray[np.float32]
    """Shape: (N, D) — per-window embeddings before aggregation."""
