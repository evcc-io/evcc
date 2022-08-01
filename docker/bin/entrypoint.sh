#!/bin/sh
set -e

# started as hassio addon
HASSIO_OPTIONSFILE=/data/options.json

if [ -f ${HASSIO_OPTIONSFILE} ]; then
    CONFIG=$(grep config_file ${HASSIO_OPTIONSFILE}| cut -d ':' -f 2 | sed s/[\"}]//g )
    echo "Using config file: ${CONFIG}"

    if [ ! -f ${CONFIG} ]; then
        echo "Config not found. Please create a config under ${CONFIG}."
        echo "For details see evcc documentation at https://github.com/evcc-io/evcc#readme."
    else
        echo "starting evcc: 'evcc --config ${CONFIG}'"
        exec evcc --config ${CONFIG}
    fi
elif [ -f ${CONFIG} ]; then
    echo "starting evcc: 'evcc --config ${CONFIG}'"
    exec evcc --config ${CONFIG}
else
    if [ "$1" == '"evcc"' ] || expr "$1" : '-*' > /dev/null; then
        exec evcc "$@"
    else
        exec "$@"
    fi
fi
