#!/bin/sh

echo "evcc addon Startup script run.sh"

CONFIG=$(jq -r .config_file /data/options.json)

echo "Using config ${CONFIG}"

if [ ! -f ${CONFIG} ]; then
    echo "config not found. Please see evcc documentation and /config/evcc.dist.yaml for example configuration."
    cp /evcc/evcc.dist.yaml /config/evcc.dist.yaml
fi

echo "starting evcc --config ${CONFIG}"
evcc --config ${CONFIG}