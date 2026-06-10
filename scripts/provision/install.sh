#!/bin/bash
# Run this on the Pi to download and execute the latest setup from CDN.
# Usage: curl -fsSL https://storage.googleapis.com/s3-autonomous-upgrade-3/os/install.sh | sudo bash
set -euo pipefail

# OTA metadata URL — the per-deployment source of truth for this install path.
# Override by exporting OTA_METADATA_URL before running. setup.sh consumes it.
OTA_METADATA_URL="${OTA_METADATA_URL:-https://storage.googleapis.com/s3-autonomous-upgrade-3/os/ota/metadata.json}"

# Device class for this install — picks devices/<type>/{DEVICE,SOUL}.md at
# runtime and drives the network identity (<type>-XXXX SSID + mDNS hostname).
# REQUIRED, no default: a device-agnostic OS has no universal device class, so the
# operator must declare it. E.g. DEVICE_TYPE=lamp curl -fsSL …/install.sh | sudo bash
DEVICE_TYPE="${DEVICE_TYPE:?DEVICE_TYPE must be set (e.g. DEVICE_TYPE=lamp) — no default}"

curl -fsSL -H "Cache-Control: no-cache" -H "Pragma: no-cache" \
  -o /tmp/setup.sh \
  "https://cdn.autonomous.ai/os/setup.sh"
chmod +x /tmp/setup.sh
OTA_METADATA_URL="$OTA_METADATA_URL" DEVICE_TYPE="$DEVICE_TYPE" bash /tmp/setup.sh
