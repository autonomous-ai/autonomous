#!/bin/bash
# Migrate openclaw service from /root/openclaw → /root/.openclaw
# Run on each Pi that was installed before the path migration.

set -e

SERVICE=/etc/systemd/system/openclaw.service

if [ ! -f "$SERVICE" ]; then
  echo "[migrate] $SERVICE not found, skipping"
  exit 0
fi

if grep -q '/root/openclaw' "$SERVICE"; then
  echo "[migrate] Patching $SERVICE: /root/openclaw → /root/.openclaw"
  sed -i 's|/root/openclaw|/root/.openclaw|g' "$SERVICE"
  systemctl daemon-reload
  echo "[migrate] Restarting openclaw..."
  systemctl restart openclaw
  systemctl status openclaw --no-pager
else
  echo "[migrate] Already up to date, nothing to do"
fi
