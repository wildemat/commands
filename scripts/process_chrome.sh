#!/bin/bash
# Process: Chrome Browser
# Opens Chrome in incognito mode with a specific profile

set -e

PORT="${1:-5601}"
BROWSER_PATH="${BROWSER_PATH:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
PROFILE_DIR="/tmp/chrome-profile-${PORT}"

"$BROWSER_PATH" --incognito --user-data-dir="$PROFILE_DIR" "http://localhost:${PORT}"

