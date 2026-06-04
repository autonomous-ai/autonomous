# Flow B — Habit match (helper for wellbeing/SKILL.md Step 3b)

`wellbeing/SKILL.md` calls this after a threshold nudge fires. Read `patterns.json` directly (no API), find the matching habit for the nudge action, and let wellbeing weave the context into Step 4 phrasing. Habit itself never speaks or writes a `nudge_*` row — wellbeing owns the nudge.

## Steps

1. Get current time (hour + minute).
2. For the action wellbeing is about to nudge (`drink` or `break`), find any habit entry with the same `action` and `strength` moderate+ (`frequency >= 0.5`).
3. Is `now` within `typical_hour:typical_minute ± window_minutes`?
4. If yes → return the matched habit so wellbeing can phrase as *"you usually drink around now…"*. If no match → wellbeing falls back to its generic phrasing table.

## Window sizes by action

| Action | Suggested window |
|---|---|
| `drink` | ±30 min |
| `break` | ±30 min |
| `enter` (arrival) | ±45 min |
| Sedentary labels | ±60 min |
| `meal` | ±45 min |
| `coffee` | ±30 min |
| `sleep` | ±30 min |
| `exercise` | ±60 min |
