# Flow C — Music personalization

How `music_patterns` in `patterns.json` is computed and consumed. Read this when building / refreshing music personalization or when integrating from `music-suggestion/SKILL.md`.

## Building `music_patterns`

1. Read `music-suggestions/` for accepted suggestions + their trigger times.
2. Group accepted suggestions by hour → find peak hours. Extract genre hint from the `message` field (e.g. "lo-fi", "jazz", "piano").
3. Write to `music_patterns` in `patterns.json` (schema in `reference/build-patterns.md`).

## Consuming from music-suggestion

`music-suggestion/SKILL.md` reads `habit/patterns.json` → `music_patterns`. If the current hour matches `peak_hour ± 1`, use `preferred_genre` instead of the default mood-based genre table.

## JSONL example

```json
{"ts": 1234567, "hour": 14, "trigger": "mood:tired", "message": "Want some calm piano?", "status": "accepted"}
```
