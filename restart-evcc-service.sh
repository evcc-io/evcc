#!/bin/bash

function quit {
   printf "\nAbort! The service was not restarted.\n"
   exit
}

function restart {
   printf "\nRestarting evcc.service now...\n"
   sudo systemctl restart evcc.service

   sleep 1

   printf "\nDone! New Service status:\n"
   sudo systemctl status evcc.service
}

printf "\n"
while true; do
    read -p "Do you really wish to restart the evcc.service? " yn
    case $yn in
        [Yy]* ) restart; break;;
        [Nn]* ) quit;;
        * ) echo "Please answer yes or no. ";;
    esac
done
