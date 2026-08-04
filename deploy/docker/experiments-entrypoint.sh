#!/usr/bin/env bash
# Resolve GT_CHROME to the Chromium the Playwright image ships. The
# path embeds the browser build number (/ms-playwright/chromium-<n>/),
# so it is discovered at start rather than hardcoded at build.
set -euo pipefail

if [ -z "${GT_CHROME:-}" ]; then
    resolved=$(find /ms-playwright -maxdepth 3 -type f -name chrome -path '*chrome-linux*' 2>/dev/null | head -1 || true)
    if [ -n "${resolved}" ]; then
        export GT_CHROME="${resolved}"
        echo "entrypoint: GT_CHROME=${GT_CHROME}"
    else
        echo "entrypoint: no Chromium found under /ms-playwright; browser tiers will record ABSENT" >&2
    fi
fi

exec "$@"
