# Vision Tracking — Theo dõi vật thể bằng servo

Lamp có thể theo dõi và hướng theo bất kỳ vật thể nào mà người dùng gọi tên. Một detector tìm vật thể theo tên và seed một ViT tracker, sau đó một vòng lặp vision tốc độ cao bám theo nó real-time, trong khi một servo worker tách rời điều khiển đầu trượt mượt về phía target.

Toàn bộ code tracking nằm ở `os/hal/drivers/tracking/tracker_service.py`.

## Kiến trúc

```
User: "Lamp, follow the cup"
         |
    POST /servo/track {"target": "cup"}
         |
    1. Freeze servos 0.3s → grab a sharp frame
         |
    2. Detect the object (YuNet face | local YOLOv8n | remote YOLOWorld) → bbox
         |
    3. TrackerVit init on the bbox
         |
    4. Two decoupled threads:
         |   a. Vision loop @ FAST_LOOP_FPS (10):
         |        ViT update → alpha-beta centroid filter → soft dead zone
         |        → PID + velocity feedforward → publish an absolute servo goal
         |        (background YOLO re-detect every 1.5s corrects drift)
         |   b. Servo worker: SmoothDamp glide toward the latest goal
         |        (ease-in/ease-out, one bus write per ~30ms tick)
         |
    5. Lost / bloated / no-detect / timeout → auto-stop, hold or return to zero
```

Vòng lặp vision không bao giờ block chờ motor di chuyển: nó publish một servo goal *tuyệt đối* rồi chuyển ngay sang frame kế tiếp. Servo worker sở hữu chuyển động vật lý và liên tục ease về phía goal mới nhất. Đây chính là cái giữ cho cả fps của tracker cao lẫn chuyển động đầu mượt.

### Vision downscale, tính toán ở độ phân giải gốc

Camera chạy **1280×720**. Mọi thành phần vision nặng — ViT tracker và cả ba detector — đều chạy trên frame đã downscale xuống `VISION_MAX_WIDTH` (640 px rộng, 0.5× → ¼ số pixel) để tăng tốc. Mỗi bbox chúng tạo ra được map **ngược về tọa độ gốc 1280×720** trước bất kỳ phép tính servo/PID nào (`_downscale` / `_scale_bbox` / `_vit_init` / `_vit_update`, và `detect_object` là transparent). Vì hợp đồng tọa độ ở phía sau luôn là độ phân giải gốc, nên không hằng số nào được tune theo pixel (PID gains, gates, dead zones, ngưỡng feedforward) thay đổi khi hệ số downscale thay đổi. Đặt `VISION_MAX_WIDTH = 0` để tắt.

## Detection

`detect_object(frame, target)` trả về một bbox `(x, y, w, h)` theo tọa độ camera gốc, thử ba đường theo thứ tự:

| Path | Detector | Khi nào | Tốc độ (A523) |
|------|----------|---------|---------------|
| 0 | **YuNet** face detector (`face_detection_yunet_2023mar.onnx`) | target ∈ {`face`, `human face`, `khuôn mặt`, `mặt`} | ~30 ms |
| 1 | **Local YOLOv8n** (COCO classes, `yolov8n.pt`, imgsz=320) | target map tới một COCO class | ~260–770 ms |
| 2 | **Remote YOLOWorld** open-vocab (`{DL_BACKEND_URL}/detect/yoloworld`) | target không thuộc COCO, hoặc local miss (fallback) | ~1.3–2.8 s |

- COCO không có class hand/face, nên `hand`/`face` cố ý rơi xuống YuNet/YOLOWorld thay vì map tới `person` (vốn khóa vào toàn thân).
- Khi local-YOLO miss, code fallback về remote YOLOWorld, **throttle** tối đa một lần mỗi `REMOTE_FALLBACK_MIN_INTERVAL` (2.0 s) để một target thật sự không thể thấy không gọi remote mỗi lần redetect.
- Bộ lọc chất lượng detection: confidence ≥ `DETECT_MIN_CONFIDENCE` (0.15), diện tích nằm giữa `DETECT_MIN_AREA_RATIO` (0.3%) và `DETECT_MAX_AREA_RATIO` (80%) của frame.

Weights được check vào repo (`os/hal/drivers/tracking/models/`) nên deploy chỉ một lần rsync và Pi không bao giờ cần internet lúc boot để bắt đầu tracking.

## Tracker: TrackerVit

**Model:** `os/hal/drivers/tracking/models/vittrack.onnx` (đã check vào repo)

| Tính năng | Giá trị |
|-----------|---------|
| Tốc độ | ~15–25 ms/frame trên frame đã downscale |
| Confidence score | `getTrackingScore()` 0.0–1.0 mỗi frame |
| Xử lý scale | Tự động điều chỉnh kích thước bbox |
| Phát hiện mất | Trả về `ok=False` + score thấp khi vật thể biến mất |

**Chuỗi fallback:** TrackerVit → CSRT → KCF → MIL. Chỉ ViT phơi ra confidence score (dùng cho phát hiện ghost-lock); các tracker khác trả về 1.0.

## Servo Control

Tracking điều khiển 4 joint:

- **base_yaw** (ID 1) — pan trái/phải (100 % của yaw)
- **base_pitch** (ID 2) — tilt lên/xuống, 10 % của pitch
- **elbow_pitch** (ID 3) — tilt lên/xuống, 90 % của pitch
- **wrist_pitch** (ID 5) — tilt lên/xuống, 0 %

Pitch được dồn vào elbow (`PITCH_WEIGHT_ELBOW = 0.90`). Thực nghiệm cho thấy chỉ các joint xoay-thuần mới đưa vật thể về giữa; base/wrist chủ yếu tịnh tiến camera (kinematic coupling), nên weight của chúng thấp/bằng không. Chiều dương của motor elbow bị đảo ở phần cứng, nên đóng góp của nó mang `ELBOW_PITCH_SIGN = -1.0`.

### Control law (vision loop → servo goal)

Mỗi frame vòng lặp biến bbox của tracker thành một servo goal tuyệt đối:

1. **Alpha-beta filter trên centroid** (`AlphaBetaFilter2D`) — một Kalman steady-state vận tốc-hằng. Làm mượt jitter, coast qua các frame bị rớt/rác bằng prediction, gate các cú teleport outlier (`AB_GATE_PX`), và phơi ra ước lượng vận tốc. Một velocity lead (`AB_LEAD_S = 0.12 s`) nhắm hơi vượt trước target.
2. **Soft dead zone** (`_soft_deadband`) — error bằng 0 bên trong dead zone và tăng dần từ 0 tại rìa (không có bước nhảy giá trị). Cái này loại bỏ cú giật "đá ra khỏi giữa" mà hard dead zone cũ tạo ra.
3. **PID + velocity feedforward** — một PID time-aware có anti-windup trên position error đã qua soft-deadband, **cộng** một term feedforward tỉ lệ với vận tốc pixel đo được của target (`VFF_GAIN`). Feedforward pan camera *theo tốc độ của target* ngay cả khi position error bằng không, nên một target di chuyển đều là một pha pan liên tục thay vì các cú bùng đuổi bắt. Một target ở giữa nhưng đang di chuyển vẫn tiếp tục pan (không đóng băng trong dead zone). Output kết hợp bị clamp về `PID_OUTPUT_MAX_DEG` (5°).
4. **Publish goal** — joint target tuyệt đối kết quả được giao cho servo worker (non-blocking).

### Servo worker (SmoothDamp follower)

`_servo_worker` chạy trên thread riêng và liên tục ease các joint về phía goal mới nhất bằng **SmoothDamp** (`_smooth_damp`, một follower critically-damped): mỗi joint mang vận tốc riêng, nên mọi cú di chuyển đều tăng tốc mượt và ease-out vào target, và một goal mới đến giữa cú di chuyển sẽ retarget mà không giật restart — chuyển động "film camera" điện ảnh. Nó phát ra **một bus write mỗi tick `SERVO_SUBSTEP_SLEEP` (30 ms)**, cùng nhịp click như ramp fixed-substep cũ (Feetech STS3215 click mỗi lần write, nên tần suất write phải giữ có giới hạn — SmoothDamp thay đổi *cái gì* được ra lệnh mỗi tick, không phải *bao lâu một lần*).

Giới hạn chuyển động phần cứng khi tracking: `TRACKING_GOAL_VELOCITY = 150` steps/s và `TRACKING_ACCELERATION = 30` (ramp nhẹ nhàng). Khôi phục về default nhạy bén khi tracking kết thúc.

### Sửa drift & quản lý lock

- **Background YOLO re-detect** mỗi `YOLO_REDETECT_S` (1.5 s) trên một worker thread (không bao giờ block fast loop; kết quả gửi qua một queue `maxsize=1`). Bị buộc chạy ngay khi vật thể tiến gần rìa frame (>25 %) hoặc lần CSRT miss đầu tiên.
- **Reinit gating (kiểu SORT/ByteTrack)** — một re-detect chỉ reinit tracker khi nó đã rõ ràng phân kỳ, để tránh churn reinit làm servo quật qua lại:
  - **Area gate** `YOLO_AREA_GATE_MULT` (4.0) — loại một detection có diện tích >4× hoặc <¼ median của 5 cái gần nhất; đừng reinit về nó.
  - **Reinit debounce** `REINIT_COOLDOWN_S` (0.5 s) — rate-limit reinit; chỉ bypass khi lock rõ ràng đã mất (`center_dist > frame_diag × LOST_CENTER_FRAC` = 0.5).
- **Bbox-trust guard (bloat hold)** — khi ViT lock tan thành một box quá khổ thì centroid là rác, nên servo giữ nguyên thay vì đuổi theo nó:
  - `BBOX_FREEZE_RATIO` (1.0) — bbox ≥ diện tích cả frame ⇒ ViT đã tan.
  - `BLOAT_HOLD_MULT` (3.0) — bbox > 3× diện tích lock tin cậy gần nhất ⇒ hold và buộc re-detect.
- **Detector-gated trust** — nếu không detector nào xác nhận trong `TRUST_TRACKER_S` (2.5 s) và confidence ViT < `TRACKER_TRUST_CONF` (0.4), giữ servo (`WAIT-YOLO`) thay vì đuổi một bóng ma; confidence ViT cao vẫn tiếp tục fire ngay cả khi không có detector confirm mới.

### Chuyển đổi Pixel-sang-Degree

```
deg_per_px = CAMERA_FOV_DEG / frame_width          (same on both axes for square pixels)

dx = filtered_lead_x - frame_width/2   (positive = right)
dy = filtered_lead_y - frame_height/2  (positive = below)

yaw_step         = clamp(PID(soft_deadband(dx)) + VFF·vx·deg_per_px·dt,  ±5°)
pitch_correction = clamp(PID(soft_deadband(dy)) + VFF·vy·deg_per_px·dt,  ±5°)
```

### Hằng số tuning

| Hằng số | Giá trị | Mô tả |
|---------|---------|-------|
| `VISION_MAX_WIDTH` | 640 | Chiều rộng downscale cho ViT + detectors (0 = tắt) |
| `FAST_LOOP_FPS` | 10 | Tần số vòng lặp vision |
| `CAMERA_FOV_DEG` | 60 | FOV ngang, cho px→deg |
| `DEAD_ZONE_YAW_PCT` / `_PITCH_PCT` | 0.07 / 0.05 | Soft dead zone theo tỉ lệ frame |
| `PID_YAW_KP` / `PID_PITCH_KP` | 0.025 / 0.03 | PID proportional gains |
| `PID_OUTPUT_MAX_DEG` | 5.0 | Số độ tối đa mỗi lần fire (yaw & pitch kết hợp) |
| `AB_ALPHA` / `AB_BETA` | 0.6 / 0.2 | Alpha-beta position/velocity gains |
| `AB_GATE_PX` | 200 | Loại một cú teleport centroid vượt residual này |
| `AB_LEAD_S` | 0.12 | Velocity lead (nhắm vượt trước target) |
| `VFF_GAIN` | 0.6 | Tỉ lệ vận tốc target được feed forward |
| `VFF_MAX_DT_S` | 0.20 | Cap trên dt mỗi lần fire cho feedforward |
| `VFF_MOVING_MIN_PXS` | 40 | Tốc độ target mà trên đó target ở giữa vẫn tiếp tục pan |
| `SERVO_SMOOTH_TIME` | 0.18 | Hằng số thời gian SmoothDamp (↓ nhạy hơn, ↑ mượt/trễ hơn) |
| `SERVO_MAX_SPEED_DPS` | 60 | Cap tốc độ pan đỉnh của SmoothDamp |
| `SERVO_SUBSTEP_SLEEP` | 0.030 | Tick servo-worker / chu kỳ bus-write |
| `TRACKING_GOAL_VELOCITY` | 150 | Giới hạn vận tốc phần cứng (steps/s) |
| `TRACKING_ACCELERATION` | 30 | Ramp gia tốc phần cứng |
| `PITCH_WEIGHT_BASE/ELBOW/WRIST` | 0.10 / 0.90 / 0.0 | Phân bổ pitch qua các joint |
| `ELBOW_PITCH_SIGN` | -1.0 | Chiều elbow (phần cứng đảo) |
| `YOLO_REDETECT_S` | 1.5 | Khoảng thời gian background re-detect |
| `YOLO_AREA_GATE_MULT` | 4.0 | Loại re-detect có diện tích outlier |
| `REINIT_COOLDOWN_S` | 0.5 | Số giây tối thiểu giữa các lần reinit tracker |
| `BBOX_FREEZE_RATIO` | 1.0 | Bbox ≥ frame ⇒ ViT đã tan (hold) |
| `BLOAT_HOLD_MULT` | 3.0 | Bbox > 3× lock tin cậy ⇒ hold |
| `CONFIDENCE_THRESHOLD` | 0.15 | Dưới mức này = frame low-confidence |
| `MAX_LOW_CONFIDENCE_FRAMES` | 10 | Số frame low-confidence liên tiếp → dừng |
| `YOLO_MAX_MISS` | 30 | Số lần CSRT miss liên tiếp trước khi retry |
| `MAX_TRACK_DURATION_S` | 300 | Timeout tự động dừng (5 phút) |
| `_LOCAL_IMGSZ` | 320 | Kích thước inference local YOLO (640 → 1.3–2.9 s, quá chậm) |

> Ghi chú legacy: đường proportional `GIMBAL_GAIN` / `GIMBAL_MAX_STEP` / `EMA_ALPHA` (`_fire_gimbal` / `_send_gimbal_target`) đã **chết** — điều khiển live là đường PID + feedforward (`_fire_pid`). Đừng tune mấy cái đó cho độ nhạy.

### Giới hạn vị trí servo

| Joint | Min | Max |
|-------|-----|-----|
| base_yaw | -135 | 135 |
| base_pitch | -90 | 30 |
| elbow_pitch | -90 | 90 |
| wrist_pitch | -90 | 90 |

## Điều kiện tự động dừng

| Điều kiện | Hành động |
|-----------|-----------|
| `confidence < 0.15` trong 10 frame | Dừng — mất target |
| Bbox co nhỏ dưới `DETECT_MIN_AREA_RATIO` | Dừng — ghost-lock trên một mảnh nhỏ |
| Bbox tràn frame + không detect trong 3 s | Buộc retry, rồi dừng nếu không phục hồi |
| Không detector confirm trong `STOP_NO_YOLO_S` (20 s) | Dừng — ghost tracking |
| CSRT miss `YOLO_MAX_MISS` (30) sau `MAX_TRACKING_RETRIES` (4) | Dừng — vật thể biến mất |
| Thời lượng tracking > 5 phút | Dừng — timeout để tiết kiệm motor/CPU |

Lưu ý: một bbox lớn (ví dụ một người lấp đầy frame) **không** phải điều kiện dừng — PID chạy theo centroid, không phải kích thước bbox, nên một vật thể ở gần vẫn track. Khi tracking kết thúc, cánh tay trượt về zero ở tốc độ tracking (không snap).

### Tự động dừng khi mất kết nối gateway/network

Object tracking được điều khiển bởi các cập nhật vision từ xa từ agent/cloud. Khi gateway WebSocket disconnect (mất cloud hoặc internet), thiết bị tự động dừng mọi servo tracking đang chạy — `os/services/internal/openclaw/service_ws.go` gọi `hal.StopServoTracking()` → HAL `POST /servo/track/stop` (best-effort, được guard bởi `SetUpCompleted`). Nếu không có cập nhật từ xa mới, tracking tiếp tục sẽ cứ nhắm thân về một target cũ mà nó không còn sửa được, nên nó bị dừng như một phản xạ an toàn. Idle animation local vẫn tiếp tục (thiết bị vẫn "sống", không đóng băng) và phục hồi (`/servo/track/stop`, stop/release) vẫn khả dụng. Xem `devices/lamp/SAFETY.md` → `## fail-safe states` (dòng Network/gateway loss, enforced).

## API Endpoints

Tất cả nằm dưới `/servo/track`.

### GET /servo/track/targets — Liệt kê target gợi ý

```json
{"targets": ["person", "cup", "bottle", "glass", "phone", "laptop", ...]}
```

Detection là open-vocabulary qua YOLOWorld (và YuNet cho khuôn mặt) — mọi text đều được, danh sách này chỉ là gợi ý.

### POST /servo/track — Bắt đầu tracking

`target` nhận hoặc một string đơn hoặc một list các label ứng viên. Khi truyền một list, label không rỗng đầu tiên được dùng. Hữu ích khi caller (ví dụ một LLM skill) không chắc label chính xác nào sẽ match.

```json
// Auto-detect, single label
{"target": "cup"}

// Auto-detect, list of candidate labels (preferred from LLM skills)
{"target": ["cup", "mug", "coffee cup"]}

// Manual bbox (skip detection — target is for display only)
{"bbox": [190, 50, 170, 300], "target": "cup"}

// Response
{
  "status": "ok",
  "tracking": true,
  "target": "cup",
  "bbox": [190, 50, 170, 300],
  "confidence": 1.0
}
```

### POST /servo/track/stop — Dừng tracking

```json
{"status": "ok", "tracking": false}
```

### GET /servo/track — Kiểm tra trạng thái

```json
{
  "status": "ok",
  "tracking": true,
  "target": "cup",
  "bbox": [195, 55, 175, 295],
  "confidence": 0.612
}
```

### POST /servo/track/update — Re-init bbox

Re-init thủ công tracker với một bbox mới mà không dừng session (background YOLO re-detect tự xử lý drift; cái này dành cho caller muốn kiểm soát tường minh).

```json
{"bbox": [250, 160, 75, 95], "target": "cup"}
```

## Luồng End-to-End

### Happy path

```
1. User: "Lamp, follow the cup"
2. Agent calls POST /servo/track {"target": "cup"}
3. HAL internally:
   a. Freezes servos 0.3s and snapshots a sharp frame
   b. Detects "cup" (local YOLOv8n, or remote YOLOWorld) → bbox
   c. TrackerVit init uses the same frame + bbox (coordinates match)
   d. Starts the vision loop + servo worker
4. Servo pans smoothly to follow the cup, background YOLO corrects drift
5. User: "OK stop" → agent calls POST /servo/track/stop
6. Servo glides back to zero
```

### Tự động dừng khi mất

```
1. Object leaves frame or is occluded
2. TrackerVit confidence drops below 0.15 (or ViT lock dissolves)
3. Background YOLO can't re-find it → after the guards trip → auto-stop
4. Arm returns to zero
5. Agent can notify user or re-issue the follow command
```

## Overlay Camera Stream

Khi tracking đang chạy, MJPEG stream (`/camera/stream`) vẽ:
- Bounding box màu xanh quanh vật thể được track
- Label target phía trên box

## Web UI

Camera section hiển thị:
- **Vision Tracking card** — target input, bbox input, các nút Start/Stop/Status
- **Stream badge** — "LIVE" hoặc "TRACKING: {target}"
- **Confidence** — hiện trong panel thông tin tracking
- **Polling** — status refresh mỗi 3 giây

## Dependencies

- `opencv-python>=4.8.0` (đã có trong `pyproject.toml`)
- `ultralytics` — inference local YOLOv8n
- `vittrack.onnx`, `yolov8n.pt`, `face_detection_yunet_2023mar.onnx` — đã check vào `os/hal/drivers/tracking/models/`
- `requests` (đã có trong project)
- **YOLOWorld API** — DL backend tại `{DL_BACKEND_URL}/detect/yoloworld` (chỉ open-vocab fallback)

## Tương tác với các hệ thống khác

| Hệ thống | Trong khi tracking | Sau khi tracking |
|----------|--------------------|------------------|
| Servo idle animation | Bị chặn (`_hold_mode`) | Tiếp tục |
| `/servo/play` | Bị chặn bởi `_hold_mode` | Tiếp tục |
| Sensing (face, motion) | Tiếp tục — chia sẻ camera | Tiếp tục |
| Camera stream overlay | Vẽ bbox xanh | Stream bình thường |
| TTS | Tiếp tục bình thường | Tiếp tục bình thường |

## Ghi chú hiệu năng

- Sàn CPU của fast-loop trên Allwinner A523 là chi phí ViT inference + detector; frame downscale (`VISION_MAX_WIDTH`) và local imgsz=320 là các đòn bẩy chính.
- Độ mượt chuyển động đến từ servo worker tách rời + SmoothDamp + velocity feedforward; alpha-beta filter + reinit gating giữ cho bản thân goal ổn định để follower không đuổi theo nhiễu.
- Vật thể nhỏ/xa (ví dụ một cái ly ở đầu phòng) có thể vượt độ phân giải của cả detector local lẫn remote — đây là giới hạn perception, không phải bug điều khiển.
</content>
</invoke>
