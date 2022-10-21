#!/bin/sh
set -e

# started as hassio addon
HASSIO_OPTIONSFILE=/data/options.json

if [ -f ${HASSIO_OPTIONSFILE} ]; then
    CONFIG=$(grep -o '"config_file": "[^"]*' ${HASSIO_OPTIONSFILE} | grep -o '[^"]*$')
    echo "Using config file: ${CONFIG}"

    SQLITE_FILE=$(grep -o '"sqlite_file": "[^"]*' ${HASSIO_OPTIONSFILE} | grep -o '[^"]*$')

    if [ ! -f "${CONFIG}" ]; then
        echo "Config not found. Please create a config under ${CONFIG}."
        echo "For details see evcc documentation at https://github.com/evcc-io/evcc#readme."
    else
        if [ "${SQLITE_FILE}" ]; then
            echo "starting evcc: 'evcc --config ${CONFIG} --sqlite ${SQLITE_FILE}'"
            exec evcc --config "${CONFIG}" --sqlite "${SQLITE_FILE}"
        else
            echo "starting evcc: 'evcc --config ${CONFIG}'"
            exec evcc --config "${CONFIG}"
        fi
    fi
else
    if [ "$1" == '"evcc"' ] || expr "$1" : '-*' > /dev/null; then
        exec evcc "$@"
    else
        exec "$@"
    fi
fi
