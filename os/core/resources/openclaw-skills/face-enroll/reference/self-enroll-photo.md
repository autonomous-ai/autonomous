# Flow A — Self-enroll with a user-supplied photo

User sends a photo of **themselves** via Telegram (`mediaPaths`) or web chat (`[image: /path/to/file]` tag) along with an introduction.

## Trigger
- "remember **my** face" / "add **my** photo" / "this is **me**" / "enroll **me**"
- "add me" / "add my face"
- Any message where the user is introducing **themselves** with a photo attached.

Do NOT activate when the user tries to enroll someone else with a photo (e.g. "this is Alice", "add my friend Bob"). Refuse — the other person must either send their own selfie (Flow A) or stand in front of the camera and respond to a familiar-stranger prompt themselves (Flow C). The owner cannot enroll a third party on their behalf.

## Steps

1. Locate the photo path:
   - Telegram → `mediaPaths` in conversation context.
   - Web chat → path inside `[image: /path/to/file]` tag in the message text.
2. Extract the **name**:
   - Prefer name spoken in the message ("I'm Chloe", "this is me, Chloe").
   - Else fall back to sender prefix (e.g. `[telegram:Chloe]` → `chloe`).
   - If still unclear, ask the user.
3. Extract the sender's **Telegram identity** from the message context (required for DM targeting):
   - `telegram_username` (e.g. `chloe_92`)
   - `telegram_id` (numeric Telegram user ID)
   - These come from message metadata. If the message is from web chat (no Telegram), omit both fields.
4. Base64-encode the photo file.
5. Call `POST /face/enroll` with `image_base64`, `label`, and Telegram fields when present:
   ```bash
   curl -s -X POST http://127.0.0.1:5001/face/enroll \
     -H "Content-Type: application/json" \
     -d "{\"image_base64\": \"$(base64 -w0 /path/to/photo.jpg)\", \"label\": \"chloe\", \"telegram_username\": \"chloe_92\", \"telegram_id\": \"123456789\"}"
   ```
6. Confirm enrollment to the user with the new `enrolled_count`.

## Response shape
```json
{"status": "ok", "label": "chloe", "telegram_username": "chloe_92", "telegram_id": "123456789", "photo_path": "...", "enrolled_count": 2}
```

## Notes
- The `/face/enroll` endpoint stores the JPEG under `/root/local/users/<label>/`.
