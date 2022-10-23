#!/bin/sh
#
# Executed before the installation of the new package
#
#   $1=install              : On installation
#   $1=upgrade              : On upgrade

set -e

EVCC_USER=evcc
EVCC_GROUP=evcc
EVCC_HOME="/var/lib/$EVCC_USER"
RESTART_FLAG_FILE=/var/lib/evcc/.restartOnUpgrade
COPIED_FLAG=/root/.evcc/.copiedToEvccUser

copyDbToUserDir() {
  if [ -d /root/.evcc ] && [ ! -f $COPIED_FLAG ]; then
    if [ -d /run/systemd/system ] && /bin/systemctl status evcc.service > /dev/null 2>&1; then
      deb-systemd-invoke stop evcc.service >/dev/null || true
      touch ${RESTART_FLAG_FILE}
    fi
    /bin/cp -Rp /root/.evcc/evcc.db "$EVCC_HOME"
    chown "$EVCC_USER:$EVCC_GROUP" "$EVCC_HOME/evcc.db"
    touch "$COPIED_FLAG"
  fi
  return 0
}

if [ "$1" = "install" ] || [ "$1" = "upgrade" ]; then
    if ! getent group "$EVCC_GROUP" > /dev/null 2>&1 ; then
      addgroup --system "$EVCC_GROUP" --quiet
    fi
    if ! getent passwd "$EVCC_USER" > /dev/null 2>&1 ; then
      adduser --quiet --system --ingroup "$EVCC_GROUP" --no-create-home \
      --disabled-password --shell /bin/false \
      --gecos "evcc runtime user" --home "$EVCC_HOME" "$EVCC_USER"
    else
      homedir=$(getent passwd "$EVCC_USER" | cut -d: -f4)
      if [ "$homedir" != "$EVCC_HOME" ]; then
        process=$(pgrep -u "$EVCC_USER") || true
        if [ -z "$process" ]; then
          usermod -d "$EVCC_HOME" "$EVCC_USER"
        else 
          echo "Warning: evcc's home directory is incorrect ($homedir)"
          echo "but can't be changed because another process ($process) is using it."
        fi
      fi
    fi
fi

if [ "$1" = "upgrade" ]; then
    copyDbToUserDir
fi

exit 0