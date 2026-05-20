import { useEffect, useState } from "react";
import { BrowserRouter, Routes, Route, Navigate, useLocation } from "react-router-dom";
import { Toaster } from "@/components/ui/sonner";
import Setup from "@/pages/Setup";
import Login from "@/pages/Login";
import Monitor from "@/pages/monitor";
import EditConfig from "@/pages/EditConfig";
import GwConfig from "@/pages/GwConfig";
import { checkInternet, getDeviceConfig, getSetupStatus, safeSearch, scrubLocationSecrets } from "@/lib/api";

// Detect Tailscale access by either:
//  - CGNAT IPv4 in 100.64.0.0/10 (100.64.0.0 – 100.127.255.255), or
//  - MagicDNS hostname (anything ending in `.ts.net`).
function isTailscaleHost(host: string): boolean {
  if (host.endsWith(".ts.net")) return true;
  const m = host.match(/^(\d+)\.(\d+)\./);
  if (!m) return false;
  const a = parseInt(m[1], 10);
  const b = parseInt(m[2], 10);
  return a === 100 && b >= 64 && b <= 127;
}

// Setup gate: provisioned (online) → continue mode (Voice/Face enroll, TTS
// preview), else initial mode (offline form for AP setup). When the user
// lands on the AP IP (192.168.4.1) but the lamp already has a real LAN IP
// (e.g. they bookmarked the AP URL after first setup), bounce them to the
// LAN address so the rest of the page works. `#force` in the URL hash
// forces initial mode for testing.
function SetupGate() {
  const force = typeof window !== "undefined" && window.location.hash === "#force";
  const [provisioned, setProvisioned] = useState<boolean | null>(force ? false : null);
  useEffect(() => {
    if (force) return;
    let cancelled = false;
    (async () => {
      const ok = await checkInternet().catch(() => false);
      if (cancelled) return;
      if (!ok) { setProvisioned(false); return; }
      // Online: see if we should redirect to the actual LAN IP first.
      // Skip redirect when the user is already reaching the lamp via its
      // Tailscale IP (CGNAT 100.64.0.0/10) — that's a deliberate remote-access
      // path, not the AP-IP-after-setup case we're trying to fix.
      try {
        const s = await getSetupStatus();
        if (cancelled) return;
        const here = window.location.hostname;
        // Skip the lan_ip bounce when:
        //   - on Tailscale (CGNAT) — deliberate remote-access path
        //   - on the canonical .local mDNS name — bouncing to a raw IP would
        //     undo the post-AP→STA redirect (URL must stay stable so the
        //     browser auto-resolves to the new IP on every wifi change)
        const isCanonicalMdns = here.endsWith(".local");
        if (s.lan_ip && s.lan_ip !== here && !isTailscaleHost(here) && !isCanonicalMdns) {
          window.location.replace(`http://${s.lan_ip}${window.location.pathname}${safeSearch()}`);
          return;
        }
      } catch { /* keep showing continue mode if status endpoint fails */ }
      if (!cancelled) setProvisioned(true);
    })();
    return () => { cancelled = true; };
  }, [force]);
  if (provisioned === null) return null;
  return <Setup mode={provisioned ? "continue" : "initial"} />;
}

// AuthGate wraps protected routes. Hits GET /api/device/config to probe
// session state. 401 → bounce to /login (preserving the original path so
// post-login navigation lands there). 200 → render children. Anything else
// → render children too (network blip shouldn't lock the user out).
//
// We also detect the "not yet provisioned" path: if the response carries
// has_admin_password=false the operator has never set a password, so we
// route to /setup instead. checkInternet() can be unreliable from the
// browser's perspective (LAN-only devices report no internet but are
// fully provisioned), so a config probe is the more direct signal.
function AuthGate({ children }: { children: React.ReactNode }) {
  const location = useLocation();
  const [state, setState] = useState<"checking" | "ok" | "login" | "setup">("checking");
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const cfg = await getDeviceConfig();
        if (cancelled) return;
        if (!cfg.has_admin_password) {
          setState("setup");
        } else {
          setState("ok");
        }
      } catch (err) {
        if (cancelled) return;
        const status = (err as { status?: number })?.status;
        // 401 = no/expired session. Anything else (5xx, network) → let the
        // user proceed; the next admin call will surface the real error.
        if (status === 401 || status === 503) setState("login");
        else setState("ok");
      }
    })();
    return () => { cancelled = true; };
  }, []);
  if (state === "checking") return null;
  if (state === "setup") {
    return <Navigate to={`/setup${safeSearch()}`} replace />;
  }
  if (state === "login") {
    const next = encodeURIComponent(location.pathname + location.search);
    return <Navigate to={`/login?next=${next}`} replace />;
  }
  return <>{children}</>;
}

// On every mount, scrub secret query params from the URL so they don't
// survive in browser history or address bar after the page reads them.
function useScrubSecrets() {
  useEffect(() => {
    scrubLocationSecrets();
  }, []);
}

function App() {
  useScrubSecrets();
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<SetupGate />} />
        <Route path="/setup" element={<SetupGate />} />
        <Route path="/login" element={<Login />} />
        <Route path="/monitor" element={<AuthGate><Monitor /></AuthGate>} />
        <Route path="/edit" element={<AuthGate><EditConfig /></AuthGate>} />
        <Route path="/gw-config" element={<AuthGate><GwConfig /></AuthGate>} />
        <Route path="/dashboard" element={<Navigate to="/monitor" replace />} />
      </Routes>
      <Toaster richColors position="top-center" />
    </BrowserRouter>
  );
}

export default App;
