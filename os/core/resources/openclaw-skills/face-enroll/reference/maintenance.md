# Maintenance — Status / Remove / Reset

Read-only and destructive ops on the enrolled face set.

## Check who is recognized

**Trigger:** "who do you recognize?", "how many faces?", "list faces".

```bash
curl -s http://127.0.0.1:5001/face/status
```

Response:
```json
{"enrolled_count": 2, "enrolled_names": ["chloe", "leo"]}
```

Reply with names in plain prose, not raw JSON.

## Remove own face

**Trigger:** "forget my face", "remove my face".

1. Verify the requester matches the enrolled person — match by sender name (Telegram prefix) or `telegram_id` from message metadata. If a different user asks to remove someone else's face, refuse.
2. Call:
   ```bash
   curl -s -X POST http://127.0.0.1:5001/face/remove \
     -H "Content-Type: application/json" \
     -d '{"label": "chloe"}'
   ```
3. Confirm removal.

Returns 404 if the label is not enrolled — tell the user "I don't have a face on file under that name."

## Reset all faces

**Trigger:** "forget all faces", "reset faces", "wipe all faces" — owner-level, destructive.

```bash
curl -s -X POST http://127.0.0.1:5001/face/reset
```

Confirm all faces cleared. Use sparingly; this also deletes photo files.
