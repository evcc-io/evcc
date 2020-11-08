#!/bin/sh
set -e

# started as hassio addon
HASSIO_OPTIONSFILE=/data/options.json

if [ -f ${HASSIO_OPTIONSFILE} ]; then
    CONFIG=$(jq -r .config_file ${HASSIO_OPTIONSFILE})
    echo "Using config file: ${CONFIG}"

    if [ ! -f ${CONFIG} ]; then
        echo "Config not found. Please create a config under ${CONFIG}."
        echo "For details see evcc documentation at https://github.com/andig/evcc#readme."
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
