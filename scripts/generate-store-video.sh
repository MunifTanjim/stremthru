#!/usr/bin/env bash

set -euo pipefail

function generate() {
  local -r name="${1}"
  shift
  local -r text="${1}"
  shift

  ./scripts/text-to-video.sh --output "./internal/store/video/${name}.mp4" --text "${text}" $@
}

generate "401" "Invalid Credentials" --indicator "!!!|"
generate "403" "Forbidden" --indicator "!!!|"
generate "500" "Something Went Wrong" --indicator "!!!|"
generate "download_failed" "Failed to Download" --indicator "!!!|"
generate "downloading" "Downloading to Store" --indicator ".|..|..."
generate "no_matching_file" "No Matching File" --indicator "!!!|"
