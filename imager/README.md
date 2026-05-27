# imager — Lumi golden image builder

Produces SD card images that boot OrangePi 4 Pro (or Raspberry Pi 5) directly
into the Lumi AP/hotspot setup wizard. Flash, insert, power on — no `setup.sh`
needed on the target.

```bash
make build                        # → output/golden-opi.img.xz (default: OrangePi)
make sd-card-list                 # find your SD card disk number
make sd-card-flash DISK=N         # decompresses on the fly via `xz | dd`
```

The first build takes ~25–40 min (downloads vendor base image, qemu-arm64
chroot apt install, OTA backend bake, xz compress). Re-runs are faster —
`input/orangepi.7z` is cached, only Phase 3 (OTA bake) and beyond re-run.

## Targets

| Board | TARGET | Builder | Output | Status |
|-------|--------|---------|--------|--------|
| **OrangePi 4 Pro v2 (Allwinner A733)** *default* | `opi` | `build-orangepi.sh` | `output/golden-opi.img.xz` | **production** |
| Raspberry Pi 5 (RPi OS Trixie) | `rpi` | `build.sh` | `output/golden.img` | working |

`make TARGET=rpi build` for the Pi 5 path. Default no-arg `make build` uses
OrangePi.

## OrangePi build flow (the new default)

The OrangePi 4 Pro v2 uses Allwinner A733 (`sun60iw2`) which Orange Pi has not
published a public download for. The Lumi dev team uploaded the vendor "user-
built" image to a Google Drive folder
(`drive.google.com/drive/folders/1AzF-uTwA328qDFPaVBaKpiP4VjZjkmbS`), and the
imager pulls that fixed file ID. Override via `OPI_FILE_ID=<id> make build`
when a newer release is uploaded.

```
Phase 0  gdown Orangepi4pro_*.7z  →  input/orangepi.7z (cached, 734 MB)
Phase 1  7z extract  →  truncate to OUT_IMG_SIZE (default 6 GB)
         growpart + resize2fs to fill the larger output
Phase 2  chroot qemu-arm64:
         - apt install (production-matched list)
         - Node.js 22 + OpenClaw npm global + openclaw onboard --skip-health
         - uv (Python pkg mgr for LeLamp)
         - systemd units (lumi, bootstrap, lumi-lelamp, lumi-wifi-power-save, openclaw)
         - helper scripts /usr/local/bin/{device-ap-mode,device-sta-mode,connect-wifi,software-update}
           (verbatim from production OPi @ 100.111.149.69)
         - configs: hostapd, dnsmasq, dhcpcd, nginx (full prod CSP + WS + captive-portal),
           PulseAudio (WebRTC AEC + anon socket), udev PULSE_IGNORE for I2S codecs,
           /etc/asound.conf (lamp_speaker/lamp_micro1 for ES8389 sndi2s4)
         - mask orangepi-firstrun-config.service (vendor wizard)
Phase 3  OTA bake from metadata.json:
         - bootstrap-server + lumi-server binaries
         - LeLamp Python app + uv sync --python 3.12 --extra hardware
           (with webrtcvad pkg_resources patch for Py 3.12+)
         - Web UI to /usr/share/nginx/html/setup
         - Claude Desktop Buddy BLE plugin (optional, if OTA key present)
Phase 4  lumi-resize-once.service — first-boot growpart + resize2fs to fill
         the actual SD card, then self-destructs.
Phase 5  xz -9 --threads=0 → output/golden-opi.img.xz (~1.5–2 GB)
```

### First boot on the device

1. `lumi-resize-once.service` expands ext4 to the full SD card size, removes
   itself.
2. The device has no `/etc/wpa_supplicant/wpa_supplicant-wlan0.conf` network
   block, so `wpa_supplicant@wlan0` finds nothing to associate with.
3. Operator runs `sudo device-ap-mode` (or the bootstrap-server triggers it
   automatically when no STA association after a timeout).
4. SSID becomes `Lumi-XXXX` where XXXX is the last 4 hex chars of the ethernet
   MAC (board has no device-tree serial — MAC fallback chain in
   `device-ap-mode` handles this).
5. mDNS hostname `lumi-<xxxx>.local` is published by `avahi-daemon`.
6. Connect to AP → http://192.168.100.1/ → setup wizard fills API keys + home
   WiFi → device-sta-mode kicks in → device reachable via `lumi-xxxx.local`.

## Configuration knobs

All env vars, override at the `make` call:

| Variable | Default | Effect |
|----------|---------|--------|
| `TARGET` | `opi` | `opi` or `rpi` — picks builder script + output filename |
| `OUT_IMG_SIZE` | `6G` | Output image size before xz. SD card must be ≥ this. lumi-resize-once expands on first boot. |
| `OPI_FILE_ID` | `1CYfOaY6f5DozJBNvPJ0Gx1jBIFlGe8fn` | Google Drive file ID for Orangepi4pro_*.7z. Update when dev team uploads new release. |
| `OTA_METADATA_URL` | `https://storage.googleapis.com/s3-autonomous-upgrade-3/lumi/ota/metadata.json` | Backend binaries source. Used by Phase 3. |
| `OPENCLAW_VERSION` | `2026.5.7` | npm package version pin. Bump as OpenClaw releases. |
| `AP_BAND` | `2.4` | `2.4` or `5` — hostapd hw_mode. 5 GHz needs chip + regulatory support. |
| `AP_CHANNEL` | `6` (2.4 GHz) / `36` (5 GHz) | hostapd channel |
| `COUNTRY_CODE` | `US` | Regulatory domain for wpa_supplicant + hostapd |

Example: rebuild against a new vendor release uploaded to a different file ID:

```bash
rm input/orangepi.7z output/base-opi.img output/golden-opi.img.xz
OPI_FILE_ID=NEW_FILE_ID_HERE make build
```

## File layout

```
imager/
├── Dockerfile             — Ubuntu 24.04 + qemu-aarch64-static + p7zip-full + gdown
├── Makefile               — build / flash / SD-test targets, dispatches on TARGET
├── build-orangepi.sh      — OrangePi 4 Pro builder (default; ~680 lines)
├── build.sh               — Raspberry Pi 5 builder (~1990 lines)
├── lib/                   — RESERVED for shared chroot stages (see lib/README.md)
├── input/                 — cached source images (.7z / .img.xz). gitignored.
├── output/                — built golden images. gitignored.
└── README.md              — this file
```

## Sanity checks after first flash

SSH in (`ssh system@<lumi-xxxx>.local`, password `12345` until rotated by the
setup wizard) and verify:

```bash
systemctl is-enabled lumi lumi-lelamp lumi-wifi-power-save openclaw avahi-daemon
ls /usr/local/bin/{lumi-server,bootstrap-server,device-ap-mode,connect-wifi,software-update}
ls /opt/lelamp/.venv/bin/uvicorn       # LeLamp uv sync succeeded
openclaw --version                       # OpenClaw npm global installed
ls /etc/asound.conf /etc/udev/rules.d/91-pulseaudio-lelamp-ignore.rules
findmnt /                                # ext4 root, expanded to full SD
systemctl is-enabled lumi-resize-once 2>&1 | grep -q "not found" && echo OK_resize-once-self-destructed
```

## Maintenance — Pi vs OPi drift

The chroot stage logic is **duplicated** between `build.sh` (Pi) and
`build-orangepi.sh` (OPi). When you change something inside either script's
chroot block (apt list, helper script, systemd unit, nginx config), mirror it
in the other if it's board-agnostic.

The planned refactor is to extract a sourceable `imager/lib/chroot-stages.sh`
that both builders source. Blocked on first verified OPi golden image so the
refactor has a working baseline. See `lib/README.md` for the design sketch.

## Source image notes (OPi)

The base `.7z` from the Google Drive folder is the **vendor "user-built"
image** referenced by `/etc/orangepi-release` on the production device:

```
BOARD=orangepi4pro
BOARDFAMILY=sun60iw2
BUILD_REPOSITORY_URL=https://github.com/orangepi-xunlong/orangepi-build
BUILD_REPOSITORY_COMMIT=70abbec-dirty
DISTRIBUTION_CODENAME=bookworm
VERSION=1.0.6
IMAGE_TYPE=user-built
```

The `-dirty` suffix means the dev team applied local patches to
`orangepi-build` before generating the image — these patches are not public.
If we ever need to rebuild from source instead of relying on the .7z, ask the
dev team for the patch set + the exact `./build.sh` invocation.

The folder also has variants we don't use:

- `Orangepi4pro_1.0.6_debian_bookworm_desktop_xfce_*.7z` (Xfce desktop — not minimal)
- `Orangepi4pro_1.0.6_debian_bullseye_server_*.7z` (Debian 11, EOL)
- `Orangepi4pro_1.0.6_debian_bullseye_desktop_xfce_*.7z`

We pin `bookworm_server` because that's what production runs.

## Troubleshooting

**Docker on Mac**: `--privileged + losetup` is flaky on Docker Desktop. If
`losetup: cannot find unused loop device` appears, switch to OrbStack
(`brew install orbstack && orb start`) or run the build on a Linux host.

**SSL `error: docker-credential-desktop` not in PATH**: known Docker Desktop
config issue. Fix:

```bash
jq 'del(.credsStore)' ~/.docker/config.json > /tmp/c && mv /tmp/c ~/.docker/config.json
```

**`gdown` failures**: Google Drive sometimes throttles anonymous downloads
when the file is hot. Wait an hour, retry, or download the .7z manually from
the folder URL and drop it at `imager/input/orangepi.7z`.

**Final image fails to boot OPi**: open the partition table check on the
output `.img.xz` to confirm the U-Boot bootloader region (first ~16 MB,
inherited from the vendor .7z) survived. The build script never touches the
bootloader sectors — they should be byte-identical to the source image.

```bash
xz -dc output/golden-opi.img.xz | head -c 16M | hexdump -C | head -20
```

## Recent changes

**2026-05-26** — Full rewrite of the OPi builder:

- Switched base image from Armbian (wrong assumption) to vendor Orange Pi
  Bookworm 1.0.6 .7z (matches `/etc/orangepi-release` on production OPi).
- Dropped Btrfs `@`/`@factory` subvolume scheme — production runs plain ext4
  single-partition. Replaced @factory factory-reset with `lumi-resize-once`
  first-boot expand.
- Bootloader handling: now relies on vendor image's pre-baked U-Boot in raw
  sectors. No more `armbianEnv.txt` manipulation; `orangepiEnv.txt` is left
  intact.
- chroot stages now mirror production OPi exactly: ES8389 ALSA aliases,
  vendor service masks (`orangepi-firstrun-config.service`), OpenClaw with
  the production env block (XDG_*, PUPPETEER paths), production nginx config
  with CSP + WebSocket proxies + captive-portal returns.
- `make build` is one command end-to-end.

**Earlier (Pi 5 only)** — ported from `scripts/setup.sh`: openresolv +
`name_servers="1.1.1.1 8.8.8.8"` fallback (Pi-only — OPi vendor image
doesn't use openresolv), avahi `lumi-<suffix>.local` mDNS, PulseAudio udev
ignore for `sndi2s4` + `wm8960soundcard`, webrtcvad Py3.12+ patch, MAC-based
SSID fallback for non-Pi boards in `device-ap-mode`, Pi Imager `wpa.conf`
cleanup, `AP_BAND=5` knob, `stage_buddy` (Claude Desktop Buddy BLE plugin)
gated on OTA `claude-desktop-buddy.url`.
