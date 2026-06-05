import fetch from "node-fetch";

export interface GewechatConfig {
  baseUrl: string;
}

export interface GewechatIncomingMessage {
  type: string;
  fromWxid: string;
  roomWxid?: string;
  content: string;
}

export class GewechatService {
  private readonly baseUrl: string;

  constructor(config: GewechatConfig) {
    this.baseUrl = config.baseUrl.replace(/\/$/, "");
  }

  async sendText(toWxid: string, content: string): Promise<void> {
    const url = `${this.baseUrl}/v2/api/message/send`;
    await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        toWxid,
        content,
      }),
    });
  }
}

