#!/bin/sh
set -e

if [ -d /run/systemd/system ] && [ "$1" = remove ]; then
	deb-systemd-invoke stop evcc.service > /dev/null || true
fi

# If package is upgraded, back up the database
if [ "$1" = "upgrade" ]; then
	# Get the currently installed version of evcc package
	CURRENT_VERSION=$(dpkg-query -W -f='${Version}' evcc 2>/dev/null)
	if [ -z "${CURRENT_VERSION}" ]; then
		CURRENT_VERSION="unknown"
	fi

	# Backup database
	EVCC_DB="/var/lib/evcc/evcc.db"
	if [ -f "$EVCC_DB" ]; then
		BACKUP_DIR="/var/backups/evcc"
		mkdir -p "$BACKUP_DIR"
		BACKUP_FILE="$BACKUP_DIR/evcc.db.${CURRENT_VERSION}.$(date +%Y%m%d-%H%M%S).bak"
		echo "Backing up evcc database to $BACKUP_FILE"
		cp -p "$EVCC_DB" "$BACKUP_FILE"
	fi
fi
