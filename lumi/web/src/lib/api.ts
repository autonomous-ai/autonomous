import camelcaseKeys from "camelcase-keys";
import type { NetworkItem, SetupRequest } from "@/types";

const API_BASE =
  import.meta.env.VITE_API_BASE ??
  import.meta.env.VITE_NETWORK_API ??
  import.meta.env.VITE_API_URL ??
  "";

/** 0 = error, 1 = success (matches backend JSONReponseStatus) */
export type JSONResponseStatus = 0 | 1;

export interface JSONResponse<T = unknown> {
  status: JSONResponseStatus;
  message: string | null;
  data: T;
}

// Bearer token attached to every /api/* request hitting an admin-gated route
// in the Go server (`adminAuthMiddleware`). The value mirrors
// `config.json::llm_api_key` and is bootstrapped from GET /api/device/config
// on first page load until a proper login UI exists.
//
// sessionStorage so a hard reload keeps the token without a refetch.
const TOKEN_STORAGE_KEY = "lumi_api_token";
let apiToken: string =
  typeof window !== "undefined" ? sessionStorage.getItem(TOKEN_STORAGE_KEY) ?? "" : "";

export function setApiToken(token: string): void {
  apiToken = token ?? "";
  if (typeof window === "undefined") return;
  if (apiToken) sessionStorage.setItem(TOKEN_STORAGE_KEY, apiToken);
  else sessionStorage.removeItem(TOKEN_STORAGE_KEY);
}

export function getApiToken(): string {
  return apiToken;
}

/** Append ?token=<key> to a URL — used for SSE/EventSource where native
 *  EventSource cannot set custom headers. */
export function withApiToken(url: string): string {
  if (!apiToken) return url;
  const sep = url.includes("?") ? "&" : "?";
  return `${url}${sep}token=${encodeURIComponent(apiToken)}`;
}

/** Build a /api/hardware/<path> URL with the bearer token baked in as
 *  ?token=, for places where headers can't be set: <img src>, <a href>,
 *  window.open, MJPEG stream. Pair with hardwareProxy + adminAuthMiddleware
 *  on the Go side (?token= fallback). */
export function hwUrl(path: string): string {
  const url = `/api/hardware${path}`;
  return apiToken ? withApiToken(url) : url;
}

// Global fetch interceptor: attaches Authorization: Bearer to any request
// that targets /api/hardware/* (the Go hardware proxy). Avoids refactoring
// ~65 raw fetch sites in the monitor — they keep their existing
// fetch(`${HW}/...`) shape, and HW now points at the proxy.
if (typeof window !== "undefined" && !(window as unknown as { __lumiFetchPatched?: boolean }).__lumiFetchPatched) {
  const origFetch = window.fetch.bind(window);
  window.fetch = function patchedFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
    let url = "";
    if (typeof input === "string") url = input;
    else if (input instanceof URL) url = input.toString();
    else url = (input as Request).url;

    // Only intercept the hardware proxy path. Anything else is left alone
    // (apiRequest already handles its own header set; third-party code
    // shouldn't be affected).
    if (url.includes("/api/hardware/") && apiToken) {
      const headers = new Headers(init?.headers);
      if (!headers.has("Authorization")) {
        headers.set("Authorization", `Bearer ${apiToken}`);
      }
      return origFetch(input, { ...init, headers });
    }
    return origFetch(input, init);
  };
  (window as unknown as { __lumiFetchPatched?: boolean }).__lumiFetchPatched = true;
}

async function apiRequest<T>(url: string, options?: RequestInit): Promise<T> {
  const headers = new Headers(options?.headers);
  if (apiToken && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${apiToken}`);
  }
  const res = await fetch(url, { ...options, headers });
  const json = (await res.json()) as JSONResponse<T>;
  if (json.status !== 1) {
    const msg =
      typeof json.message === "string" ? json.message : res.ok ? "Request failed" : res.statusText;
    throw new Error(msg);
  }
  return json.data;
}

/**
 * Converts object keys from snake_case to camelCase (uses camelcase-keys).
 * Use for API responses that return snake_case keys.
 */
export function parseSnakeToCamel<T = Record<string, unknown>>(
  raw: Record<string, unknown>,
  options?: { deep?: boolean }
): T {
  return camelcaseKeys(raw as Record<string, unknown>, { deep: options?.deep ?? false }) as T;
}

export async function getNetworks(): Promise<NetworkItem[]> {
  return apiRequest<NetworkItem[]>(`${API_BASE}/api/network`);
}

export async function setupNetwork(ssid: string, password: string): Promise<string> {
  return apiRequest<string>(`${API_BASE}/api/network/setup`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ssid, password }),
  });
}

export async function setupDevice(body: SetupRequest): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/device/setup`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

export interface SetupStatus {
  phase: "idle" | "connecting" | "connected" | "failed";
  lan_ip: string;
  error: string;
}

/** Polled by Setup.tsx during the AP→STA transition. Returns the device's
 *  current setup phase plus the LAN IP once Wi-Fi is associated, so the web
 *  client can redirect the user to the new URL. */
export async function getSetupStatus(): Promise<SetupStatus> {
  return apiRequest<SetupStatus>(`${API_BASE}/api/device/setup/status`);
}

export async function checkInternet(): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/network/check-internet`);
}


export async function getSetup(): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/setup`);
}

export interface DeviceConfig {
  channel: string;
  telegram_bot_token: string;
  telegram_user_id: string;
  slack_bot_token: string;
  slack_app_token: string;
  slack_user_id: string;
  discord_bot_token: string;
  discord_guild_id: string;
  discord_user_id: string;
  llm_api_key: string;
  llm_model: string;
  llm_base_url: string;
  llm_disable_thinking: boolean;
  deepgram_api_key: string;
  stt_api_key: string;
  tts_api_key: string;
  stt_base_url: string;
  tts_base_url: string;
  stt_language: string;
  stt_model: string;
  tts_provider: string;
  tts_voice: string;
  device_id: string;
  mac: string;
  network_ssid: string;
  network_password: string;
  mqtt_endpoint: string;
  mqtt_username: string;
  mqtt_password: string;
  mqtt_port: number;
  fa_channel: string;
  fd_channel: string;
}

export async function getTTSVoices(provider?: string, lang?: string): Promise<string[]> {
  const qs = new URLSearchParams();
  if (provider) qs.set("provider", provider);
  if (lang) qs.set("lang", lang);
  const params = qs.toString() ? `?${qs.toString()}` : "";
  return apiRequest<string[]>(`${API_BASE}/api/device/voices${params}`);
}

export async function getTTSProviders(): Promise<string[]> {
  return apiRequest<string[]>(`${API_BASE}/api/device/tts-providers`);
}

export interface TestTTSOptions {
  text?: string;
  /** BCP-47 stt_language code; picks a friendly demo phrase in that language. */
  lang?: string;
  provider?: string;
  ttsApiKey?: string;
  ttsBaseUrl?: string;
  llmApiKey?: string;
  llmBaseUrl?: string;
}

const TTS_DEMO_PHRASES: Record<string, string> = {
  en: "[laugh] Hey! How are you doing today?",
  vi: "[laugh] Chào bạn, hôm nay bạn thế nào?",
  "zh-CN": "[laugh] 嗨，你今天怎么样？",
  "zh-TW": "[laugh] 嗨，你今天怎麼樣？",
};

function demoPhraseFor(lang?: string): string {
  if (!lang) return TTS_DEMO_PHRASES.en;
  return TTS_DEMO_PHRASES[lang] || TTS_DEMO_PHRASES.en;
}

export async function testTTSVoice(voice: string, opts: TestTTSOptions = {}): Promise<void> {
  const apiKey = (opts.ttsApiKey && opts.ttsApiKey.trim()) || opts.llmApiKey || "";
  const baseUrl = (opts.ttsBaseUrl && opts.ttsBaseUrl.trim()) || opts.llmBaseUrl || "";
  await fetch("/hw/voice/speak", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      text: opts.text || demoPhraseFor(opts.lang),
      voice,
      provider: opts.provider || undefined,
      tts_api_key: apiKey || undefined,
      tts_base_url: baseUrl || undefined,
    }),
  });
}

export async function getDeviceConfig(): Promise<DeviceConfig> {
  const cfg = await apiRequest<DeviceConfig>(`${API_BASE}/api/device/config`);
  // Bootstrap the admin bearer token so subsequent admin /api/* calls succeed.
  // GET /api/device/config is intentionally kept open in Go server until web
  // has a login UI — this is the transition path.
  if (cfg?.llm_api_key) {
    setApiToken(cfg.llm_api_key);
  }
  return cfg;
}

export async function updateDeviceConfig(body: Partial<DeviceConfig> & { password?: string; ssid?: string }): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/device/config`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}