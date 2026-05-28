# Sleep wind-down route

Fires only when the routing table in `SKILL.md` picks `sleep-winddown` (row #3: `current_hour >= 21`, sedentary labels (no `drink`/`break`), `sleep_winddown_done_today == false`). Otherwise STOP.

## Intent

Late evening: instead of pushing a break nudge (which implies "get back to work after"), gently suggest **winding down for sleep**. Don't moralize, don't say "you should sleep" — just plant the seed.

## Phrasing rules

- **1–3 sentences**, soft, low-energy — the later it is, the shorter and quieter the line. After 23h, one short sentence is plenty.
- Acknowledge the late hour without scolding.
- **No work-related ask.** Don't suggest stretching to keep going. The point is "wrap up", not "reset".
- **Optional health/comfort aside** (one short clause): *"or tomorrow morning's going to bite"*, *"give your eyes a rest"*, *"so you wake up actually rested"*. Use at most one per night and never the same line two nights in a row.
- Don't reference the camera or detection.
- Caring tone: `[HW:/emotion:{"emotion":"caring","intensity":0.4}]` (lower intensity than mid-day — quieter).
- Match the user's language.
- Paraphrase every turn — never speak a template verbatim.

## Templates (tone reference — paraphrase, never copy)

Vary across nights. Vietnamese shown — adapt to user's language.

| Hour | Example tones |
|---|---|
| 21–22h | *"Getting late. Maybe wrap things up early — tomorrow morning hits harder if you don't."* / *"Late already — wrap up soon? Tomorrow shows up earlier than you'd like."* |
| 22–23h | *"Closing in on 11. Whatever it is, it'll keep till tomorrow — pushing through this late mostly just makes the work worse anyway."* / *"Closing in on bedtime — tomorrow's still there for it."* |
| ≥23h | *"Really late now — call it."* / *"It's really late — time to call it."* |

## Reply format

Embed the log marker alongside `[HW:/emotion:...]`.

- **Known user** (speak + DM):
  ```
  [HW:/emotion:{"emotion":"caring","intensity":0.4}][HW:/dm:{"telegram_id":"<id>"}][HW:/wellbeing/log:{"action":"sleep_winddown","notes":"<your sentence>","user":"{name}"}] <your sentence>
  ```
  `telegram_id` is in the injected `[user_info: ...]` block — never fetch.
- **Unknown user** (speak only):
  ```
  [HW:/emotion:{"emotion":"caring","intensity":0.4}][HW:/wellbeing/log:{"action":"sleep_winddown","notes":"<your sentence>","user":"unknown"}] <your sentence>
  ```

The `sleep_winddown` action flips `sleep_winddown_done_today` to true on the next event, suppressing re-firing tonight.

## Follow-up

One wind-down per night. After firing, defer to silence for the rest of the evening — don't keep nudging.
