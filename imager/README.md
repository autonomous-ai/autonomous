# imager — Golden image builder

Produces ready-to-flash SD card images that boot straight into AP/hotspot mode
("Lumi-XXXX"). No setup.sh needed on the target device — flash, insert, power on.

## Targets

| Board | Builder | Output | Status |
|-------|---------|--------|--------|
| **OrangePi 4 Pro (Armbian Trixie) — default** | `build-orangepi.sh` | `output/golden-opi.img` | **skeleton — needs hardware verification + chroot stage port** |
| Raspberry Pi 5 (RPi OS Trixie arm64) | `build.sh` | `output/golden.img` | working |

OrangePi 5 / 5 Plus / Zero variants: swap `ARMBIAN_BOARD` env var when invoking
`build-orangepi.sh`. Untested.

## Quick start

```bash
# OrangePi (default)
make build
make sd-card-list                       # find your SD card disk number
make sd-card-flash DISK=N

# Raspberry Pi 5 (explicit opt-in)
make TARGET=rpi build
make sd-card-flash TARGET=rpi DISK=N
```

The first run downloads ~2 GB (base OS image) + builds for ~20 min. Subsequent
runs reuse `output/base.img` / `output/base-opi.img` and finish in ~1 min
(only Phase 2 reruns to pick up new backend/web versions from OTA metadata).
Delete `output/base*.img` to force a full rebuild.

## File layout

```
imager/
├── Dockerfile           — Ubuntu 24.04 + qemu-aarch64-static + btrfs-progs builder env
├── Makefile             — build / flash / SD-test targets, dispatches on TARGET
├── build.sh             — RPi 5 builder (Phase 1 RPi OS → Btrfs, Phase 2 OTA bake)
├── build-orangepi.sh    — OrangePi builder (Phase 1 Armbian → Btrfs, Phase 2 stub)
├── lib/                 — RESERVED for shared chroot stages (see lib/README.md)
└── README.md            — this file
```

## Btrfs subvolume layout (both targets)

```
@                 initial live root (Btrfs)
@factory          read-only snapshot, baked at build time + refreshable via fr-snapshot
@restore-<ts>     writable snapshot of @factory, created by fr-rollback
```

Factory reset = `sudo fr-rollback` → snapshots @factory into @restore-<ts>, sets
it as the default subvolume, updates the bootloader rootflags.

## Maintenance policy — RPi vs OPi builders

**Today: duplicated chroot stages.** `build.sh` and `build-orangepi.sh` each
own their chroot blocks inline. The OPi script's chroot block is currently a
minimal stub — the AP/PulseAudio/lelamp/openclaw stages need to be ported
from `build.sh` lines 709–1395 (Phase 1 chroot) and 1784–1995 (Phase 2 overlay)
before OPi images can be used in production.

**Drift rule until extraction lands:** when you change a chroot stage in
`build.sh`, mirror the change in `build-orangepi.sh`. Search for
`TODO_SHARED_STAGES` in either file to find the affected regions.

**Planned refactor:** extract chroot stages into `imager/lib/chroot-phase1.sh`
and `imager/lib/chroot-overlay.sh` so both builders run the same code path.
Blocked on first successful OPi image so the extraction has a working baseline
to test against. See `lib/README.md` for the design sketch.

## Recent changes (2026-05-26)

Ported from `scripts/setup.sh` into `build.sh`:

- `openresolv` + DNS fallback (`name_servers="1.1.1.1 8.8.8.8"` in
  `/etc/resolvconf.conf`) — fixes empty `/etc/resolv.conf` in AP mode and on
  Armbian-derived bases that ship `/etc/resolv.conf` as a non-symlink plain file
- `avahi-daemon avahi-utils libnss-mdns` install + runtime hostname
  `lumi-<last4>.local` published from `device-ap-mode` so the setup wizard's
  AP→.local handoff works
- PulseAudio udev ignore rule for `sndi2s4` (OrangePi onboard ES8389 codec) and
  `wm8960soundcard` (RPi Seeed wm8960 hat) — keeps lelamp's exclusive ALSA grip
- `webrtcvad` `pkg_resources` shim patch for Python 3.12+ — applied after
  `uv sync` in Phase 2
- `device-ap-mode` SSID serial detection now falls back to `eth0`/`end0` MAC
  when `/proc/device-tree/serial-number` and `/proc/cpuinfo Serial` are both
  empty (OrangePi 4 Pro etc.)
- Pre-write cleanup of `wpa_supplicant.conf` (Pi Imager / Armbian-baked WiFi
  credentials no longer override our per-interface AP/STA flow)
- `AP_BAND=5` env knob → 5 GHz hostapd config (`hw_mode=a` + `ieee80211ac=1`,
  default channel 36). Default stays 2.4 GHz for compatibility.
- `stage_buddy` ported (Claude Desktop Buddy BLE plugin) — parses
  `claude-desktop-buddy.url` from OTA metadata, installs to
  `/opt/claude-desktop-buddy`, registers `lumi-buddy.service`. Skipped silently
  when the OTA key is absent.
