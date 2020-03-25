#!/bin/sh

set -e

AUDI_USER=$(cat audi.json | jq -r .user)
AUDI_PASS=$(cat audi.json | jq -r .pass)
python3 audi.py

acctoken=$(cat token.json | jq -r .access_token)
vin=$(cat audi.json | jq -r .vin)
battsoc=$(curl -s --header "Accept: application/json" --header "X-App-Name: eRemote" --header "X-App-Version: 1.0.0" --header "User-Agent: okhttp/2.3.0" --header "Authorization: AudiAuth 1 $acctoken" https://msg.audi.de/fs-car/bs/batterycharge/v1/Audi/DE/vehicles/$vin/charger)
soclevel=$(echo $battsoc | jq .charger.status.batteryStatusData.stateOfCharge.content)
echo $soclevel
