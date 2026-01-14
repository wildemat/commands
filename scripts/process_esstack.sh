#!/bin/bash
# Process: Elasticsearch Stack
# Starts ES stack cluster for development

set -e

export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"

cd "${KIBANA_DIR:-$HOME/workplace/kibana}"
nvm use

yarn es snapshot --license trial --clean \
  -E http.port=9201 \
  -E transport.port=9301 \
  -E xpack.inference.elastic.url=https://localhost:8443 \
  -E xpack.inference.elastic.http.ssl.verification_mode=none \
  "$@"

