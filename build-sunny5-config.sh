#!/bin/bash

export LANGUAGE=en #important for arp -a to work correctly
node build-sunny5-config.js  evcc.sunny5.yaml ../Sunny5Lib/config.js
