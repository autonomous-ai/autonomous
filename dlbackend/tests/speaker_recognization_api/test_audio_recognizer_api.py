"""Integration tests for audio recognizer HTTP APIs.

Requires:
- DL_BACKEND_URL in environment/.env
- Optional DL_API_KEY in environment/.env
- Local mock wav data under tests/mock_data/audio
"""

from __future__ import annotations

import base64
import os
import threading
import time
import uuid
from functools import partial
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import httpx
import numpy as np
import pytest
import soundfile as sf
from dotenv import load_dotenv

_ = load_dotenv()

DL_BACKEND_URL = os.getenv("DL_BACKEND_URL", "").rstrip("/")
DL_API_KEY = os.getenv("DL_API_KEY", "")

DATA_DIR = Path(__file__).parent.parent / "mock_data" / "audio" / "speakers"
BAO_1 = DATA_DIR / "Bao" / "Bao_1.wav"
BAO_2 = DATA_DIR / "Bao" / "Bao_2.wav"
KHANH_1 = DATA_DIR / "Khanh" / "Khanh_1.wav"
KHANH_2 = DATA_DIR / "Khanh" / "Khanh_2.wav"

pytestmark = pytest.mark.skipif(
    not DL_BACKEND_URL,
    reason="DL_BACKEND_URL not set - skipping API integration tests.",
)


def _headers() -> dict[str, str]:
    if DL_API_KEY:
        return {"X-API-Key": DL_API_KEY}
    return {}


def _url(path: str) -> str:
    return f"{DL_BACKEND_URL}{path}"


def _dump_response(label: str, resp: httpx.Response) -> None:
    print(
        f"[{label}] status={resp.status_code} "
        f"content-type={resp.headers.get('content-type', '')} "
        f"len={len(resp.text)}"
    )
    try:
        print(f"[{label}] body(json)={resp.json()}")
    except Exception:
        print(f"[{label}] body(text)={resp.text}")


def _timed_request(label: str, method: str, path: str, **kwargs) -> httpx.Response:
    start = time.perf_counter()
    resp = httpx.request(method, _url(path), **kwargs)
    elapsed_ms = (time.perf_counter() - start) * 1000.0
    print(f"[Timing] {label}: {elapsed_ms:.2f} ms")
    _dump_response(label, resp)
    return resp


def _audio_api_available() -> bool:
    try:
        resp = _timed_request(
            "preflight /audio-recognizer/speakers",
            "GET",
            "/api/dl/audio-recognizer/speakers",
            headers=_headers(),
            timeout=5.0,
        )
    except Exception:
        return False
    return resp.status_code != 404


def _mock_files_ready() -> bool:
    required = [BAO_1, BAO_2, KHANH_1, KHANH_2]
    return all(p.exists() for p in required)


def _wav_to_chunks(path: Path, chunk_seconds: float = 0.5) -> tuple[list[list[float]], int]:
    waveform, sr = sf.read(str(path), dtype="float32")
    arr = np.asarray(waveform, dtype=np.float32)
    if arr.ndim == 2:
        arr = arr.mean(axis=1)
    chunk_size = max(1, int(sr * chunk_seconds))
    chunks: list[list[float]] = []
    for start in range(0, len(arr), chunk_size):
        piece = arr[start : start + chunk_size]
        if piece.size > 0:
            chunks.append(piece.astype(np.float32).tolist())
    return chunks, int(sr)


def _wav_to_pcm16_b64(path: Path) -> tuple[str, int]:
    waveform, sr = sf.read(str(path), dtype="float32")
    arr = np.asarray(waveform, dtype=np.float32)
    if arr.ndim == 2:
        arr = arr.mean(axis=1)
    pcm16 = np.clip(arr, -1.0, 1.0)
    pcm16 = (pcm16 * 32767.0).astype(np.int16)
    return base64.b64encode(pcm16.tobytes()).decode("ascii"), int(sr)


def _wav_file_to_b64(path: Path) -> str:
    with open(path, "rb") as f:
        return base64.b64encode(f.read()).decode("ascii")


def _require_mock_wav(path: Path) -> None:
    if not path.exists():
        pytest.skip(f"Missing mock wav (add under tests/mock_data/audio): {path}")


def _post_register_multipart(speaker_name: str, wav_path: Path) -> httpx.Response:
    """Upload mock wav via multipart/form-data (field name `wav`)."""
    _require_mock_wav(wav_path)
    with open(wav_path, "rb") as f:
        return _timed_request(
            f"register multipart {wav_path.name}",
            "POST",
            "/api/dl/audio-recognizer/register",
            headers=_headers(),
            data={"name": speaker_name},
            files=[("wav", (wav_path.name, f, "audio/wav"))],
            timeout=120.0,
        )


def _post_recognize_multipart(wav_path: Path) -> httpx.Response:
    """Upload mock wav for recognition via multipart/form-data (field name `wav`)."""
    _require_mock_wav(wav_path)
    with open(wav_path, "rb") as f:
        return _timed_request(
            f"recognize multipart {wav_path.name}",
            "POST",
            "/api/dl/audio-recognizer/recognize",
            headers=_headers(),
            files=[("wav", (wav_path.name, f, "audio/wav"))],
            timeout=120.0,
        )


@pytest.fixture(scope="session")
def mock_http_server():
    """Serve local mock wav files via http://127.0.0.1:<port>/..."""
    if not _mock_files_ready():
        pytest.skip("Missing mock wav files under tests/mock_data/audio")

    handler = partial(SimpleHTTPRequestHandler, directory=str(DATA_DIR))
    server = ThreadingHTTPServer(("127.0.0.1", 0), handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    host, port = server.server_address
    base_url = f"http://{host}:{port}"
    try:
        yield base_url
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=3)


@pytest.fixture(autouse=True)
def ensure_audio_api_ready():
    if not _audio_api_available():
        pytest.skip("Audio recognizer API endpoints are not available at DL_BACKEND_URL.")


def test_register_with_wav_path_url(mock_http_server):
    print("==========================================test_register_with_wav_path_url==========================================")
    print("     Registering Bao_1.wav")
    payload = {
        "name": "Bao",
        "wav_path": f"{mock_http_server}/Bao/Bao_1.wav",
    }
    resp = _timed_request(
        "register wav_path URL",
        "POST",
        "/api/dl/audio-recognizer/register",
        headers=_headers(),
        json=payload,
    )
    assert resp.status_code == 200, resp.text
    body = resp.json()
    assert body["name"] == "Bao"
    assert body["num_samples"] > 0
    assert body["embedding_dim"] > 0


def test_register_with_wav_paths_url(mock_http_server):
    print("==========================================test_register_with_wav_paths_url==========================================")
    print("     Registering Bao_1.wav and Bao_2.wav")
    payload = {
        "name": "Bao",
        "wav_paths": [
            f"{mock_http_server}/Bao/Bao_1.wav",
            f"{mock_http_server}/Bao/Bao_2.wav",
        ],
    }
    resp = _timed_request(
        "register wav_paths URL list",
        "POST",
        "/api/dl/audio-recognizer/register",
        headers=_headers(),
        json=payload,
    )
    assert resp.status_code == 200, resp.text
    body = resp.json()
    assert body["name"] == "Bao"
    assert body["num_samples"] > 0


def test_register_with_chunks():
    print("==========================================test_register_with_chunks==========================================")
    print("     Registering Khanh_1.wav")
    chunks, sr = _wav_to_chunks(KHANH_1)
    payload = {
        "name": "Khanh",
        "chunks": chunks,
        "chunk_sample_rate": sr,
    }
    resp = _timed_request(
        "register chunks",
        "POST",
        "/api/dl/audio-recognizer/register",
        headers=_headers(),
        json=payload,
    )
    assert resp.status_code == 200, resp.text
    body = resp.json()
    assert body["name"] == "Khanh"


def test_register_with_pcm16_b64():
    print("==========================================test_register_with_pcm16_b64==========================================")
    print("     Registering Khanh_2.wav")
    pcm16_b64, sr = _wav_to_pcm16_b64(KHANH_2)
    payload = {
        "name": "Khanh",
        "pcm16_b64": pcm16_b64,
        "chunk_sample_rate": sr,
    }
    resp = _timed_request(
        "register pcm16_b64",
        "POST",
        "/api/dl/audio-recognizer/register",
        headers=_headers(),
        json=payload,
    )
    assert resp.status_code == 200, resp.text
    assert resp.json()["name"] == "Khanh"


def test_register_with_multipart_upload():
    resp = _post_register_multipart("Bao", BAO_1)
    assert resp.status_code == 200, resp.text
    assert resp.json()["name"] == "Bao"


def test_recognize_with_wav_path_url(mock_http_server):
    reg_payload = {
        "name": "Bao",
        "wav_path": f"{mock_http_server}/Bao/Bao_1.wav",
    }
    reg_resp = _timed_request(
        "recognize flow register wav_path URL",
        "POST",
        "/api/dl/audio-recognizer/register",
        headers=_headers(),
        json=reg_payload,
    )
    assert reg_resp.status_code == 200, reg_resp.text

    rec_payload = {"wav_path": f"{mock_http_server}/Bao/Bao_2.wav"}
    rec_resp = _timed_request(
        "recognize wav_path URL",
        "POST",
        "/api/dl/audio-recognizer/recognize",
        headers=_headers(),
        json=rec_payload,
    )
    assert rec_resp.status_code == 200, rec_resp.text
    body = rec_resp.json()
    assert "name" in body
    assert "confidence" in body


def test_recognize_with_chunks():
    print("==========================================test_recognize_with_chunks==========================================")
    print("     Registering Khanh_1.wav")
    reg_chunks, sr = _wav_to_chunks(KHANH_1)
    reg_resp = _timed_request(
        "recognize flow register chunks",
        "POST",
        "/api/dl/audio-recognizer/register",
        headers=_headers(),
        json={"name": "Khanh", "chunks": reg_chunks, "chunk_sample_rate": sr},
    )
    assert reg_resp.status_code == 200, reg_resp.text

    query_chunks, _ = _wav_to_chunks(KHANH_2)
    rec_resp = _timed_request(
        "recognize chunks",
        "POST",
        "/api/dl/audio-recognizer/recognize",
        headers=_headers(),
        json={"chunks": query_chunks, "chunk_sample_rate": sr},
    )
    assert rec_resp.status_code == 200, rec_resp.text
    assert "confidence" in rec_resp.json()


def test_recognize_with_pcm16_b64():
    print("==========================================test_recognize_with_pcm16_b64==========================================")
    print("     Registering Bao_1.wav")
    reg_pcm, sr = _wav_to_pcm16_b64(BAO_1)
    reg_resp = _timed_request(
        "recognize flow register pcm16_b64",
        "POST",
        "/api/dl/audio-recognizer/register",
        headers=_headers(),
        json={"name": "Bao", "pcm16_b64": reg_pcm, "chunk_sample_rate": sr},
    )
    assert reg_resp.status_code == 200, reg_resp.text

    query_pcm, _ = _wav_to_pcm16_b64(BAO_2)
    rec_resp = _timed_request(
        "recognize pcm16_b64",
        "POST",
        "/api/dl/audio-recognizer/recognize",
        headers=_headers(),
        json={"pcm16_b64": query_pcm, "chunk_sample_rate": sr},
    )
    assert rec_resp.status_code == 200, rec_resp.text
    assert "name" in rec_resp.json()


def test_recognize_with_multipart_upload():
    print("==========================================test_recognize_with_multipart_upload==========================================")
    print("     Registering Bao_1.wav")
    reg_resp = _post_register_multipart("Bao", BAO_1)
    assert reg_resp.status_code == 200, reg_resp.text

    rec_resp = _post_recognize_multipart(BAO_2)
    assert rec_resp.status_code == 200, rec_resp.text
    assert "confidence" in rec_resp.json()


def test_list_speakers_and_remove():
    print("==========================================test_list_speakers_and_remove==========================================")
    print("     Registering Bao_1.wav")
    chunks, sr = _wav_to_chunks(BAO_1)
    reg_resp = _timed_request(
        "list/remove flow register chunks",
        "POST",
        "/api/dl/audio-recognizer/register",
        headers=_headers(),
        json={"name": "Bao", "chunks": chunks, "chunk_sample_rate": sr},
    )
    assert reg_resp.status_code == 200, reg_resp.text

    list_resp = _timed_request(
        "list speakers",
        "GET",
        "/api/dl/audio-recognizer/speakers",
        headers=_headers(),
    )
    assert list_resp.status_code == 200, list_resp.text
    data = list_resp.json()
    assert "total" in data
    assert "speakers" in data
    assert any(item["name"] == "Bao" for item in data["speakers"])

    rm_resp = _timed_request(
        "remove speaker",
        "DELETE",
        f"/api/dl/audio-recognizer/speakers/Bao",
        headers=_headers(),
    )
    assert rm_resp.status_code == 200, rm_resp.text
    assert rm_resp.json()["removed"] is True


def test_embed_audio_from_wav_b64():
    print("==========================================test_embed_audio_from_wav_b64==========================================")
    audios_b64 = [_wav_file_to_b64(BAO_1), _wav_file_to_b64(BAO_2)]
    resp = _timed_request(
        "embed audio wav_b64 list",
        "POST",
        "/api/dl/audio-recognizer/embed",
        headers=_headers(),
        json={"audios_b64": audios_b64, "chunk_seconds": 0.5},
    )
    assert resp.status_code == 200, resp.text
    body = resp.json()
    assert "embedding" in body
    assert "embedding_dim" in body
    assert isinstance(body["embedding"], list)
    assert body["embedding_dim"] == len(body["embedding"])
    vec = np.asarray(body["embedding"], dtype=np.float32)
    assert vec.ndim == 1
    assert vec.shape[0] > 0
    assert np.isclose(np.linalg.norm(vec), 1.0, atol=1e-2)

