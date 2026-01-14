#!/bin/bash
# Process: Kibana Optimizer
# Watches and compiles Kibana platform plugins

set -e

export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"

cd "${KIBANA_DIR:-$HOME/workplace/kibana}"
nvm use

node scripts/build_kibana_platform_plugins --watch

