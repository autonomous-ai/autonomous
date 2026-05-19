import { useEffect, useRef, useState, useCallback } from "react";
import { toast } from "sonner";
import { getDeviceConfig, updateDeviceConfig, getTTSVoices, getTTSProviders } from "@/lib/api";
import type { DeviceConfig } from "@/lib/api";
import { useTheme } from "@/lib/useTheme";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import type { ChannelType } from "@/types";
import type { SectionId as SharedSectionId } from "@/hooks/setup/types";
import type { FaceOwner } from "@/hooks/setup/useFaceEnroll";
import { C } from "@/components/setup/shared";
import { DeviceSection } from "@/components/setup/DeviceSection";
import { LLMSection } from "@/components/setup/LLMSection";
import { WifiSection } from "@/components/edit/WifiSection";
import { VoiceSection as EditVoiceSection } from "@/components/edit/VoiceSection";
import { FaceSection as EditFaceSection } from "@/components/edit/FaceSection";
import { TTSSection } from "@/components/edit/TTSSection";
import { STTSection, type SttProvider } from "@/components/edit/STTSection";
import { ChannelSection } from "@/components/edit/ChannelSection";
import { MqttSection } from "@/components/edit/MqttSection";
import { Wifi, UserCircle, Lamp, Brain, Volume2, MicVocal, MessageSquare, Globe, Link } from "lucide-react";

// Local subset of the shared SectionId — EditConfig uses `stt` (Language is
// rendered under id="stt"), not `language` / `deepgram` like Setup.
type SectionId = Extract<SharedSectionId, "device" | "wifi" | "llm" | "voice" | "face" | "tts" | "stt" | "channel" | "mqtt">;
const ICON_SIZE = 15;
const ALL_SECTIONS: { id: SectionId; label: string; icon: React.ReactNode; debugOnly?: boolean }[] = [
  { id: "device",   label: "Device",   icon: <Lamp size={ICON_SIZE} /> },
  { id: "wifi",     label: "Wi-Fi",    icon: <Wifi size={ICON_SIZE} /> },
  { id: "llm",      label: "AI Brain", icon: <Brain size={ICON_SIZE} /> },
  // Language above Lumi's Voice — picking the language determines which
  // voices sound natural, so the operator should set it first.
  // Language + Lumi's Voice are gated behind ?debug=true: typical operators
  // shouldn't see STT/TTS provider knobs.
  { id: "stt",      label: "Language", icon: <Globe size={ICON_SIZE} />, debugOnly: true },
  { id: "tts",      label: "Lumi's Voice", icon: <Volume2 size={ICON_SIZE} />, debugOnly: true },
  { id: "voice",    label: "My Voice", icon: <MicVocal size={ICON_SIZE} /> },
  { id: "face",     label: "Face",     icon: <UserCircle size={ICON_SIZE} /> },
  { id: "channel",  label: "Channels", icon: <MessageSquare size={ICON_SIZE} /> },
  { id: "mqtt",     label: "MQTT",     icon: <Link size={ICON_SIZE} /> },
];

const isDebugMode = () => new URLSearchParams(window.location.search).get("debug") === "true";

// Field / LockedField / LockedPasswordField / SectionCard moved to
// @/components/setup/shared. SkeletonBlock stays inline because EditConfig's
// version renders 4 stacked cards whereas Setup's renders just one.

function SkeletonBlock() {
  const bar = (w: string | number, h = 10) => (
    <div style={{ width: w, height: h, borderRadius: 6, background: C.surface, marginBottom: 10 }} />
  );
  return (
    <>
      {[1, 2, 3, 4].map((i) => (
        <div key={i} style={{ background: C.card, border: `1px solid ${C.border}`, borderRadius: 12, padding: "18px 20px", marginBottom: 16 }}>
          {bar(80, 8)}
          <div style={{ marginTop: 14 }}>{bar("100%", 32)}{bar("100%", 32)}</div>
        </div>
      ))}
    </>
  );
}

// ── main page ─────────────────────────────────────────────────────────────────

export default function EditConfig() {
  const [theme, toggleTheme, themeClass] = useTheme();
  const [loadingCfg, setLoadingCfg] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const debug = isDebugMode();
  const SECTIONS = debug ? ALL_SECTIONS : ALL_SECTIONS.filter((s) => !s.debugOnly);
  const [activeSection, setActiveSection] = useState<SectionId>(() => {
    const hash = window.location.hash.replace("#", "") as SectionId;
    return SECTIONS.some((s) => s.id === hash) ? hash : "device";
  });
  const contentRef = useRef<HTMLDivElement>(null);

  const activeSectionLabel = SECTIONS.find((s) => s.id === activeSection)?.label ?? "Settings";
  useDocumentTitle(["Settings", activeSectionLabel]);

  // form state
  const [ssid, setSsid] = useState("");
  const [password, setPassword] = useState("");
  const [deviceId, setDeviceId] = useState("");
  const [mac, setMac] = useState("");
  const [llmApiKey, setLlmApiKey] = useState("");
  const [llmUrl, setLlmUrl] = useState("");
  const [llmModel, setLlmModel] = useState("");
  const [llmDisableThinking, setLlmDisableThinking] = useState(false);
  const [deepgramApiKey, setDeepgramApiKey] = useState("");
  const [sttApiKey, setSttApiKey] = useState("");
  const [sttBaseUrl, setSttBaseUrl] = useState("");
  // STT provider: derived from saved config (deepgram if key present, else autonomous).
  // Default for fresh devices is "autonomous" — uses LLM endpoint as fallback.
  const [sttProvider, setSttProvider] = useState<SttProvider>("autonomous");
  // STT language drives model selection on the server (operators don't pick model directly).
  const [sttLanguage, setSttLanguage] = useState("");
  const [ttsApiKey, setTtsApiKey] = useState("");
  const [ttsBaseUrl, setTtsBaseUrl] = useState("");
  const [ttsProvider, setTtsProvider] = useState("openai");
  const [ttsProviders, setTtsProviders] = useState<string[]>([]);
  const [ttsVoice, setTtsVoice] = useState("alloy");
  const [ttsVoices, setTtsVoices] = useState<string[]>([]);
  const [channel, setChannel] = useState<ChannelType>("telegram");
  const [teleToken, setTeleToken] = useState("");
  const [teleUserId, setTeleUserId] = useState("");
  const [slackBotToken, setSlackBotToken] = useState("");
  const [slackAppToken, setSlackAppToken] = useState("");
  const [slackUserId, setSlackUserId] = useState("");
  const [discordBotToken, setDiscordBotToken] = useState("");
  const [discordGuildId, setDiscordGuildId] = useState("");
  const [discordUserId, setDiscordUserId] = useState("");
  const [mqttEndpoint, setMqttEndpoint] = useState("");
  const [mqttPort, setMqttPort] = useState("");
  const [mqttUsername, setMqttUsername] = useState("");
  const [mqttPassword, setMqttPassword] = useState("");
  const [faChannel, setFaChannel] = useState("");
  const [fdChannel, setFdChannel] = useState("");
  // Snapshot of MQTT fields that were already populated when config loaded.
  // Locks those fields against edits; fields blank at load remain editable.
  const [mqttLoaded, setMqttLoaded] = useState({
    endpoint: false, port: false, username: false,
    password: false, faChannel: false, fdChannel: false,
  });
  // Same idea for messaging-channel credentials. Already-saved values render
  // read-only with an inline "Edit" button to opt-in to changing them.
  const [channelLoaded, setChannelLoaded] = useState({
    teleToken: false, teleUserId: false,
    slackBotToken: false, slackAppToken: false, slackUserId: false,
    discordBotToken: false, discordGuildId: false, discordUserId: false,
  });
  const [wifiLoaded, setWifiLoaded] = useState({ ssid: false, password: false });
  const [llmLoaded, setLlmLoaded] = useState({ apiKey: false, baseUrl: false, model: false });
  const [ttsLoaded, setTtsLoaded] = useState({ apiKey: false, baseUrl: false });
  const [sttLoaded, setSttLoaded] = useState({ deepgram: false, apiKey: false, baseUrl: false });

  // Face owners — top-level state because both Voice and Face sections read
  // it. Section-local state (faceName, voiceLabel, etc.) lives in the section
  // components themselves.
  const [faceOwners, setFaceOwners] = useState<FaceOwner[]>([]);

  const loadFaceOwners = useCallback(async () => {
    try {
      const r = await fetch("/hw/face/owners").then((x) => x.json());
      if (Array.isArray(r?.persons)) setFaceOwners(r.persons);
    } catch {}
  }, []);

  useEffect(() => { loadFaceOwners(); }, [loadFaceOwners]);

  useEffect(() => {
    getDeviceConfig()
      .then((cfg: DeviceConfig) => {
        setSsid(cfg.network_ssid ?? "");
        setPassword(cfg.network_password ?? "");
        setDeviceId(cfg.device_id ?? "");
        setMac(cfg.mac ?? "");
        setLlmApiKey(cfg.llm_api_key ?? "");
        setLlmUrl(cfg.llm_base_url ?? "");
        setLlmModel(cfg.llm_model ?? "");
        setLlmDisableThinking(cfg.llm_disable_thinking ?? false);
        setDeepgramApiKey(cfg.deepgram_api_key ?? "");
        setSttApiKey(cfg.stt_api_key ?? "");
        setSttBaseUrl(cfg.stt_base_url ?? "");
        setSttProvider(cfg.deepgram_api_key ? "deepgram" : "autonomous");
        setSttLanguage(cfg.stt_language ?? "");
        setTtsApiKey(cfg.tts_api_key ?? "");
        setTtsBaseUrl(cfg.tts_base_url ?? "");
        setTtsProvider(cfg.tts_provider || "openai");
        setTtsVoice(cfg.tts_voice || "alloy");
        setChannel((cfg.channel as ChannelType) || "telegram");
        setTeleToken(cfg.telegram_bot_token ?? "");
        setTeleUserId(cfg.telegram_user_id ?? "");
        setSlackBotToken(cfg.slack_bot_token ?? "");
        setSlackAppToken(cfg.slack_app_token ?? "");
        setSlackUserId(cfg.slack_user_id ?? "");
        setDiscordBotToken(cfg.discord_bot_token ?? "");
        setDiscordGuildId(cfg.discord_guild_id ?? "");
        setDiscordUserId(cfg.discord_user_id ?? "");
        setMqttEndpoint(cfg.mqtt_endpoint ?? "");
        setMqttPort(cfg.mqtt_port ? String(cfg.mqtt_port) : "");
        setMqttUsername(cfg.mqtt_username ?? "");
        setMqttPassword(cfg.mqtt_password ?? "");
        setFaChannel(cfg.fa_channel ?? "");
        setFdChannel(cfg.fd_channel ?? "");
        setMqttLoaded({
          endpoint: !!cfg.mqtt_endpoint,
          port: !!cfg.mqtt_port,
          username: !!cfg.mqtt_username,
          password: !!cfg.mqtt_password,
          faChannel: !!cfg.fa_channel,
          fdChannel: !!cfg.fd_channel,
        });
        setChannelLoaded({
          teleToken: !!cfg.telegram_bot_token,
          teleUserId: !!cfg.telegram_user_id,
          slackBotToken: !!cfg.slack_bot_token,
          slackAppToken: !!cfg.slack_app_token,
          slackUserId: !!cfg.slack_user_id,
          discordBotToken: !!cfg.discord_bot_token,
          discordGuildId: !!cfg.discord_guild_id,
          discordUserId: !!cfg.discord_user_id,
        });
        setWifiLoaded({
          ssid: !!cfg.network_ssid,
          password: !!cfg.network_password,
        });
        setLlmLoaded({
          apiKey: !!cfg.llm_api_key,
          baseUrl: !!cfg.llm_base_url,
          model: !!cfg.llm_model,
        });
        setTtsLoaded({
          apiKey: !!cfg.tts_api_key,
          baseUrl: !!cfg.tts_base_url,
        });
        setSttLoaded({
          deepgram: !!cfg.deepgram_api_key,
          apiKey: !!cfg.stt_api_key,
          baseUrl: !!cfg.stt_base_url,
        });
      })
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoadingCfg(false));
    getTTSProviders().then(setTtsProviders).catch(() => {});
    getTTSVoices().then(setTtsVoices).catch(() => {});
  }, []);

  // Refetch voices when provider OR stt_language changes — only reset voice
  // if the currently-saved voice is not in the new (filtered) list.
  // Passing sttLanguage filters ElevenLabs voices to the active language's
  // bucket so VN/CN owners only see voices that sound natural for them.
  const providerChangedByUser = useRef(false);
  useEffect(() => {
    getTTSVoices(ttsProvider, sttLanguage).then((voices) => {
      setTtsVoices(voices);
      if (providerChangedByUser.current && voices.length > 0 && !voices.includes(ttsVoice)) {
        setTtsVoice(voices[0]);
      }
      providerChangedByUser.current = true;
    }).catch(() => {});
  }, [ttsProvider, sttLanguage]);

  // Auto-mirror AI Brain key/URL into TTS while TTS field is empty.
  // Once the user types into TTS the sync stops; clearing it re-enables mirroring.
  useEffect(() => {
    if (!ttsApiKey && llmApiKey) setTtsApiKey(llmApiKey);
  }, [llmApiKey, ttsApiKey]);
  useEffect(() => {
    if (!ttsBaseUrl && llmUrl) setTtsBaseUrl(llmUrl);
  }, [llmUrl, ttsBaseUrl]);
  // Same auto-mirror for STT in autonomous mode (Deepgram has its own key).
  useEffect(() => {
    if (sttProvider === "autonomous" && !sttApiKey && llmApiKey) setSttApiKey(llmApiKey);
  }, [llmApiKey, sttApiKey, sttProvider]);
  useEffect(() => {
    if (sttProvider === "autonomous" && !sttBaseUrl && llmUrl) setSttBaseUrl(llmUrl);
  }, [llmUrl, sttBaseUrl, sttProvider]);

  const scrollTo = (id: SectionId) => {
    setActiveSection(id);
    window.location.hash = id;
  };

  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSaving(true);
    try {
      let channelCreds: Record<string, string> = {};
      if (channel === "telegram") {
        channelCreds = { telegram_bot_token: teleToken, telegram_user_id: teleUserId };
      } else if (channel === "slack") {
        channelCreds = { slack_bot_token: slackBotToken, slack_app_token: slackAppToken, slack_user_id: slackUserId };
      } else {
        channelCreds = { discord_bot_token: discordBotToken, discord_guild_id: discordGuildId, discord_user_id: discordUserId };
      }
      // STT provider is implicit on backend: deepgram if deepgram_api_key set,
      // else autonomous via stt_api_key/stt_base_url. So when picking
      // autonomous we send deepgram_api_key="" to clear it (backend now allows
      // blank-clears for STT/Deepgram), and vice versa.
      const sttFields = sttProvider === "deepgram"
        ? { deepgram_api_key: deepgramApiKey, stt_api_key: "", stt_base_url: "" }
        : { deepgram_api_key: "", stt_api_key: sttApiKey, stt_base_url: sttBaseUrl };
      await updateDeviceConfig({
        ssid: ssid.trim(), password,
        channel, ...channelCreds,
        llm_base_url: llmUrl, llm_api_key: llmApiKey, llm_model: llmModel,
        llm_disable_thinking: llmDisableThinking,
        ...sttFields,
        stt_language: sttLanguage,
        tts_api_key: ttsApiKey, tts_base_url: ttsBaseUrl, tts_provider: ttsProvider, tts_voice: ttsVoice, device_id: deviceId,
        mqtt_endpoint: mqttEndpoint, mqtt_username: mqttUsername,
        mqtt_password: mqttPassword,
        mqtt_port: mqttPort ? parseInt(mqttPort, 10) : 0,
        fa_channel: faChannel, fd_channel: fdChannel,
      } as Parameters<typeof updateDeviceConfig>[0]);
      toast.success("Config saved — restart Lumi for changes to take effect.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Save failed.");
    }
    setSaving(false);
  }, [
    channel, teleToken, teleUserId, slackBotToken, slackAppToken, slackUserId,
    discordBotToken, discordGuildId, discordUserId, ssid, password, llmUrl,
    llmApiKey, llmModel, llmDisableThinking, deepgramApiKey, sttApiKey, sttBaseUrl, sttProvider, sttLanguage,
    ttsApiKey, ttsBaseUrl, ttsProvider, ttsVoice, deviceId,
    mqttEndpoint, mqttUsername, mqttPassword, mqttPort, faChannel, fdChannel,
  ]);

  return (
    <div className={`lm-root lm-edit ${themeClass}`} style={{
      display: "flex", height: "100vh",
      background: C.bg, color: C.text,
      fontFamily: "'Inter', 'Segoe UI', sans-serif", fontSize: 14,
    }}>
      <style>{`
        @media (max-width: 640px) {
          .lm-edit .lm-sidebar { display: none !important; }
          .lm-edit .lm-mobile-tabs { display: flex !important; }
          .lm-edit .lm-mobile-footer { display: block !important; }
          .lm-edit .lm-main-content { padding: 16px !important; }
        }
      `}</style>

      {/* ── Sidebar (hidden on mobile) ── */}
      <aside className="lm-sidebar" style={{
        width: 192, flexShrink: 0,
        background: C.sidebar, borderRight: `1px solid ${C.border}`,
        display: "flex", flexDirection: "column",
      }}>

        <nav style={{ padding: "10px 0", flex: 1 }}>
          {SECTIONS.map((s) => {
            const active = activeSection === s.id;
            return (
              <button key={s.id} onClick={() => scrollTo(s.id)} style={{
                display: "flex", alignItems: "center", gap: 9,
                padding: "8px 14px", borderRadius: 8, margin: "2px 8px",
                fontSize: 12.5, fontWeight: active ? 600 : 400,
                color: active ? C.amber : "var(--lm-text-dim)",
                background: active ? C.amberDim : "transparent",
                cursor: "pointer", transition: "all 0.15s",
                border: "none", width: "calc(100% - 16px)", textAlign: "left",
              }}>
                {s.icon}
                {s.label}
              </button>
            );
          })}
        </nav>

        <div style={{ padding: "12px 16px", borderTop: `1px solid ${C.border}`, display: "flex", alignItems: "center", justifyContent: "space-between" }}>
          <a href="/monitor" style={{
            display: "flex", alignItems: "center", gap: 7,
            color: C.textMuted, textDecoration: "none", fontSize: 12,
            transition: "color 0.15s",
          }}
            onMouseEnter={(e) => (e.currentTarget.style.color = C.textDim)}
            onMouseLeave={(e) => (e.currentTarget.style.color = C.textMuted)}
          >
            ← Monitor
          </a>
          <button onClick={toggleTheme} style={{
            background: "none", border: "none", cursor: "pointer",
            fontSize: 14, color: C.textMuted, padding: "2px 4px",
          }} title={`Theme: ${theme}`}>
            {theme === "dark" ? "◑" : "◐"}
          </button>
        </div>
      </aside>

      {/* ── Main ── */}
      <main style={{ flex: 1, minWidth: 0, display: "flex", flexDirection: "column", overflow: "hidden" }}>

        {/* Mobile tabs (hidden on desktop) */}
        <div className="lm-mobile-tabs" style={{
          display: "none", overflowX: "auto", gap: 4, padding: "8px 12px",
          borderBottom: `1px solid ${C.border}`, flexShrink: 0, alignItems: "center",
        }}>
          {SECTIONS.map((s) => {
            const active = activeSection === s.id;
            return (
              <button key={s.id} onClick={() => scrollTo(s.id)} style={{
                padding: "5px 10px", borderRadius: 6, fontSize: 11, fontWeight: active ? 600 : 400,
                color: active ? C.amber : C.textDim,
                background: active ? C.amberDim : "transparent",
                border: "none", cursor: "pointer", whiteSpace: "nowrap", flexShrink: 0,
              }}>
                {s.label}
              </button>
            );
          })}
          <button onClick={toggleTheme} style={{
            background: "none", border: "none", cursor: "pointer",
            fontSize: 14, color: C.textMuted, padding: "2px 6px", marginLeft: "auto", flexShrink: 0,
          }}>
            {theme === "dark" ? "◑" : "◐"}
          </button>
        </div>

        {/* Topbar */}
        <div style={{
          padding: "10px 24px", borderBottom: `1px solid ${C.border}`,
          display: "flex", alignItems: "center", justifyContent: "space-between", flexShrink: 0,
        }}>
          <span style={{ fontSize: 13, fontWeight: 600, color: C.text }}>
            {SECTIONS.find((s) => s.id === activeSection)?.label}
          </span>
          {activeSection !== "face" && (
            <button
              form="edit-form"
              type="submit"
              disabled={saving || loadingCfg}
              style={{
                padding: "6px 18px", borderRadius: 7, fontSize: 12, fontWeight: 600,
                cursor: saving || loadingCfg ? "not-allowed" : "pointer",
                border: "none",
                background: saving || loadingCfg ? C.surface : C.amber,
                color: saving || loadingCfg ? C.textMuted : "#0C0B09",
                transition: "all 0.15s",
                opacity: saving || loadingCfg ? 0.6 : 1,
              }}
            >
              {saving ? "Saving…" : "Save Changes"}
            </button>
          )}
        </div>

        {/* Content */}
        <div ref={contentRef} className="lm-fade-in lm-main-content" style={{
          flex: 1, minHeight: 0, overflowY: "auto", padding: "24px 32px",
        }}>
          <div style={{ maxWidth: 560, margin: "0 auto" }}>

            {error && (
              <div style={{
                background: "rgba(248,113,113,0.08)", border: "1px solid rgba(248,113,113,0.25)",
                borderRadius: 8, padding: "10px 14px", fontSize: 12, color: C.red, marginBottom: 16,
              }}>
                {error}
              </div>
            )}

            <div style={{
              background: C.amberDim, border: "1px solid rgba(245,158,11,0.2)",
              borderRadius: 8, padding: "10px 14px", fontSize: 11.5,
              color: C.textDim, marginBottom: 20, lineHeight: 1.6,
            }}>
              ↻ &nbsp;Restart Lumi after saving for AI brain and channel changes to take full effect.
            </div>

            {loadingCfg ? <SkeletonBlock /> : (
              <form id="edit-form" onSubmit={handleSubmit}>

                <DeviceSection
                  active={activeSection === "device"}
                  deviceId={deviceId} setDeviceId={setDeviceId}
                  mac={mac}
                />

                <WifiSection
                  active={activeSection === "wifi"}
                  wifiLoaded={wifiLoaded}
                  ssid={ssid} setSsid={setSsid}
                  password={password} setPassword={setPassword}
                />

                <LLMSection
                  active={activeSection === "llm"}
                  llmLoaded={llmLoaded}
                  llmApiKey={llmApiKey} setLlmApiKey={setLlmApiKey}
                  llmUrl={llmUrl} setLlmUrl={setLlmUrl}
                  llmModel={llmModel} setLlmModel={setLlmModel}
                />

                <EditVoiceSection
                  active={activeSection === "voice"}
                  sttLanguage={sttLanguage}
                  faceOwners={faceOwners}
                  loadFaceOwners={loadFaceOwners}
                />

                <EditFaceSection
                  active={activeSection === "face"}
                  faceOwners={faceOwners}
                  loadFaceOwners={loadFaceOwners}
                />

                <TTSSection
                  active={activeSection === "tts"}
                  ttsLoaded={ttsLoaded}
                  llmLoaded={llmLoaded}
                  ttsApiKey={ttsApiKey} setTtsApiKey={setTtsApiKey}
                  ttsBaseUrl={ttsBaseUrl} setTtsBaseUrl={setTtsBaseUrl}
                  ttsProvider={ttsProvider} setTtsProvider={setTtsProvider}
                  ttsProviders={ttsProviders}
                  ttsVoice={ttsVoice} setTtsVoice={setTtsVoice}
                  ttsVoices={ttsVoices}
                  sttLanguage={sttLanguage}
                  llmApiKey={llmApiKey} llmUrl={llmUrl}
                />

                <STTSection
                  active={activeSection === "stt"}
                  sttLanguage={sttLanguage} setSttLanguage={setSttLanguage}
                  sttProvider={sttProvider} setSttProvider={setSttProvider}
                  sttLoaded={sttLoaded}
                  llmLoaded={llmLoaded}
                  deepgramApiKey={deepgramApiKey} setDeepgramApiKey={setDeepgramApiKey}
                  sttApiKey={sttApiKey} setSttApiKey={setSttApiKey}
                  sttBaseUrl={sttBaseUrl} setSttBaseUrl={setSttBaseUrl}
                />

                <ChannelSection
                  active={activeSection === "channel"}
                  channel={channel} setChannel={setChannel}
                  channelLoaded={channelLoaded}
                  teleToken={teleToken} setTeleToken={setTeleToken}
                  teleUserId={teleUserId} setTeleUserId={setTeleUserId}
                  slackBotToken={slackBotToken} setSlackBotToken={setSlackBotToken}
                  slackAppToken={slackAppToken} setSlackAppToken={setSlackAppToken}
                  slackUserId={slackUserId} setSlackUserId={setSlackUserId}
                  discordBotToken={discordBotToken} setDiscordBotToken={setDiscordBotToken}
                  discordGuildId={discordGuildId} setDiscordGuildId={setDiscordGuildId}
                  discordUserId={discordUserId} setDiscordUserId={setDiscordUserId}
                />

                <MqttSection
                  active={activeSection === "mqtt"}
                  mqttLoaded={mqttLoaded}
                  mqttEndpoint={mqttEndpoint} setMqttEndpoint={setMqttEndpoint}
                  mqttPort={mqttPort} setMqttPort={setMqttPort}
                  mqttUsername={mqttUsername} setMqttUsername={setMqttUsername}
                  mqttPassword={mqttPassword} setMqttPassword={setMqttPassword}
                  faChannel={faChannel} setFaChannel={setFaChannel}
                  fdChannel={fdChannel} setFdChannel={setFdChannel}
                />

              </form>
            )}
          </div>
        </div>

        {/* Mobile footer — back to Monitor. Hidden on desktop (sidebar has it). */}
        <div className="lm-mobile-footer" style={{
          display: "none", padding: "10px 16px",
          borderTop: `1px solid ${C.border}`, background: C.sidebar, flexShrink: 0,
        }}>
          <a href="/monitor" style={{
            display: "inline-flex", alignItems: "center", gap: 7,
            color: C.textMuted, textDecoration: "none", fontSize: 13,
          }}>← Monitor</a>
        </div>
      </main>
    </div>
  );
}
