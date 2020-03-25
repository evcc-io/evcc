#!/bin/sh
set -e

if [ "$1" == '"evcc"' ] || expr "$1" : '-*' > /dev/null; then
    exec evcc "$@"
else
    exec "$@"
fi
