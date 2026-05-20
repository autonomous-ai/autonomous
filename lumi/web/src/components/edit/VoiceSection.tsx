import { useRef, useState } from "react";
import { C, Field, SectionCard } from "@/components/setup/shared";
import { pickVoicePhrases, pickVoiceIntro, VOICE_DURATION_SEC } from "@/components/setup/voice-phrases";
import type { FaceOwner } from "@/hooks/setup/useFaceEnroll";

// Voice enroll — remote-trigger lelamp's /speaker/record-enroll. Lamp captures
// via its own mic; web only does countdown UI. Sharing label with face enroll
// keeps both biometrics in one per-user folder. State stays local since
// nothing outside this section reads it.
export function VoiceSection({
  active, sttLanguage, faceOwners, loadFaceOwners,
}: {
  active: boolean;
  sttLanguage: string;
  faceOwners: FaceOwner[];
  loadFaceOwners: () => Promise<void>;
}) {
  const VOICE_PHRASES = pickVoicePhrases(sttLanguage);
  const VOICE_INTRO = pickVoiceIntro(sttLanguage);

  const [voiceLabel, setVoiceLabel] = useState("");
  const [voicePhase, setVoicePhase] = useState<"idle" | "countdown" | "recording" | "processing">("idle");
  const [voiceCountdown, setVoiceCountdown] = useState(0);
  const [voiceMsg, setVoiceMsg] = useState<string | null>(null);
  const voiceTickRef = useRef<number | null>(null);
  const [voiceExpanded, setVoiceExpanded] = useState<Record<string, boolean>>({});
  const toggleVoiceExpanded = (label: string) =>
    setVoiceExpanded((prev) => ({ ...prev, [label]: !prev[label] }));

  const removeVoiceFile = async (name: string, file: string) => {
    if (!confirm(`Delete voice sample "${file}" for "${name}"?`)) return;
    try {
      await fetch("/api/voice/file/remove", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, file }),
      });
      loadFaceOwners();
    } catch { /* ignore */ }
  };

  const startVoiceEnroll = () => {
    if (!voiceLabel.trim()) {
      setVoiceMsg("Enter a name first");
      return;
    }
    setVoiceMsg(null);
    setVoicePhase("countdown");
    let pre = 3;
    setVoiceCountdown(pre);
    voiceTickRef.current = window.setInterval(() => {
      pre -= 1;
      if (pre > 0) {
        setVoiceCountdown(pre);
        return;
      }
      if (voiceTickRef.current) clearInterval(voiceTickRef.current);
      setVoicePhase("recording");
      let remaining = VOICE_DURATION_SEC;
      setVoiceCountdown(remaining);
      voiceTickRef.current = window.setInterval(() => {
        remaining -= 1;
        if (remaining <= 0) {
          if (voiceTickRef.current) clearInterval(voiceTickRef.current);
          setVoicePhase("processing");
          setVoiceCountdown(0);
        } else {
          setVoiceCountdown(remaining);
        }
      }, 1000);
      fetch("/hw/speaker/record-enroll", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: voiceLabel.trim().toLowerCase(), duration_sec: VOICE_DURATION_SEC }),
      })
        .then((r) => r.json().then((data) => ({ ok: r.ok, data })))
        .then(({ ok, data }) => {
          if (voiceTickRef.current) clearInterval(voiceTickRef.current);
          setVoicePhase("idle");
          setVoiceCountdown(0);
          if (ok && data.status === "ok") {
            setVoiceMsg(`Enrolled "${voiceLabel.trim().toLowerCase()}"`);
            loadFaceOwners();
          } else {
            setVoiceMsg(`Error: ${data.detail ?? data.message ?? "enroll failed"}`);
          }
        })
        .catch((e) => {
          if (voiceTickRef.current) clearInterval(voiceTickRef.current);
          setVoicePhase("idle");
          setVoiceCountdown(0);
          setVoiceMsg(`Error: ${e instanceof Error ? e.message : String(e)}`);
        });
    }, 1000);
  };

  const withVoice = faceOwners.filter((p) => (p.voice_samples?.length ?? 0) > 0);

  return (
    <SectionCard id="voice" title="My Voice (optional)" active={active}>
      <div style={{ fontSize: 11, color: C.textDim, marginBottom: 12 }}>
        {VOICE_INTRO}
      </div>
      <Field label="Name" id="voice_label" value={voiceLabel} onChange={setVoiceLabel} placeholder="e.g. Leo" />
      <div style={{
        background: C.surface, border: `1px solid ${C.border}`, borderRadius: 7,
        padding: "12px 14px", marginBottom: 12, fontSize: 13, lineHeight: 1.55, color: C.text,
      }}>
        {VOICE_PHRASES.map((p, i) => (
          <div key={i} style={{ marginBottom: i < VOICE_PHRASES.length - 1 ? 6 : 0 }}>
            <span style={{ color: C.textMuted, marginRight: 6 }}>{i + 1}.</span>
            {p}
          </div>
        ))}
      </div>
      {voiceMsg && (
        <div style={{
          fontSize: 11, padding: "6px 10px", borderRadius: 6, marginBottom: 10,
          background: voiceMsg.startsWith("Error") ? "rgba(248,113,113,0.08)" : "rgba(52,211,153,0.08)",
          color: voiceMsg.startsWith("Error") ? C.red : "rgb(52,211,153)",
        }}>{voiceMsg}</div>
      )}
      <button
        type="button"
        onClick={startVoiceEnroll}
        disabled={!voiceLabel.trim() || voicePhase !== "idle"}
        style={{
          width: "100%", padding: "11px 0", borderRadius: 7, fontSize: 13, fontWeight: 600,
          cursor: voicePhase === "idle" && voiceLabel.trim() ? "pointer" : "not-allowed",
          background: voicePhase === "recording" ? "rgba(248,113,113,0.18)"
            : voicePhase === "countdown" ? "rgba(245,158,11,0.18)"
            : voicePhase === "processing" ? C.surface
            : !voiceLabel.trim() ? C.surface : "rgba(52,211,153,0.12)",
          border: `1px solid ${voicePhase === "recording" ? "rgba(248,113,113,0.4)"
            : voicePhase === "countdown" ? "rgba(245,158,11,0.4)"
            : !voiceLabel.trim() ? C.border : "rgba(52,211,153,0.35)"}`,
          color: voicePhase === "recording" ? C.red
            : voicePhase === "countdown" ? C.amber
            : voicePhase === "processing" ? C.textDim
            : !voiceLabel.trim() ? C.textMuted : "rgb(52,211,153)",
        }}
      >
        {voicePhase === "idle" && `Start Recording (${VOICE_DURATION_SEC}s on lamp)`}
        {voicePhase === "countdown" && `Get ready... ${voiceCountdown}`}
        {voicePhase === "recording" && `● Recording on lamp — read aloud (${voiceCountdown}s)`}
        {voicePhase === "processing" && "Processing..."}
      </button>
      {withVoice.length > 0 && (
        <div style={{ marginTop: 16, borderTop: `1px solid ${C.border}`, paddingTop: 14 }}>
          <div style={{ fontSize: 10, fontWeight: 700, color: C.textDim, textTransform: "uppercase", letterSpacing: "0.09em", marginBottom: 10 }}>
            Voice Files
          </div>
          {withVoice.map((p) => {
            const expanded = !!voiceExpanded[p.label];
            return (
              <div key={p.label} style={{ padding: "10px 0", borderBottom: `1px solid ${C.border}` }}>
                <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: expanded ? 8 : 0 }}>
                  <button
                    type="button"
                    onClick={() => toggleVoiceExpanded(p.label)}
                    style={{
                      flex: 1, display: "flex", alignItems: "center", gap: 8,
                      background: "none", border: "none", cursor: "pointer", padding: 0,
                      textAlign: "left", color: C.text,
                    }}
                  >
                    <span style={{ fontSize: 11, color: C.textMuted, transition: "transform 0.15s", transform: expanded ? "rotate(90deg)" : "none" }}>▶</span>
                    <span style={{ fontSize: 13, fontWeight: 600 }}>{p.label}</span>
                    <span style={{ fontSize: 10, color: C.textMuted, fontWeight: 400 }}>({p.voice_samples!.length} file{p.voice_samples!.length !== 1 ? "s" : ""})</span>
                  </button>
                  <button
                    type="button"
                    onClick={async () => {
                      if (!confirm(`Remove ALL voice files for "${p.label}"? Face data is preserved.`)) return;
                      try {
                        await fetch("/hw/speaker/remove", {
                          method: "POST",
                          headers: { "Content-Type": "application/json" },
                          body: JSON.stringify({ name: p.label }),
                        });
                        loadFaceOwners();
                      } catch { /* ignore */ }
                    }}
                    style={{
                      background: "none", border: `1px solid ${C.border}`, borderRadius: 5,
                      cursor: "pointer", fontSize: 10, color: C.red, padding: "3px 8px",
                    }}
                  >
                    Remove all
                  </button>
                </div>
                {expanded && p.voice_samples!.map((file) => {
                  const ext = file.toLowerCase().split(".").pop() || "";
                  const url = `/hw/face/file/${p.label}/voice/${encodeURIComponent(file)}`;
                  const isAudio = ["wav", "ogg", "mp3", "webm", "m4a"].includes(ext);
                  const viewLabel = ["json", "jsonl", "txt"].includes(ext) ? "view" : "open";
                  return (
                    <div key={file} title={file} style={{
                      display: "flex", alignItems: "center", gap: 6, padding: "3px 0",
                      fontSize: 11, color: C.textDim,
                    }}>
                      <span style={{ flex: 1, minWidth: 0, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap", fontFamily: "monospace" }}>
                        {file}
                      </span>
                      {isAudio ? (
                        <>
                          <audio controls src={url} style={{ width: 180, height: 24 }} />
                          <button type="button" onClick={() => removeVoiceFile(p.label, file)}
                            style={{ background: "none", border: "none", cursor: "pointer", color: C.red, fontSize: 14, lineHeight: 1, padding: "0 4px" }} title="Delete">
                            ×
                          </button>
                        </>
                      ) : (
                        <a href={url} target="_blank" rel="noopener noreferrer"
                          style={{ fontSize: 10, color: C.amber, textDecoration: "none", padding: "2px 6px", border: `1px solid ${C.border}`, borderRadius: 4 }}>
                          {viewLabel}
                        </a>
                      )}
                    </div>
                  );
                })}
              </div>
            );
          })}
        </div>
      )}
    </SectionCard>
  );
}
