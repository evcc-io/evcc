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
RESTART_FLAG_FILE="/tmp/.restartEvccOnUpgrade"

copyDbToUserDir() {
  CURRENT_USER=$(systemctl show -pUser evcc | cut -d= -f2)
  if [ -z "$CURRENT_USER" ]; then
  	CURRENT_USER=root
  fi
  CURRENT_HOME=$(getent passwd "$CURRENT_USER" | cut -d: -f6)
  COPIED_FLAG="$CURRENT_HOME/.evcc/.copiedToEvccUser"
  if [ -f "$CURRENT_HOME/.evcc/evcc.db" ] && [ ! -f "$COPIED_FLAG" ]; then
    if [ -f "$EVCC_HOME/evcc.db" ]; then
      echo "--------------------------------------------------------------------------------"
      echo "Not copying $CURRENT_HOME/.evcc/evcc.db to $EVCC_HOME/evcc.db, since there is"
      echo "already a database there."
      echo "Either delete one of the databases or run 'touch $COPIED_FLAG' to keep both,"
      echo "then restart installation."
      echo "Hint: usually the larger one is the one to keep."
      ls -la "$CURRENT_HOME/.evcc/evcc.db" "$EVCC_HOME/evcc.db"
      echo "--------------------------------------------------------------------------------"
      exit 1
    else
      cp -Rp "$CURRENT_HOME"/.evcc/evcc.db "$EVCC_HOME"
    fi
    chown "$EVCC_USER:$EVCC_GROUP" "$EVCC_HOME/evcc.db"
    touch "$COPIED_FLAG"
    if [ -n "$(ls -A /etc/systemd/system/evcc.service.d 2>/dev/null)" ]; then
        echo "--------------------------------------------------------------------------------"
		echo "You have overrides defined in /etc/systemd/system/evcc.service.d."
		echo "This update changes the evcc user to 'evcc' (from root) and the database file"
		echo "to '/var/lib/evcc/evcc.db"
		echo "Make sure that you neither override 'User' nor 'ExecStart'"
		echo "Hint: you can delete all overrides with 'systemctl revert evcc'"
		echo "As a precaution, evcc is not started even if it was previously started."
        echo "--------------------------------------------------------------------------------"
		rm -f "$RESTART_FLAG_FILE"
	else
        echo "--------------------------------------------------------------------------------"
		echo "NOTE: evcc user has changed from $CURRENT_USER to $EVCC_USER, db has been copied to new"
		echo "directory $EVCC_HOME/evcc.db, old db in $CURRENT_USER/.evcc has been retained."
      	echo "--------------------------------------------------------------------------------"
    fi
  fi
  return 0
}

if [ "$1" = "install" ] || [ "$1" = "upgrade" ]; then
	if [ -d /run/systemd/system ] && /bin/systemctl status evcc.service > /dev/null 2>&1; then
	  deb-systemd-invoke stop evcc.service >/dev/null || true
	  touch "$RESTART_FLAG_FILE"
	fi
    if ! getent group "$EVCC_GROUP" > /dev/null 2>&1 ; then
      addgroup --system "$EVCC_GROUP" --quiet
    fi
    if ! getent passwd "$EVCC_USER" > /dev/null 2>&1 ; then
      adduser --quiet --system --ingroup "$EVCC_GROUP" \
      --disabled-password --shell /bin/false \
      --gecos "evcc runtime user" --home "$EVCC_HOME" "$EVCC_USER"
      chown -R "$EVCC_USER:$EVCC_GROUP" "$EVCC_HOME"
      adduser --quiet "$EVCC_USER" dialout
    else
      adduser --quiet "$EVCC_USER" dialout
      homedir=$(getent passwd "$EVCC_USER" | cut -d: -f4)
      if [ "$homedir" != "$EVCC_HOME" ]; then
      	mkdir -p "$EVCC_HOME"
      	chown "$EVCC_USER:$EVCC_GROUP" "$EVCC_HOME"
        process=$(pgrep -u "$EVCC_USER") || true
        if [ -z "$process" ]; then
          usermod -d "$EVCC_HOME" "$EVCC_USER"
          if [ -f "$homedir/.evcc/evcc.db" ]; then
          	cp "$homedir/.evcc/evcc.db" "$EVCC_HOME" && touch "$homedir/.evcc/.copiedToEvccUser"
          fi
        else
      	  echo "--------------------------------------------------------------------------------"
          echo "Warning: evcc's home directory is incorrect ($homedir)"
          echo "but can't be changed because another process ($process) is using it."
          echo "Stop offending process(es), then restart installation"
          echo "Note that you should NOT use the evcc user as login user, since that will"
          echo "inevitably lead to this error."
          echo "in that case, please create a different user as login user."
          echo "--------------------------------------------------------------------------------"
          exit 1
        fi
      fi
    fi
fi

if [ "$1" = "upgrade" ]; then
    copyDbToUserDir
fi

exit 0
