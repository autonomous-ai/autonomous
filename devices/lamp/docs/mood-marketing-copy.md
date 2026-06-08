# Mood Skill — brief cho Marketing chỉnh copy

> File này tóm tắt **Mood skill** + chuỗi skill kéo theo (user-emotion-detection → mood → music-suggestion) để marketing chỉnh lời nói của Lamp khi phát hiện cảm xúc người dùng.
>
> **Khác với wellbeing:** Mood skill **bản thân là skill im lặng** — chỉ log dữ liệu, không nói. Phần user-facing copy nằm ở **music-suggestion** (gợi ý nhạc theo mood).
>
> File gốc cần chỉnh:
> - `lamp/resources/openclaw-skills/music-suggestion/SKILL.md` — câu gợi ý nhạc theo mood (chính)
> - `lamp/resources/openclaw-skills/mood/SKILL.md` — không có copy table, chỉ có ràng buộc "1 câu caring tự nhiên hoặc im lặng"
> - `lamp/resources/openclaw-skills/user-emotion-detection/SKILL.md` — bảng map cảm xúc khuôn mặt → mood (không có copy, chỉ logic)

---

## 1. Chuỗi xử lý (để marketing hiểu bối cảnh)

```
Camera phát hiện cảm xúc khuôn mặt
      ↓
[user-emotion-detection]  ← map cảm xúc → mood, log signal (im lặng)
      ↓
[mood]                    ← tổng hợp signal + lịch sử → quyết định mood (im lặng)
      ↓
[music-suggestion]        ← nếu mood "đáng gợi ý nhạc" → NÓI 1 câu rủ user
```

**Người dùng chỉ nghe** câu cuối cùng từ music-suggestion. 2 skill phía trước hoàn toàn im lặng.

## 2. Map cảm xúc khuôn mặt → mood

Camera detect được 7 cảm xúc cơ bản (FER labels). Lamp quy về 1 trong 11 mood:

| Camera detect | Mood |
|---|---|
| `Happy` | happy |
| `Sad` | sad |
| `Angry` | frustrated |
| `Fear` | stressed |
| `Surprise` | excited |
| `Disgust` | frustrated |
| `Neutral` | normal |

Ngoài ra mood còn nhận signal từ **giọng nói** (sigh, laugh, raised...) và **tin nhắn Telegram** ("I'm tired", "let's gooo"...) — Lamp tự suy ra mood từ ngữ cảnh.

Danh sách mood đầy đủ: `happy, sad, stressed, tired, excited, bored, frustrated, energetic, affectionate, unwell, normal`.

## 3. Mood nào sẽ kích hoạt câu nói?

Chỉ **6 mood "đáng gợi ý nhạc"** mới khiến Lamp mở miệng:

| Mood | Có nói không? |
|---|---|
| `sad`, `stressed`, `tired`, `excited`, `happy`, `bored` | ✅ Có — gợi ý nhạc |
| `frustrated`, `energetic`, `affectionate`, `unwell`, `normal` | ❌ Không — im lặng |

**Thêm điều kiện im lặng** (tất cả phải pass):
- Đang phát nhạc rồi → im lặng.
- Đã gợi ý cách đây <30 phút → im lặng (tránh spam).
- Mood quá cũ (>30 phút) hoặc không có → im lặng.

## 4. Copy hiện tại — câu gợi ý nhạc theo mood

File: `music-suggestion/SKILL.md` — phần Examples (dòng 88–98).

| Mood | Câu gợi ý hiện tại |
|---|---|
| **tired** | *"You seem tired — want some calm piano?"* |
| **stressed** | *"You look a bit tense — want some soft piano to ease into?"* |
| **sad** | *"Rough moment? Some gentle acoustic might help."* |
| **bored** | *"Need a lift? How about some upbeat indie?"* |
| **excited** | *"Riding the energy — feel-good pop?"* |
| **happy** | (chưa có sample cố định — Lamp tự nghĩ; thường là một câu vui rủ nhạc upbeat) |

Sau khi user **đồng ý**, Lamp nói tiếp:
- *"Great choice!"* (kèm marker phát nhạc)

## 5. Bảng thể loại nhạc default theo mood

Lamp chọn thể loại dựa trên mood (file `music-suggestion/SKILL.md` dòng ~50):

| User state | Thể loại default |
|---|---|
| Tired / fatigued | Calm piano, gentle acoustic, nature sounds |
| Stressed / tense | Soft jazz, classical, meditation |
| Happy / energetic | Upbeat pop, jazz, feel-good classics |
| Bored / restless | Fun pop, disco, upbeat indie |

Nếu user có lịch sử nghe rõ rệt (vd hay nghe K-pop) → override bảng này. Nếu có habit pattern (vd 2–4h chiều thường nghe lo-fi) → override luôn theo habit.

## 6. Tone & ràng buộc khi chỉnh copy

Đây là phần **rất chặt** — đã được fix qua nhiều lần điều chỉnh:

### 6.1. Bắt buộc
- **1 câu duy nhất**, không tách 2–3 câu.
- **Có dấu hỏi hoặc lời rủ** ("...?", "...might help.", "How about...?") — phải rủ user, không thông báo.
- **Tone match với mood:**
  - `sad` / `stressed` / `tired` → **caring**, nhẹ nhàng, đồng cảm. KHÔNG vui vẻ, KHÔNG cợt nhả.
  - `happy` / `excited` / `bored` → có thể tươi tắn hơn, vẫn không over.
- **Gợi ý 1 thể loại / 1 bài cụ thể** trong câu (vd: "calm piano", "lo-fi", "Norah Jones") — không được mơ hồ kiểu "muốn nghe nhạc không?".

### 6.2. CẤM tuyệt đối
- ❌ **Không chào** mở đầu (`hello`, `hi`, `hey`, `welcome back`, `oh you're back`, "again..."). Đây không phải sự kiện user mới đến — chỉ là cảm xúc thay đổi giữa chừng.
- ❌ **Không nhắc đến camera / cảm biến / detection.** Cấm các cụm:
  - *"I noticed you look..."*
  - *"I can see..."*
  - *"Your face shows..."*
  - *"The camera detected..."*

  Lamp phải nói như **một người bạn quan tâm tự nhiên**, không phải robot báo cáo cảm biến.
- ❌ **Không tự ý phát nhạc.** Chỉ gợi ý — chờ user xác nhận mới phát.
- ❌ **Không nói tên mood ra mặt** (vd: *"You're sad"*, *"Mood: tired"*). Phải diễn đạt gián tiếp (*"Rough moment?"*, *"You seem tired"*).
- ❌ **Không nhắc cooldown / interval / timestamp** trong câu nói.

### 6.3. Ví dụ tốt vs xấu

| ❌ Xấu | ✅ Tốt | Lý do |
|---|---|---|
| *"Hey! I noticed you look sad — want music?"* | *"Rough moment? Some gentle acoustic might help."* | Có chào + nhắc detect, mood mơ hồ, gợi ý không cụ thể |
| *"You're tired. Listen to music?"* | *"You seem tired — want some calm piano?"* | Thông báo mood thẳng, không có lời rủ, không cụ thể |
| *"Welcome back! You seem stressed."* | *"You look a bit tense — want some soft piano to ease into?"* | Có "welcome back" (bị cấm) |
| *"The camera shows you're happy! Pop?"* | *"Riding the energy — feel-good pop?"* | Nhắc camera + dùng label "happy" trực tiếp |

## 7. Bản thân Mood skill — copy spoken reply

File `mood/SKILL.md` **không có bảng copy** vì mood là skill internal — chỉ log dữ liệu.

Tuy nhiên, **nếu Lamp có nói gì** sau khi log mood (trước khi music-suggestion chạy), ràng buộc là:
- **Tối đa 1 câu caring ngắn**, hoặc `NO_REPLY`.
- **Không nhắc workflow / step / mood label / log / curl** trong câu nói.
- **Không liệt kê** lịch sử mood vừa đọc (vd cấm: *"- Normal (15:00) — ..."*).
- **Không bắt đầu bằng "Now I follow...", "Let me check...", "Next step..."**.

Trên thực tế, mood gần như luôn `NO_REPLY` và để music-suggestion lo phần nói.

---

## 8. Checklist khi marketing chỉnh copy

- [ ] Mỗi câu vẫn là **1 câu** (không xuống dòng, không 2 sentence).
- [ ] Có dấu hỏi hoặc lời rủ rõ ràng.
- [ ] Có tên thể loại / artist cụ thể trong câu (không "want some music?" trống).
- [ ] **Tone match mood:** sad/stressed/tired → gentle, không cheerful; happy/excited → tươi nhưng vừa phải.
- [ ] Không có bất kỳ lời chào nào (`hello`, `hi`, `welcome back`, `again`...).
- [ ] Không nhắc camera/detection/sensor/face.
- [ ] Không gọi mood ra trực tiếp (*"You're sad"* → đổi thành *"Rough moment?"*).
- [ ] Có ≥2–3 biến thể cho mỗi mood để Lamp không lặp lại.
- [ ] Nếu làm bản tiếng Việt: thêm cột `vi` ở Examples; **giữ** bản English (các skill khác đang fallback).

---

## 9. File để chỉnh

| Nội dung | File | Vị trí |
|---|---|---|
| Câu gợi ý nhạc theo từng mood | `lamp/resources/openclaw-skills/music-suggestion/SKILL.md` | mục **Examples**, dòng ~88–98 |
| Bảng thể loại nhạc default | `lamp/resources/openclaw-skills/music-suggestion/SKILL.md` | mục **Pick genre → default genre table**, dòng ~50–56 |
| Ràng buộc tone (không chào, không nhắc camera...) | `lamp/resources/openclaw-skills/music-suggestion/SKILL.md` | mục **Rules**, dòng ~100–108 |
| Map camera emotion → mood | `lamp/resources/openclaw-skills/user-emotion-detection/SKILL.md` | mục **Emotion → mood**, dòng ~38–48 |
| Mood values list | `lamp/resources/openclaw-skills/mood/SKILL.md` | mục **Mood Values**, dòng ~30–34 |

Sau khi chỉnh, deploy lại Lamp để Pi load skill mới (hỏi dev — không tự SSH).
