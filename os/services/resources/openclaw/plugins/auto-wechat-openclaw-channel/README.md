# auto-wechat-openclaw-channel

OpenClaw WeChat channel plugin that uses **Gewechat** as the adapter layer.

It exposes a simple webhook:

WeChat → Gewechat → Webhook adapter → OpenClaw → LLM/Agents → Gewechat API → WeChat

The adapter:

- Receives webhook events from Gewechat
- Sends the message into OpenClaw
- Sends the reply back to Gewechat

## Installation

```bash
openclaw plugins install auto-wechat-openclaw-channel
```

Or via npm:

```bash
npm install auto-wechat-openclaw-channel
```

## Configuration

In `~/.openclaw/openclaw.json`:

```json
{
  "channels": {
    "auto-wechat-openclaw-channel": {
      "enabled": true,
      "gewechatBaseUrl": "http://localhost:2531",
      "webhookPort": 3002,
      "botMention": "@bot"
    }
  }
}
```

- `gewechatBaseUrl`: Base URL of your Gewechat service
- `webhookPort`: Local HTTP port where the adapter listens for `/wechat/webhook`
- `botMention`: Trigger keyword; only messages containing this will be sent to OpenClaw

## Message flow

1. Gewechat calls `POST /wechat/webhook` with payload:

   ```json
   {
     "type": "text",
     "fromWxid": "wxid_xxx",
     "roomWxid": "room_123",
     "content": "@bot hello"
   }
   ```

2. Adapter checks `botMention` and, if matched, calls OpenClaw:

   ```ts
   const reply = await openclaw.sendChat(content, sessionId);
   ```

3. Adapter sends reply back via Gewechat:

   ```ts
   await gewechat.sendText(roomWxid || fromWxid, reply);
   ```

## CLI

The plugin registers:

```bash
openclaw auto-wechat info
```

This prints a short hint about configuring `channels.auto-wechat-openclaw-channel` in `openclaw.json`.

