import fetch from "node-fetch";

export interface OpenClawServiceConfig {
  baseUrl: string;
}

export class OpenClawService {
  private readonly baseUrl: string;

  constructor(config: OpenClawServiceConfig) {
    this.baseUrl = config.baseUrl.replace(/\/$/, "");
  }

  async sendChat(message: string, sessionId: string): Promise<string> {
    const url = `${this.baseUrl}/chat`;
    const res = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        message,
        sessionId,
      }),
    });

    if (!res.ok) {
      const body = await res.text().catch(() => "");
      throw new Error(`OpenClaw chat failed: ${res.status} ${body}`);
    }

    const data = (await res.json()) as { reply?: string };
    return data.reply ?? "";
  }
}

