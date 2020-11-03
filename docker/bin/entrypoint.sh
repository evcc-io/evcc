#!/bin/sh
set -e

# started as hassio addon
HASSIO_OPTIONSFILE=/data/options.json
if [ -f ${HASSIO_OPTIONSFILE} ]; then
    CONFIG=$(jq -r .config_file ${HASSIO_OPTIONSFILE})
    echo "Using configurationfile: ${CONFIG}"
    if [ ! -f ${CONFIG} ]; then
        echo "config not found. Please see evcc documentation and create a configurationfile with the name ${CONFIG}"
    else
        echo "starting evcc: 'evcc --config ${CONFIG}'"
        exec evcc --config ${CONFIG}
    fi
else
    if [ "$1" == '"evcc"' ] || expr "$1" : '-*' > /dev/null; then
        exec evcc "$@"
    else
        exec "$@"
    fi
fi
