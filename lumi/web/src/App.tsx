import { useEffect, useState } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { Toaster } from "@/components/ui/sonner";
import Setup from "@/pages/Setup";
import Monitor from "@/pages/monitor";
import EditConfig from "@/pages/EditConfig";
import GwConfig from "@/pages/GwConfig";
import { checkInternet, getApiToken, getDeviceConfig, getSetupStatus } from "@/lib/api";

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
          window.location.replace(`http://${s.lan_ip}${window.location.pathname}${window.location.search}`);
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

// Bootstrap the admin bearer token for /api/* admin routes (PUT
// device/config, software-update, logs, etc.). Skips when sessionStorage
// already has one. Pre-login transition: token == llm_api_key fetched from
// the still-open GET /api/device/config. Silent on error — Setup mode and
// unprovisioned devices may not have a token yet, that's expected.
function useApiTokenBootstrap() {
  useEffect(() => {
    if (getApiToken()) return;
    getDeviceConfig().catch(() => {});
  }, []);
}

function App() {
  useApiTokenBootstrap();
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<SetupGate />} />
        <Route path="/monitor" element={<Monitor />} />
        <Route path="/setup" element={<SetupGate />} />
        <Route path="/edit" element={<EditConfig />} />
        <Route path="/gw-config" element={<GwConfig />} />
        <Route path="/dashboard" element={<Navigate to="/monitor" replace />} />
      </Routes>
      <Toaster richColors position="top-center" />
    </BrowserRouter>
  );
}

export default App;
