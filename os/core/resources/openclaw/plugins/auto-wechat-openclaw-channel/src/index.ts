import type { OpenClawPluginApi } from "openclaw/plugin-sdk";
import { emptyPluginConfigSchema } from "openclaw/plugin-sdk";
import { createAdapterServer, type AdapterConfig } from "./adapter/server.js";

const meta = {
  id: "auto-wechat-openclaw-channel",
  label: "WeChat (Gewechat)",
  selectionLabel: "WeChat (Gewechat)",
  detailLabel: "WeChat (Gewechat)",
  docsPath: "/channels/wechat-gewechat",
  docsLabel: "auto-wechat-openclaw-channel",
  blurb: "WeChat channel via Gewechat adapter",
  systemImage: "message.fill",
  order: 86,
};

const channelId = "auto-wechat-openclaw-channel";

const wechatGewechatChannel = {
  id: channelId,
  meta,
  capabilities: {
    chatTypes: ["direct", "group"] as ("direct" | "group")[],
    reactions: false,
    threads: false,
    media: false,
    nativeCommands: false,
    blockStreaming: false,
  },
  reload: {
    configPrefixes: [`channels.${channelId}`],
  },
  config: {
    listAccountIds: (cfg: any) => {
      const accounts = cfg.channels?.[channelId]?.accounts;
      if (accounts && typeof accounts === "object") {
        return Object.keys(accounts);
      }
      return ["default"];
    },
    resolveAccount: (cfg: any, accountId: string) => {
      const accounts = cfg.channels?.[channelId]?.accounts;
      const account = accounts?.[accountId ?? "default"];
      return account ?? { accountId: accountId ?? "default" };
    },
    resolveAllowFrom: () => ["*"],
  },
  auth: {
    login: async ({ runtime }: { runtime: any }) => {
      runtime.log("[auto-wechat-openclaw-channel] Gewechat 模式无需额外登录，请在配置中设置 gewechatBaseUrl。");
    },
  },
  outbound: {
    deliveryMode: "direct" as const,
    sendText: async () => {
      return { ok: true };
    },
  },
  status: {
    buildAccountSnapshot: () => {
      return { running: true };
    },
  },
  gateway: {
    startAccount: async (ctx: any) => {
      const { cfg, abortSignal, log } = ctx;
      const channelCfg = cfg.channels?.[channelId] ?? {};

      const gewechatBaseUrl: string =
        channelCfg.gewechatBaseUrl || "http://localhost:2531";
      const webhookPort: number =
        Number(channelCfg.webhookPort || 3002);
      const botMention: string = channelCfg.botMention || "@bot";

      const openclawPort =
        cfg?.gateway?.port != null ? String(cfg.gateway.port) : "3001";
      const openclawBaseUrl = `http://localhost:${openclawPort}`;

      const adapterConfig: AdapterConfig = {
        gewechatBaseUrl,
        webhookPort,
        botMention,
        openclawBaseUrl,
      };

      const app = createAdapterServer(adapterConfig);

      const server = app.listen(webhookPort, () => {
        log?.info(
          `[auto-wechat-openclaw-channel] webhook server listening on port ${webhookPort}`,
        );
        ctx.setStatus({ running: true });
      });

      abortSignal.addEventListener("abort", () => {
        log?.info("[auto-wechat-openclaw-channel] stopping webhook server");
        server.close(() => {
          ctx.setStatus({ running: false });
        });
      });
    },
    stopAccount: async (ctx: any) => {
      ctx.setStatus({ running: false });
    },
  },
};

const index = {
  id: channelId,
  name: "WeChat via Gewechat",
  description: "WeChat channel plugin that connects to OpenClaw via a Gewechat adapter.",
  configSchema: emptyPluginConfigSchema(),
  register(api: OpenClawPluginApi) {
    api.registerChannel({ plugin: wechatGewechatChannel as any });

    api.registerCli(({ program }) => {
      const wechat = program
        .command("auto-wechat")
        .description("WeChat (Gewechat) channel management");

      wechat
        .command("info")
        .description("Show basic configuration hints for Gewechat adapter")
        .action(async () => {
          // eslint-disable-next-line no-console
          console.log(
            "Configure channels.auto-wechat-openclaw-channel in openclaw.json with gewechatBaseUrl, webhookPort, and botMention.",
          );
        });
    });
  },
};

export default index;

