# Wellbeing Skill — brief cho Marketing chỉnh copy nhắc nhở

> File này tóm tắt skill **wellbeing** (nhắc uống nước / nghỉ ngơi) và phần **habit-aware** (nhắc theo thói quen riêng) để marketing chỉnh lại lời nói của Lamp.
>
> File gốc cần chỉnh:
> - `lamp/resources/openclaw-skills/wellbeing/SKILL.md` — bảng copy chính (Step 4)
> - `lamp/resources/openclaw-skills/habit/SKILL.md` — copy habit-aware (Output Examples)

---

## 1. Mục đích

Lamp quan sát qua camera/sensor và **nhắc người dùng uống nước hoặc nghỉ ngơi** khi ngồi/làm việc lâu một chỗ. Mỗi lượt chỉ nói **1 câu ngắn**, hoặc im lặng (`NO_REPLY`) nếu chưa đến lúc.

## 2. Khi nào Lamp nhắc

Mỗi lần camera phát hiện hoạt động (`motion.activity`), Lamp tính thời gian từ lần "reset" gần nhất:

| Loại nhắc | Ngưỡng production | Reset khi |
|---|---|---|
| **Hydration** (uống nước) | 45 phút | user uống nước, mới vào phòng (`enter`), hoặc Lamp vừa nhắc xong |
| **Break** (nghỉ ngơi) | 30 phút | user đứng dậy nghỉ, mới vào phòng (`enter`), hoặc Lamp vừa nhắc xong |

- Mỗi lượt chỉ nhắc **1 thứ** — hydration ưu tiên trước break.
- Sau khi đã nhắc, đồng hồ reset → lần sau phải đợi đủ ngưỡng nữa.
- Chưa đủ ngưỡng → **im lặng**, không nói gì.

## 3. Tone & ràng buộc câu nói

- **1–2 câu ngắn**, không xuống dòng. Pattern khuyến nghị: 1 câu quan sát + 1 câu rủ hành động (vd: *"You've been at the screen a while. Want some water?"*). 1 câu cũng OK nếu gọn và đủ ấm.
- Ấm áp, kiểu **bạn bè quan sát** — không phải robot báo động, không "thưa anh/chị".
- **Phải có dấu hỏi hoặc lời rủ hành động** ("...?", "...for a sec?") — mục tiêu là rủ user uống/nghỉ chứ không phải thông báo.
- **Bám vào hành động đang thấy** (đang dùng máy tính, đang viết, đang đọc...) cho cảm giác Lamp để ý người dùng thật.
- **Không lặp lại** — mỗi lần phải đổi cách nói. Bảng dưới là **tham khảo tone**, agent không bao giờ được nói y chang một dòng trong bảng.
- Không emoji, không hashtag.

---

## 4. Bảng copy hiện tại — Wellbeing (file `wellbeing/SKILL.md` Step 4)

Lamp chọn câu dựa trên **hoạt động đang quan sát** (raw label từ camera):

| Hoạt động đang thấy | Câu nhắc nước (tone, không copy y nguyên) | Câu nhắc nghỉ (tone, không copy y nguyên) |
|---|---|---|
| `using computer` | *"You've been at the screen a while. Want some water?"* | *"Your eyes have been working. Look up for a sec?"* |
| `writing` | *"Pen's been moving a while. Sip of water?"* | *"Your hand's been busy. Time for a stretch?"* |
| `texting` | *"Phone's had your attention a bit. Water nearby?"* | *"You've been on your phone a while. Stand up for a sec?"* |
| `reading book` | *"Deep in it. Water before the next chapter?"* | *"You've been reading a while. Rest your eyes?"* |
| `reading newspaper` | *"You've been on the page a while. Water alongside?"* | *"Eyes have been working. Look up for a moment?"* |
| `drawing` | *"You've been at it. Sip of water?"* | *"Your hand's been working. Quick stretch?"* |
| `playing controller` | *"Mid-session. Water within reach?"* | *"You've been playing a while. Stand up between rounds?"* |
| (không rõ / chung chung) | *"Been a while since I saw you drink anything. Water?"* | *"You've been still a while. Stretch?"* |

**Lưu ý:**
- Bảng này là **tham khảo tone**, không phải template — agent phải paraphrase mỗi lượt, không nói y nguyên 1 dòng.
- Nhiều hoạt động cùng lúc → có thể gộp tự nhiên (vd nhắc cả mắt và cổ tay trong 1 câu).
- Hiện copy đang **full English**. Muốn dùng tiếng Việt cần thêm cột `vi` cho từng raw label.

---

## 5. Bảng copy hiện tại — Habit-aware (file `habit/SKILL.md` Output Examples)

Khi Lamp đã học được **thói quen riêng** của user (cần ≥3 ngày dữ liệu), nó có thể chèn ngữ cảnh "thường ngày này..." vào câu nhắc, thay vì câu chung chung.

### Điều kiện kích hoạt habit-aware
- User có ≥3 ngày lịch sử wellbeing.
- Pattern đủ mạnh: `strength` = moderate (xuất hiện 50–75% số ngày) hoặc strong (>75%).
- Thời điểm hiện tại trùng khung giờ thường làm hành động đó (vd: thường uống nước lúc 9h sáng, giờ là 9:15).

Nếu không đủ điều kiện → dùng câu chung ở bảng mục 4.

### Câu nhắc enrich theo habit (khi pattern khớp)

| Tình huống | Ví dụ câu nói |
|---|---|
| **Nhắc nước theo thói quen** | *"You usually have water around now — everything okay?"* |
| **Xác nhận thói quen** (user vừa quay lại bàn đúng giờ thường lệ) | *"Back at your desk right on schedule."* (chỉ dùng nếu thấy tự nhiên) |
| **Gợi ý nhạc theo thói quen** (nếu được music skill gọi) | *"It's your usual coding time — want some lo-fi?"* |
| **Không có dữ liệu thói quen** | Im lặng — **không bịa**, không đoán mò |

### Câu trả lời khi user hỏi về thói quen mình (Flow E — open question)

Phần này dài hơn 1 câu, vì user chủ động hỏi:

| Mode | Khi dùng | Ví dụ |
|---|---|---|
| **Pattern mode** | Có ≥5 ngày data, pattern rõ | *"Leo usually arrives around 8:30 with breakfast, settles at the computer through the morning, and wraps up close to 5. Lo-fi tends to land between 2 and 4. Pretty steady the last week."* |
| **Narrative mode** | 2–4 ngày data, chưa đủ thành habit | *"I've only got two real days on Chloe so far — April 28 was an evening at the computer with a lot of water breaks, and April 29 ran late, working past midnight. Not enough days yet to call it a habit, but that's what I've seen."* |
| **Honest-gap mode** | Data cũ, lâu không gặp user | *"Honestly, I haven't seen Leo much lately — just one short session yesterday. The patterns I have are from two weeks ago, so I'd rather not pretend they're still true."* |

**Tone của Flow E:** trung thực, không phán xét, dám nói "tôi chưa biết đủ để nói chắc". Đây là khác biệt quan trọng so với câu nhắc 1-câu — ở đây Lamp kể lại quan sát.

---

## 6. Checklist khi marketing chỉnh copy

Trước khi merge bản copy mới, kiểm tra:

- [ ] Mỗi ô trong bảng là **1–2 câu ngắn**, không xuống dòng. Pattern khuyến nghị: quan sát + rủ hành động.
- [ ] Có dấu hỏi hoặc gợi ý hành động (không phải thông báo trống "Đã đến giờ uống nước.").
- [ ] Bảng được hiểu là **tham khảo tone** — agent paraphrase mỗi lượt, không copy verbatim.
- [ ] Ngôi xưng **nhất quán** với các skill khác Lamp đang dùng (hiện: bạn bè, không "anh/chị/quý khách").
- [ ] Câu **bám vào hành động** trong cột bên trái (vd cột `writing` không nên copy giống cột `using computer`).
- [ ] Có **ít nhất 2–3 biến thể** cho mỗi ô để Lamp không lặp.
- [ ] Câu habit-aware (mục 5) có yếu tố "you usually..." hoặc tương đương — đó là điểm khác biệt với câu chung.
- [ ] Nếu làm bản tiếng Việt: thêm cột `vi` ở cả 2 file, **không xoá** cột English (nhiều skill khác đang dùng English làm fallback).

---

## 7. File để chỉnh

| Nội dung | File | Vị trí trong file |
|---|---|---|
| Bảng copy chính (8 hành động × 2 loại nhắc) | `lamp/resources/openclaw-skills/wellbeing/SKILL.md` | mục **Step 4 — Speak**, dòng ~119–126 |
| Câu habit-aware enrich nudge | `lamp/resources/openclaw-skills/habit/SKILL.md` | mục **Output Examples → Nudge enrichment** |
| Câu trả lời khi user hỏi habit (3 mode) | `lamp/resources/openclaw-skills/habit/SKILL.md` | mục **Output Examples → Open habit question** |

Sau khi chỉnh, deploy lại Lamp để Pi load skill mới (hỏi dev để deploy — không tự SSH).
