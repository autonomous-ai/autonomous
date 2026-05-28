import express from "express";
import type { Express } from "express";
import type { GewechatIncomingMessage } from "../services/gewechat.js";
import { GewechatService } from "../services/gewechat.js";
import { OpenClawService } from "../services/openclaw.js";

export interface AdapterConfig {
  gewechatBaseUrl: string;
  webhookPort: number;
  botMention: string;
  openclawBaseUrl: string;
}

export function createAdapterServer(config: AdapterConfig): Express {
  const app = express();
  app.use(express.json());

  const gewechat = new GewechatService({ baseUrl: config.gewechatBaseUrl });
  const openclaw = new OpenClawService({ baseUrl: config.openclawBaseUrl });

  app.post("/wechat/webhook", async (req, res) => {
    const body = req.body as GewechatIncomingMessage;
    const msg = body.content ?? "";

    if (!msg.includes(config.botMention)) {
      return res.send("ignored");
    }

    const userId = body.roomWxid || body.fromWxid;
    const sessionId = `wechat:${userId}`;

    try {
      const reply = await openclaw.sendChat(msg, sessionId);
      if (reply) {
        await gewechat.sendText(userId, reply);
      }
      res.send("ok");
    } catch (err) {
      // eslint-disable-next-line no-console
      console.error("[auto-wechat-openclaw-channel] webhook error", err);
      res.status(500).send("error");
    }
  });

  return app;
}

