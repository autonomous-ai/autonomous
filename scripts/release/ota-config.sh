#!/usr/bin/env bash
# Shared GCS config for the upload + release scripts.
#
# These scripts run on a developer's machine (not on the device), so they can
# all source this one file. To retarget releases at a different bucket or path
# prefix, edit the values HERE — every upload-*.sh / tag-release.sh picks them up.
#
# Values use `:=` so an environment override still wins (CI, ad-hoc runs); this
# file is just the default + the single edit point.

# Repo paths — resolved HERE so every release script that sources this file
# shares ONE definition (no per-script SCRIPT_DIR/ROOT_DIR boilerplate that can
# drift out of sync when the scripts move). ${BASH_SOURCE[0]} is THIS file; it
# lives in scripts/release/, so ../.. is the repo root regardless of which
# script sourced it or the caller's working directory.
RELEASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${RELEASE_DIR}/../.." && pwd)"

# Target bucket for all OTA artifacts.
: "${GCS_BUCKET:=s3-autonomous-upgrade-3}"

# Path namespace inside the bucket — the single knob to re-namespace every
# artifact (OTA zips, metadata.json, skills, hooks, setup scripts). Referenced
# as ${BUCKET_PREFIX} wherever a bucket path is built.
: "${BUCKET_PREFIX:=os}"
