import { C, Field, PasswordField, SectionCard } from "./shared";

export function DeviceSection({
  active, deviceId, setDeviceId, mac,
  adminPassword, setAdminPassword,
  adminPasswordConfirm, setAdminPasswordConfirm,
}: {
  active: boolean;
  deviceId: string;
  setDeviceId: (v: string) => void;
  mac?: string;
  // Optional — only Setup passes these. EditConfig doesn't show the admin
  // password fields here because the credential is set once at provision
  // and rotated via a dedicated UI later (TODO).
  adminPassword?: string;
  setAdminPassword?: (v: string) => void;
  adminPasswordConfirm?: string;
  setAdminPasswordConfirm?: (v: string) => void;
}) {
  const showAdminPasswordFields = setAdminPassword !== undefined;
  const mismatch =
    showAdminPasswordFields &&
    !!adminPasswordConfirm &&
    !!adminPassword &&
    adminPassword !== adminPasswordConfirm;
  return (
    <SectionCard id="device" title="Device" active={active}>
      <Field label="Device ID" id="device_id" value={deviceId} onChange={setDeviceId} placeholder="lumi-001" readOnly />
      <Field label="MAC" id="mac" value={mac ?? ""} onChange={() => {}} placeholder="Lumi-XXXX" readOnly />
      {showAdminPasswordFields && (
        <>
          <div style={{
            fontSize: 11, color: C.textDim, marginTop: 4, marginBottom: 8, lineHeight: 1.5,
          }}>
            Set an admin password — you'll sign in with this from any browser
            after setup.
          </div>
          <PasswordField
            label="Admin Password"
            id="admin_password"
            value={adminPassword ?? ""}
            onChange={setAdminPassword!}
            placeholder="At least 6 characters"
          />
          <PasswordField
            label="Confirm Password"
            id="admin_password_confirm"
            value={adminPasswordConfirm ?? ""}
            onChange={setAdminPasswordConfirm!}
            placeholder="Re-enter password"
          />
          {mismatch && (
            <div style={{ fontSize: 11, color: C.red, marginTop: -4, marginBottom: 8 }}>
              Passwords don't match.
            </div>
          )}
        </>
      )}
    </SectionCard>
  );
}
