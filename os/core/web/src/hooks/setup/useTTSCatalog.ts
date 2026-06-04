import { useEffect, useRef, useState } from "react";
import { getTTSProviders, getTTSVoices } from "@/lib/api";

// Manages the TTS provider+voice dropdowns for Setup. Encapsulates the
// fetch-on-mount + refetch-on-provider/lang-change pattern plus the
// URL-prefill validation (server has no allow-list, so we gate FE-side).
export function useTTSCatalog({
  ttsProvider,
  sttLanguage,
  ttsVoice,
  urlProvider,
  urlVoice,
  setTtsProvider,
  setTtsVoice,
}: {
  ttsProvider: string;
  sttLanguage: string;
  ttsVoice: string;
  urlProvider: string;
  urlVoice: string;
  setTtsProvider: (v: string) => void;
  setTtsVoice: (v: string) => void;
}) {
  const [ttsProviders, setTtsProviders] = useState<string[]>([]);
  const [ttsVoices, setTtsVoices] = useState<string[]>([]);

  // Mount: load provider list + validate URL provider against allow-list.
  useEffect(() => {
    getTTSProviders().then((providers) => {
      setTtsProviders(providers);
      if (urlProvider && providers.length > 0 && !providers.includes(urlProvider)) {
        console.warn(`[setup] URL tts_provider="${urlProvider}" not in ${providers.join(",")}, using ${providers[0]}`);
        setTtsProvider(providers[0]);
      }
    }).catch(() => {});
    getTTSVoices().then(setTtsVoices).catch(() => {});
    // Intentional empty deps — mount-only, like the original effect.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Refetch voices when provider OR sttLanguage changes — only reset voice
  // if the currently-selected one is not in the new (filtered) list. Passing
  // sttLanguage filters ElevenLabs voices to the active language bucket.
  const providerChangedByUser = useRef(false);
  const urlVoiceValidated = useRef(false);
  useEffect(() => {
    getTTSVoices(ttsProvider, sttLanguage).then((voices) => {
      setTtsVoices(voices);
      if (voices.length > 0 && !voices.includes(ttsVoice)) {
        // Reset cases: (a) user switched provider/lang, voice no longer valid;
        // (b) first load and URL prefilled an invalid voice. Skip otherwise to
        // avoid clobbering a saved-cfg voice that's still loading in parallel.
        const urlVoiceInvalid = !urlVoiceValidated.current && !!urlVoice;
        if (providerChangedByUser.current || urlVoiceInvalid) {
          if (urlVoiceInvalid) {
            console.warn(`[setup] URL tts_voice="${urlVoice}" not in voice list for provider=${ttsProvider} lang=${sttLanguage || "auto"}, using ${voices[0]}`);
          }
          setTtsVoice(voices[0]);
        }
      }
      urlVoiceValidated.current = true;
      providerChangedByUser.current = true;
    }).catch(() => {});
  }, [ttsProvider, sttLanguage]);

  return { ttsProviders, ttsVoices };
}
