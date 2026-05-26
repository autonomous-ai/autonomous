# imager/lib — shared bash libraries for image builders

Reserved for the upcoming refactor that extracts the chroot stages
(currently inline in `build.sh` lines ~709–1395 and ~1784–1995, plus a
TODO_SHARED_STAGES placeholder in `build-orangepi.sh`) into sourceable
bash scripts so the RPi and OrangePi builders stop drifting.

Planned files (none implemented yet):

| File | Purpose |
|------|---------|
| `chroot-phase1.sh` | Board-agnostic Phase 1 chroot block — apt install, AP setup, PulseAudio AEC + udev, resolvconf fallback, lelamp uv prep, Node.js + OpenClaw, systemd units. |
| `chroot-overlay.sh` | Board-agnostic Phase 2 chroot block — OTA metadata fetch, backend/lelamp/web/buddy install, webrtcvad patch. |
| `device-ap-mode.sh` / `device-sta-mode.sh` / `connect-wifi.sh` | The three runtime helper scripts currently embedded as heredocs in `build.sh`. Once extracted they can be reused verbatim by both builders and lint-checked by CI. |

See `imager/README.md` (root of imager/) for the maintenance policy until
the extraction lands.
