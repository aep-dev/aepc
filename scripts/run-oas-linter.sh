#!/usr/bin/env bash
set -e
if [ ! -f /tmp/spectral ]; then
    curl -L "https://github.com/stoplightio/spectral/releases/download/v6.14.2/spectral-linux-x64" -o /tmp/spectral
    chmod +x /tmp/spectral
fi

exec /tmp/spectral lint --ruleset "./spectral.yaml" "./example/bookstore/v1/bookstore_openapi.yaml"