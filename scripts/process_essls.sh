#!/bin/bash
# Process: Elasticsearch Serverless
# Starts ES serverless cluster for development

set -e

export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"

cd "${KIBANA_DIR:-$HOME/workplace/kibana}"
nvm use

yarn es serverless --projectType elasticsearch_general_purpose --clean --kill \
  -E xpack.inference.elastic.url=https://localhost:8443 \
  -E xpack.inference.elastic.http.ssl.verification_mode=none \
  "$@"

