---
name: input-branching
description: Understand voice input from the realtime voice agent pipeline. Messages may contain [voice-instruction]/[transcript] (delegated with context), [HANDLED]/[REPLY] (chit-chat already spoken), or plain text. Route and respond accordingly.
---

# Input Branching — Realtime Voice Agent Prefixes

Voice input passes through a realtime voice agent (Gemini Live / OpenAI Realtime) before reaching you. The agent decides whether to handle the utterance as chit-chat or delegate it to you for full processing. The message format tells you which path was taken.

## Message formats

### 1. Delegated with voice instruction

```
[voice-instruction] Turn on the desk light to warm white
[transcript] turn on the light please warm
```

The realtime agent decided this needs the main system and provided a summarized instruction. Two tags are present:

| Tag | Meaning |
|---|---|
| `[voice-instruction]` | The realtime agent's interpretation/summary of the user's request — cleaner and more actionable than raw STT. **Use this as the primary input for processing.** |
| `[transcript]` | The raw STT transcript with speaker decoration. May be noisy, incomplete, or in the wrong language — STT is locked to one language while the realtime voice agent understands multiple languages. Use as supplementary context only. |

**Process the `[voice-instruction]` as the user's request.** Run tools, call APIs, reply as usual. The instruction is what the realtime agent understood from the audio — it's typically more accurate than the raw transcript.

### 2. Delegated without instruction (fallback)

```
play some jazz music
```

No prefix. The realtime agent delegated but didn't provide an instruction (older flow or instruction was empty). **Process the message as-is** — same as a normal voice event.

### 3. Handled by realtime agent (chit-chat)

```
[HANDLED] Hey, how's it going?
[REPLY] I'm doing great! How about you?
```

The realtime agent already answered via TTS. The user heard the reply.

| Tag | Meaning |
|---|---|
| `[HANDLED]` | The user's original message. Already spoken to by the realtime model. |
| `[REPLY]` | The realtime model's response text. Already spoken via TTS. |

**Do NOT reply with speech.** The user already heard the answer. Instead:

- **Update context** — note the conversation happened (for memory, mood tracking, habit awareness).
- **Log if relevant** — if the exchange reveals mood, intent, or information worth tracking, update the appropriate logs (mood, wellbeing, habit).
- **Stay silent** — respond with `NO_REPLY`. Do not echo, paraphrase, or add to what was already said.
- **Exception** — if the `[REPLY]` content is clearly wrong, harmful, or incomplete in a way that matters, you may speak up. This should be rare.

## Rules

1. **Never strip or echo the tags in your reply.** They are metadata for routing, not user-facing text.
2. **`[voice-instruction]` is the preferred input.** When present, use it over `[transcript]` for processing the request.
3. **`[HANDLED]` messages are informational.** Treat them as a notification that a conversation turn already happened, not as a request for action.
4. **The realtime agent handles casual conversation only.** Anything requiring tools, device control, memory, scheduling, or skills is always delegated.
5. **Both paths include speaker decoration.** The user's identity (speaker name or "Unknown Speaker") is always present regardless of which path was taken.
6. **Do not duplicate TTS.** If `[HANDLED]` is present, respond `NO_REPLY` to stay silent.

## Examples

### Delegated with instruction — process the instruction

Input:
```
[voice-instruction] Play jazz music on Spotify
[transcript] play some jazz please on spotify
```

Action: Use "Play jazz music on Spotify" as the request. Route to music skill.

### Delegated without instruction — process as-is

Input:
```
turn on the lights
```

Action: Route to LED/scene skill, turn on lights, reply with confirmation.

### Handled — stay silent

Input:
```
[HANDLED] What's the weather like?
[REPLY] It looks pretty nice outside today!
```

Action: Note the exchange. Respond `NO_REPLY`. The user already heard the answer.

### Handled — mood signal worth logging

Input:
```
[HANDLED] I'm so tired today
[REPLY] That sounds rough. Maybe take a short break?
```

Action: Log mood signal (fatigue) via mood/wellbeing. Respond `NO_REPLY`.
