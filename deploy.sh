#!/bin/bash
env GOOS=linux GOARCH=arm make build
rsync evcc sunny5@192.168.5.32:/home/sunny5/git/evcc/
rsync evcc.yaml sunny5@192.168.5.32:/home/sunny5/git/evcc/