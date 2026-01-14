#!/bin/bash
# Process: Kibana Serverless
# Starts Kibana in serverless mode

set -e

export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"

cd "${KIBANA_DIR:-$HOME/workplace/kibana}"
nvm use

export KBN_OPTIMIZER_USE_MAX_AVAILABLE_RESOURCES=false

yarn serverless-es --config=config/kibana.serverless.dev.yml --server.port=5601 --no-optimizer "$@"

