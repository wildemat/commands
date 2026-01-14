#!/bin/bash
# Process: Kibana Stack
# Starts Kibana in traditional stack mode

set -e

export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"

cd "${KIBANA_DIR:-$HOME/workplace/kibana}"
nvm use

export KBN_OPTIMIZER_USE_MAX_AVAILABLE_RESOURCES=false

yarn start --config=config/kibana.stack.dev.yml --server.port=5611 --no-optimizer "$@"

