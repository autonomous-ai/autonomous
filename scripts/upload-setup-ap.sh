#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SETUP_FILE="${SCRIPT_DIR}/setup-ap.sh"

# Bucket and path matching https://storage.googleapis.com/s3-autonomous-upgrade-3/${BUCKET_PREFIX}/setup-ap.sh
source "${SCRIPT_DIR}/ota-config.sh"
GCS_PATH="${GCS_PATH:-${BUCKET_PREFIX}/setup-ap.sh}"

if [[ ! -f "$SETUP_FILE" ]]; then
  echo "Error: setup-ap.sh not found at $SETUP_FILE"
  exit 1
fi

echo "========== Upload setup-ap.sh to Google Cloud Storage (no-cache) =========="
gsutil -h "Cache-Control:no-cache, no-store, must-revalidate" cp "$SETUP_FILE" "gs://${GCS_BUCKET}/${GCS_PATH}"
echo "Done: gs://${GCS_BUCKET}/${GCS_PATH}"
