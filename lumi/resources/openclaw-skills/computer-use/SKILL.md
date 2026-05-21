---
name: computer-use
description: Control the user's Mac via the paired Lumi Buddy companion app. Two paths — (A) fast inline HW markers for known intents (open/close apps, URLs, type into focused field, keyboard shortcuts, named-button clicks); (B) the `computer` tool (registered by the `lumi-computer-use` OpenClaw plugin) for vision-driven tasks (screenshot, click at coordinate, drag, read screen text). Always try Path A first; only call the `computer` tool when the task requires actually seeing the screen. Use when the user asks Lumi to do something on their COMPUTER (e.g. "open Chrome", "go to Gmail", "join Meet at <code>", "close Slack", "type … into my Mac", "copy … to clipboard", "click the blue button on my screen", "what's on my Mac right now?"). Do NOT use for hardware control of the lamp itself (LED, scene, emotion, music, servo) — those are separate skills. Do NOT use if no Mac is paired (the lamp web UI shows pairing status under the Buddy card).
---

# Computer Use (Mac via Lumi Buddy)

## Quick Start

Lumi Buddy is a small macOS app the user installs on their Mac. Once paired with the lamp via the Buddy card on the Monitor web UI, it lets Lumi launch apps, open URLs, type text, fire keyboard shortcuts, and click UI elements **on the user's actual Mac**.

This skill emits inline markers that Lumi fires asynchronously while TTS speaks the confirmation:

```
[HW:/buddy/exec/<action>:<params-json>]
```

The marker hits `/api/buddy/exec/<action>` on the lamp, which dispatches over the buddy's persistent WebSocket. Fire-and-forget — no response is awaited (TTS continues immediately).

## Workflow

1. Determine the user's intent and pick one or more actions from the table below.
2. Build the marker(s) — flat params JSON only (no nested objects).
3. Place markers at the **start of the reply**, then add a short confirmation that TTS will speak.
4. If no Mac is paired, say so and tell the user to set it up via the Lumi web UI Buddy card.

### Two paths — pick the right one

The skill has **two complementary surfaces**. Pick based on the task.

**Path A — HW markers (this skill).** Fire-and-forget, no return value, ~100% reliable when the action matches a known intent. Use for: launching apps, opening URLs (including deep links), typing into the focused field, keyboard shortcuts, named-button clicks (when Accessibility tree exposes the label). Examples are listed below.

**Path B — `computer` tool (separate tool, registered by the `lumi-computer-use` OpenClaw plugin).** Native tool-call loop. Lets you take screenshots, observe the screen, and click at coordinates you compute yourself. Use ONLY when the task requires actually *seeing* the screen:

- "Click that blue button in the toolbar" (no stable label / not a known intent)
- "Read me what the dialog says" / "What's on screen now?"
- "Drag the slider to the middle" / multi-step UI navigation
- Any task where a deep link or marker action cannot do the job

Path B is slower (multiple Anthropic round-trips, screenshots upload) and less reliable (~20-30% per multi-step task on uncommon UI). **Always try Path A first.** Only invoke the `computer` tool when no marker action covers the request, or when the user explicitly asks Lumi to "look at the screen".

When you use Path B, call `computer({action: "screenshot"})` first to see the screen, then issue one action per turn (click_at / type / key / scroll), screenshot again to verify, and stop after at most 6-8 iterations. The tool description has the full action reference.

## Examples

Input: "Open Chrome on my computer"
Output: `[HW:/buddy/exec/open_app:{"app":"Google Chrome"}]` Opening Chrome on your Mac.

Input: "Open Gmail"
Output: `[HW:/buddy/exec/open_url:{"url":"https://gmail.com"}]` Opening Gmail.

Input: "Join the Meet at abc-defg-hij"
Output: `[HW:/buddy/exec/open_url:{"url":"https://meet.google.com/abc-defg-hij"}]` Joining the meeting.

Input: "Search Google for 'best pho Saigon'"
Output: `[HW:/buddy/exec/open_url:{"url":"https://www.google.com/search?q=best+pho+Saigon"}]` Searching Google.

Input: "Close Slack"
Output: `[HW:/buddy/exec/close_app:{"app":"Slack"}]` Closed Slack.

Input: "Open Spotify"
Output: `[HW:/buddy/exec/open_app:{"app":"Spotify"}]` Opening Spotify.

Input: "Type 'hello world' into the active field"
Output: `[HW:/buddy/exec/type_text:{"text":"hello world"}]` Typed.

Input: "Open Spotlight"
Output: `[HW:/buddy/exec/key_combo:{"keys":["cmd","space"]}]` Spotlight open.

Input: "Close the current Chrome tab"
Output: `[HW:/buddy/exec/key_combo:{"keys":["cmd","w"]}]` Tab closed.

Input: "Remind me in 5 minutes about the meeting"
Output: `[HW:/buddy/exec/notification:{"title":"Meeting in 5 min","body":"Get ready"}]` Reminder set.

Input: "Copy leo@example.com to my clipboard"
Output: `[HW:/buddy/exec/write_clipboard:{"text":"leo@example.com"}]` Copied to clipboard.

Input: "Click the Submit button"
Output: `[HW:/buddy/exec/click_button:{"label":"Submit"}]` Clicked.

Input: "Hello" / "What time is it?"
Output: Do NOT use this skill. Reply normally.

Input: "Turn the lamp yellow"
Output: Do NOT use this skill — use **led-control** skill instead.

Input: "Reading mode" / "Make it cozy"
Output: Do NOT use this skill — use **scene** skill instead.

## Available actions

### `open_app` — launch a macOS app

```
[HW:/buddy/exec/open_app:{"app":"Google Chrome"}]
```
- `app` (required): app display name (e.g. "Google Chrome", "Spotify", "Calculator") OR bundle id (e.g. "com.google.Chrome").

### `close_app` — quit a macOS app

```
[HW:/buddy/exec/close_app:{"app":"Slack"}]
```
- `app` (required): app display name. Runs AppleScript `tell app to quit` under the hood; macOS may show a per-app Automation prompt on first close.

### `open_url` — open a URL in the default browser

```
[HW:/buddy/exec/open_url:{"url":"https://gmail.com"}]
```
- `url` (required): full URL with `https://`.
- `browser` (optional): `"chrome"`, `"safari"`, `"firefox"`, `"arc"`, `"edge"`, `"brave"`. Omit → default browser.

Tip: many apps expose deep links — use them instead of clicking through UI:
- Gmail compose: `https://mail.google.com/mail/?view=cm&to=X&subject=...&body=...`
- Google search: `https://www.google.com/search?q=...`
- Meet room: `https://meet.google.com/<id>`
- YouTube search: `https://www.youtube.com/results?search_query=...`
- GitHub repo: `https://github.com/<owner>/<repo>`

### `type_text` — type text into the focused field

```
[HW:/buddy/exec/type_text:{"text":"hello world"}]
```
- `text` (required): text to type.
- `delay_ms` (optional): per-character delay, default 15.
- Needs macOS Accessibility permission (one-time grant in System Settings).

### `key_combo` — fire a keyboard shortcut

```
[HW:/buddy/exec/key_combo:{"keys":["cmd","space"]}]
```
- `keys` (required): array of strings. Modifiers: `"cmd"`/`"command"`, `"shift"`, `"opt"`/`"option"`/`"alt"`, `"ctrl"`/`"control"`, `"fn"`. Plus exactly one key: `"a"`–`"z"`, `"0"`–`"9"`, `"return"`, `"escape"`, `"tab"`, `"space"`, `"delete"`, `"left"`/`"right"`/`"up"`/`"down"`, `"f1"`–`"f12"`, etc.
- Needs Accessibility permission.

### `notification` — show a macOS desktop notification

```
[HW:/buddy/exec/notification:{"title":"Meeting in 5","body":"Get ready"}]
```
- `title` (required), `body` (optional).
- First call may prompt for notification permission.

### `write_clipboard` — set clipboard text

```
[HW:/buddy/exec/write_clipboard:{"text":"some text"}]
```
- `text` (required). User can paste with `Cmd+V` afterwards.

### `click_button` — Accessibility-based click by label

```
[HW:/buddy/exec/click_button:{"label":"Submit"}]
```
- `label` (required): visible button text. Matches via macOS Accessibility API.
- `app` (optional): restrict search to a specific app (display name or bundle id).
- Works well for **native apps** (Settings, Finder, Notes, Calculator). For web content (Chrome / Safari), coverage is inconsistent — Chrome may not expose all button labels in its accessibility tree.

## Combining actions

Markers fire in order. Useful patterns:

**Send a tweet** — open compose URL, type text, submit:
```
[HW:/buddy/exec/open_url:{"url":"https://twitter.com/compose/tweet"}][HW:/buddy/exec/type_text:{"text":"hello"}][HW:/buddy/exec/key_combo:{"keys":["cmd","return"]}]
```
*Caveat:* markers fire sequentially but do NOT wait for page load between steps. If the URL takes time to load, the `type_text` may land in the wrong field. Prefer a single-marker action when timing is uncertain.

**Spotlight → open Notes**:
```
[HW:/buddy/exec/key_combo:{"keys":["cmd","space"]}][HW:/buddy/exec/type_text:{"text":"Notes"}][HW:/buddy/exec/key_combo:{"keys":["return"]}]
```

## Error handling

- **No buddy paired** (`no buddy connected`): respond "No Mac is paired with the lamp yet. Open the Monitor page → Buddy card to pair one." Do NOT fire any markers.
- **Timeout / connection error**: respond "I couldn't reach your Mac — the Buddy app may be offline."
- **Unknown action**: not possible if you only use the actions listed above. Stick to the table.

## Rules

- **Markers must appear at the START of the reply**, before the TTS sentence. Lumi parses and strips them before reading.
- **No nested JSON** in marker params (the marker regex doesn't support nested `{}`). All actions above take flat params.
- **One action per marker.** Don't try to batch multiple ops into a single marker body.
- **Don't use this skill for lamp hardware** (LED, scene, emotion, audio playback on the lamp speaker, servo, display) — those are separate skills.
- **Don't fire `screenshot`, `click_at`, `scroll`, `mouse_move`, `drag`, `cursor_pos`** through inline markers — those need return values. Use the **`computer` tool** (Path B above) when the task requires seeing the screen. The tool is provided by the `lumi-computer-use` OpenClaw plugin and exposes the Anthropic-style action set (screenshot, left_click, type, key, scroll, …).
- **Match the user's input language** in the TTS confirmation (English in, English out; Vietnamese in, Vietnamese out). Keep the TTS reply to one short sentence.
- **If the user asks for lamp-side actions** ("turn yellow", "play music", "show emotion"), redirect to the appropriate skill (`led-control`, `music`, `emotion`, `scene`).

## Output template

```
[HW:/buddy/exec/<action>:<flat-params-json>] <short confirmation sentence>
```

Examples:
- `[HW:/buddy/exec/open_url:{"url":"https://gmail.com"}] Opening Gmail.`
- `[HW:/buddy/exec/close_app:{"app":"Slack"}] Closed Slack.`
- `[HW:/buddy/exec/key_combo:{"keys":["cmd","tab"]}] Switched app.`
