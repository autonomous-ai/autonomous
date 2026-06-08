# Tổng Quan Kiến Trúc — Autonomous

## Kiến Trúc 3 Tầng

```
OpenClaw (AI/LLM) → Lamp Server (Go, :5000) → HAL (Python, :5001) → Phần cứng
```

| Tầng | Ngôn ngữ | Port | Vai trò |
|------|----------|------|---------|
| OpenClaw | Go | WS | Bộ não AI, LLM, SKILL.md, memory, channels |
| Lamp Server | Go | 5000 | Hệ thống (mạng, OTA, MQTT, reset), sensing event routing, local intent |
| HAL | Python | 5001 | Hardware drivers (servo, LED, camera, audio, display), FastAPI |

## Thư Mục Dự Án

```
os/services/
├── cmd/lamp/main.go              — Entry point Lamp Server
├── cmd/bootstrap/main.go         — OTA bootstrap worker
├── server/
│   ├── server.go                 — Gin HTTP server, route setup
│   ├── config/                   — JSON config management
│   ├── health/delivery/http/     — Health, system info, dashboard
│   ├── network/delivery/http/    — WiFi scan, connect
│   ├── device/delivery/          — Setup (HTTP + MQTT handlers)
│   ├── sensing/delivery/http/    — Sensing event → intent match / OpenClaw
│   └── openclaw/delivery/sse/    — OpenClaw status, SSE events
├── internal/
│   ├── agent/                    — OpenClaw WebSocket gateway
│   ├── ambient/                  — Idle behaviors (breathing LED, micro-movements)
│   ├── beclient/                 — Backend status reporting
│   ├── device/                   — Device setup orchestration
│   ├── intent/                   — Local intent matching (voice commands)
│   ├── monitor/                  — Event bus (ring buffer 200 events)
│   ├── network/                  — WiFi AP/STA management
│   ├── openclaw/                 — OpenClaw config + SOUL.md
│   └── resetbutton/              — GPIO reset button
├── lib/mqtt/                     — MQTT client (Eclipse Paho autopaho)
├── domain/                       — Shared structs
├── bootstrap/                    — OTA worker
├── resources/openclaw-hooks/     — OpenClaw hook scripts
└── web/                          — React 19 + Vite + Tailwind CSS 4 SPA

os/hal/
├── server.py                     — FastAPI server (38 endpoints)
├── config.py                     — Hằng số runtime (ngưỡng sensing, timeout, URL)
├── devices/                      — Camera device abstraction (LocalVideoCaptureDevice)
├── service/
│   ├── voice/voice_service.py    — Local VAD + Deepgram STT, speaker ID, SER submit
│   ├── voice/speech_emotion/     — Queue SER → dlbackend → Lamp speech_emotion.detected
│   ├── voice/tts_service.py      — OpenAI-compatible TTS
│   ├── sensing/
│   │   ├── sensing_service.py    — Vòng lặp sensing nền
│   │   ├── presence_service.py   — State machine tự bật/tắt đèn theo presence
│   │   └── perceptions/          — Các detector có thể plug in
│   │       ├── motion.py         — Phát hiện chuyển động (frame diff)
│   │       ├── facerecognizer.py — Nhận diện friend/stranger (InsightFace)
│   │       └── light_level.py    — Phát hiện thay đổi độ sáng môi trường
│   └── display/                  — GC9A01 LCD eyes + info
└── pyproject.toml                — Python dependencies (opencv-python, insightface)

devices/                          — Per-device configs and overlays
skills/                           — SKILL.md files cho OpenClaw
companions/                       — Companion apps (e.g. Lamp Buddy)
contract/                         — Shared API contracts
cts/                              — Compatibility test suite
```

## Nguyên Tắc

- **Hardware là plugin** — cắm vào thì play, không cắm thì skip
- **Tầng hệ thống chạy KHÔNG cần OpenClaw** — thiết bị luôn phản hồi
- **Code là source of truth** — docs phản ánh code
- **HAL là hardware driver** — không chứa logic AI
- **SKILL.md native** — không dùng MCP, LLM tự đọc skill và gọi curl

## Voice Pipeline

```
Mic (always on) → Local VAD (RMS energy, free)
    → Speech detected → Connect Deepgram STT
        → "hey lamp, tắt đèn" → voice_command → local intent → thực thi
        → "anh ơi đi ăn không" → voice (ambient) → OpenClaw
    → Silence 3s → Disconnect Deepgram
    → _submit_speech_emotion_from_session: WAV → dlbackend SER → Lamp event (luôn chạy, độc lập transcript)
    → _identify_and_decorate (1 lần) → if transcript: _send_to_lamp voice/voice_command
```

Chi tiết SER: [speech-emotion_vi.md](speech-emotion_vi.md).

## Sensing Flow

```
HAL sensing loop (mỗi 2s) → Đọc 1 frame camera, chạy tất cả detectors:
    ├─ Motion detection (frame diff) → event nếu >8% pixel thay đổi
    ├─ Face recognition (InsightFace buffalo_sc) → phân loại friend/stranger
    │     → presence.enter (JPEG được annotate bbox: xanh=friend, đỏ=stranger)
    │     → presence.leave (3 tick liên tiếp không thấy mặt)
    ├─ Light level (mean brightness, mỗi 30s) → event nếu thay đổi >30/255
    └─ Sound detection (mic RMS) → event nếu > threshold

Event có ảnh? (large motion, face enter) → encode frame full-resolution JPEG q85
Ảnh face enter: frame gốc được vẽ bounding box + nhãn friend/stranger

POST /api/sensing/event {type, message, image?}
    → Lamp Go:
        1. Voice event + local intent match? → thực thi trực tiếp (~50ms)
        2. Không match → forward OpenClaw:
           - Có image → SendChatMessageWithImage (text + vision content block)
           - Không image → SendChatMessage (text only)
        3. OpenClaw AI nhìn ảnh + đọc context → quyết định hành động → gọi SKILL API
```

Cooldown bảo vệ chi phí LLM: motion/sound 60s, presence 10s, light.level 30s.
