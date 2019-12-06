#!/bin/bash
set -e

if [ "$1" = "evcc" ]; then
    exec /go/bin/evcc "$@"
fi

exec "$@"
