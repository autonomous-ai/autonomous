import { useCallback, useEffect, useRef, useState } from "react";
import { HW } from "../types";
import type { FaceOwnersDetail } from "../types";
import { usePolling } from "../../../hooks/usePolling";
import type { CooldownState } from "./types";

// Enrolled-owners list + face detection state (cooldowns, current user), with
// background polling and a user-triggered refresh. `refresh` is returned so the
// owner-mutation flows (enroll / rename / remove) can reload the list after a
// change.
export function useFaceData() {
  const [data, setData] = useState<FaceOwnersDetail | null>(null);
  const [error, setError] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  // Cooldown state (strangers/friends forget countdown — unrelated to
  // current_user; /face/cooldowns is purely a debug view of detection state).
  const [cooldowns, setCooldowns] = useState<CooldownState | null>(null);
  const [cdError, setCdError] = useState(false);
  const [resetting, setResetting] = useState(false);
  // Spins the ↻ glyph only on an explicit user-triggered refresh (not on the
  // 10s background poll, which would flicker the icon nonstop).
  const [manualRefreshing, setManualRefreshing] = useState(false);

  // Current user (effective user the device sees right now). Polled separately
  // from /face/current-user — this is the source used by the OS-server handler,
  // activity logging, and the "Here now" UI.
  const [currentUser, setCurrentUser] = useState<string>("");

  const refresh = useCallback(async () => {
    abortRef.current?.abort();
    const ctrl = new AbortController();
    abortRef.current = ctrl;
    try {
      const r = await fetch(`${HW}/face/owners`, { signal: ctrl.signal }).then((x) => x.json());
      if (ctrl.signal.aborted) return;
      setData({ enrolled_count: r.enrolled_count ?? 0, persons: r.persons ?? [] });
      setError(false);
    } catch (e) {
      if ((e as Error).name === "AbortError") return;
      setError(true);
    }
  }, []);

  // User-triggered refresh: spins the ↻ for a beat so the click feels answered,
  // even if the fetch returns instantly. The 600ms floor keeps the spin visible.
  const handleManualRefresh = useCallback(async () => {
    setManualRefreshing(true);
    const started = performance.now();
    try {
      await refresh();
    } finally {
      const elapsed = performance.now() - started;
      const wait = Math.max(0, 600 - elapsed);
      window.setTimeout(() => setManualRefreshing(false), wait);
    }
  }, [refresh]);

  useEffect(() => {
    return () => { abortRef.current?.abort(); };
  }, []);

  usePolling(async (signal) => {
    // Delegate to refresh(), but we can't pass the signal because refresh
    // uses its own AbortController. The usePolling timeout will still fire
    // its own abort — refresh's internal controller handles staleness.
    void signal;
    await refresh();
  }, 10_000, { timeoutMs: 8000 });

  const refreshFaceState = useCallback(async (signal?: AbortSignal) => {
    const [cdRes, cuRes] = await Promise.allSettled([
      fetch(`${HW}/face/cooldowns`, { signal }),
      fetch(`${HW}/face/current-user`, { signal }),
    ]);
    if (cdRes.status === "fulfilled" && cdRes.value.ok) {
      setCooldowns(await cdRes.value.json());
      setCdError(false);
    } else {
      setCdError(true);
    }
    if (cuRes.status === "fulfilled" && cuRes.value.ok) {
      const j = await cuRes.value.json();
      setCurrentUser(typeof j?.current_user === "string" ? j.current_user : "");
    }
  }, []);

  usePolling(async (signal) => { await refreshFaceState(signal); }, 5000);

  const handleResetCooldowns = async () => {
    setResetting(true);
    try {
      await fetch(`${HW}/face/cooldowns/reset`, { method: "POST" });
      await refreshFaceState();
    } catch {
      // ignore
    } finally {
      setResetting(false);
    }
  };

  return {
    data, error, currentUser,
    cooldowns, cdError, resetting, manualRefreshing,
    refresh, handleManualRefresh, handleResetCooldowns,
  };
}
