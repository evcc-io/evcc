#!/bin/bash
#make
env GOOS=linux GOARCH=arm make build
rsync --progress -e "ssh -i ~/.ssh/S5B" evcc sunny5@192.168.5.186:/home/sunny5/git/evcc/
#rsync evcc.yaml sunny5@192.168.5.186:/home/sunny5/git/evcc/
rsync --progress -e "ssh -i ~/.ssh/S5B" evcc.sunny5.yaml sunny5@192.168.5.186:/home/sunny5/git/evcc/
rsync --progress -e "ssh -i ~/.ssh/S5B" build-sunny5-config.js sunny5@192.168.5.186:/home/sunny5/git/evcc/
