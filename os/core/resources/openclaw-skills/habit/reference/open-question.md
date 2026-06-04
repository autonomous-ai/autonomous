# Flow E — Open Habit Question

Triggered when the user explicitly asks about a person's habits, patterns, or routines:
- *"What are Leo's habits?"*, *"Bạn biết thói quen của Chloe không?"*
- *"Has Leo been keeping to his routine?"*, *"Notice anything about my patterns?"*

## Step 1 — Run Flow A

Always invoke `reference/build-patterns.md` first to refresh `patterns.json` (the freshness guard makes this cheap).

## Step 2 — Pick reply mode

Decide based on what Flow A returned **and** the on-disk `patterns.json` mtime:

| Flow A returned | `patterns.json` mtime | Reply mode |
|---|---|---|
| `days_observed >= 3` AND ≥1 moderate/strong pattern | any | **Pattern** |
| `insufficient_data` OR all patterns weak OR <2 patterns | ≤ 3 days old (or missing) | **Narrative** |
| `insufficient_data` | > 3 days old | **Honest-gap** |

### Pattern mode

Name the 2–3 strongest patterns with concrete hour and frequency framing. Concrete numbers (hours, "most days", "3 of last 7") ARE allowed in the spoken reply for this flow — they are not allowed in the nudge OUTPUT RULE, but Flow E overrides that rule (see below).

### Narrative mode

Patterns aren't ready, so summarise raw activity instead of hedging. Read the last 7 days of wellbeing JSONL directly:

```bash
ls /root/local/users/{name}/wellbeing/*.jsonl | sort | tail -7
cat /root/local/users/{name}/wellbeing/YYYY-MM-DD.jsonl   # for each
```

Pick out 2–4 distinct (action, day, hour) facts and weave them with dates: which days the user was active, what time they were at the computer, when they drank water, late-night sessions, etc. End with an honest line: *"not enough days yet to call it a habit, but that's what I've seen."*

### Honest-gap mode

Existing `patterns.json` is more than 3 days old AND today's data is insufficient. Do **NOT** recite the stale patterns as if they are current — the freshness guard preserves them on disk even when they no longer reflect reality. Acknowledge the gap:

> *"I haven't seen [name] much in the last while — only one short session on May 4. The patterns I had from two weeks ago are stale, so I'd rather not guess."*

## Output rule for Flow E

Overrides the nudge OUTPUT RULE in `SKILL.md`:

- 2–4 sentence reply allowed
- Concrete dates, hours, and approximate frequencies are permitted in the spoken text
- Still no raw timestamps, no JSON, no internal computation traces
