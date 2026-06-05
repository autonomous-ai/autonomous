import { LockedField, LockedPasswordField, SectionCard } from "@/components/setup/shared";

export interface WifiLoadedState {
  ssid: boolean;
  password: boolean;
}

export function WifiSection({
  active, wifiLoaded,
  ssid, setSsid, password, setPassword,
}: {
  active: boolean;
  wifiLoaded: WifiLoadedState;
  ssid: string; setSsid: (v: string) => void;
  password: string; setPassword: (v: string) => void;
}) {
  return (
    <SectionCard id="wifi" title="Wi-Fi" active={active}>
      <LockedField lockedInitially={wifiLoaded.ssid} label="Wi-Fi network" id="ssid" value={ssid} onChange={setSsid} placeholder="Network name" />
      <LockedPasswordField lockedInitially={wifiLoaded.password} label="Password" id="password" value={password} onChange={setPassword} placeholder="Wi-Fi password" />
    </SectionCard>
  );
}
