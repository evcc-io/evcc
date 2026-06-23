#!/bin/sh
set -e

# started as hassio addon
HASSIO_OPTIONSFILE=/data/options.json

if [ -f ${HASSIO_OPTIONSFILE} ]; then
	CONFIG=$(grep -o '"config_file": "[^"]*' ${HASSIO_OPTIONSFILE} | grep -o '[^"]*$' || true)
	SQLITE_FILE=$(grep -o '"sqlite_file": "[^"]*' ${HASSIO_OPTIONSFILE} | grep -o '[^"]*$' || true)

	# Resolve database path: prefer configured path, otherwise use default if present
	DEFAULT_DB="/data/evcc.db"
	DB_PATH=""
	if [ -n "${SQLITE_FILE}" ]; then
		DB_PATH="${SQLITE_FILE}"
	elif [ -f "${DEFAULT_DB}" ]; then
		DB_PATH="${DEFAULT_DB}"
	fi

	# Config File Migration
	# If there is no config file found in '/config' we copy it from '/homeassistant' and rename the old config file to .migrated
	if [ -n "${CONFIG}" ] && [ ! -f "${CONFIG}" ]; then
		CONFIG_OLD=$(echo "${CONFIG}" | sed 's#^/config#/homeassistant#')
		if [ -f "${CONFIG_OLD}" ]; then
			mkdir -p "$(dirname "${CONFIG}")" && cp "${CONFIG_OLD}" "${CONFIG}"
			mv "${CONFIG_OLD}" "${CONFIG_OLD}.migrated"
			echo "Moving old config file '${CONFIG_OLD}' to new location '${CONFIG}', appending '.migrated' to old config file! Old file can safely be deleted by user."
		fi
	fi

	# Database File Migration (optional, in case it is in /config)
	# Only in case the user put her DB into the '/config' folder instead of default '/data' we will migrate it aswell
	if [ "${SQLITE_FILE#/config}" != "${SQLITE_FILE}" ] && [ ! -f "${SQLITE_FILE}" ]; then
		SQLITE_FILE_OLD=$(echo "${SQLITE_FILE}" | sed 's#^/config#/homeassistant#')
		if [ -f "${SQLITE_FILE_OLD}" ]; then
			mkdir -p "$(dirname "${SQLITE_FILE}")" && cp "${SQLITE_FILE_OLD}" "${SQLITE_FILE}"
			mv "${SQLITE_FILE_OLD}" "${SQLITE_FILE_OLD}.migrated"
			echo "Moving old db file '${SQLITE_FILE_OLD}' to new location '${SQLITE_FILE}', appending '.migrated' to old db file! Old file can safely be deleted by user."
		fi
	fi

	# Status overview (decoupled)
	if [ -n "${CONFIG}" ]; then
		if [ -f "${CONFIG}" ]; then
			echo "Config: configured (${CONFIG}), exists"
		else
			echo "Config: configured (${CONFIG}), missing"
		fi
	else
		echo "Config: not configured"
	fi

	if [ -n "${SQLITE_FILE}" ]; then
		if [ -f "${SQLITE_FILE}" ]; then
			echo "Database: configured (${SQLITE_FILE}), exists"
		else
			echo "Database: configured (${SQLITE_FILE}), missing"
		fi
	else
		if [ -f "${DEFAULT_DB}" ]; then
			echo "Database: not configured; using default database: ${DEFAULT_DB} (add-on persistent storage)"
		else
			echo "Database: not configured; no default present"
		fi
	fi

	if [ -n "${CONFIG}" ] && [ -f "${CONFIG}" ]; then
		# Config file exists and is configured
		if [ -n "${DB_PATH}" ]; then
			exec env EVCC_DATABASE_DSN="${DB_PATH}" evcc --config "${CONFIG}"
		else
			exec evcc --config "${CONFIG}"
		fi
	elif [ -n "${CONFIG}" ]; then
		# Config file configured but doesn't exist
		exec env EVCC_DATABASE_DSN="${DB_PATH}" evcc
	elif [ -n "${DB_PATH}" ]; then
		# No config file configured, using database
		exec env EVCC_DATABASE_DSN="${DB_PATH}" evcc
	fi
else
	PUID="${PUID:-1000}"
	PGID="${PGID:-1000}"
	DATA_DIR="/root/.evcc" # documented mount path, kept for compatibility
	RUNAS=""

	# When started as root, repair ownership of the database directory and drop
	# privileges so the long-lived evcc process does not run as root. The evcc
	# user's home is set to /root so ~/.evcc/evcc.db keeps resolving to the
	# existing mount (su-exec sets HOME from the passwd entry, not the env).
	if [ "$(id -u)" = "0" ]; then
		getent group evcc > /dev/null 2>&1 || addgroup -g "$PGID" evcc 2> /dev/null || addgroup evcc
		getent passwd evcc > /dev/null 2>&1 || adduser -D -H -h /root -u "$PUID" -G evcc evcc 2> /dev/null || adduser -D -H -h /root -G evcc evcc

		mkdir -p "$DATA_DIR"
		chown -R "$PUID:$PGID" "$DATA_DIR"
		# fix a custom database directory if EVCC_DATABASE_DSN points elsewhere
		[ -n "$EVCC_DATABASE_DSN" ] && chown -R "$PUID:$PGID" "$(dirname "$EVCC_DATABASE_DSN")" 2> /dev/null || true

		# join the group(s) owning mounted serial devices (Modbus RTU, P1/SML readers);
		# --device keeps the host owner (often root:dialout), so non-root needs the group
		DEVICE_GIDS="$EVCC_DEVICE_GIDS"
		for dev in /dev/ttyUSB* /dev/ttyACM* /dev/ttyAMA* /dev/serial/by-id/*; do
			[ -e "$dev" ] || continue
			DEVICE_GIDS="$DEVICE_GIDS $(stat -c '%g' "$dev")"
		done
		for gid in $DEVICE_GIDS; do
			[ "$gid" = "$PGID" ] && continue
			grp=$(getent group "$gid" | cut -d: -f1)
			if [ -z "$grp" ]; then
				grp="evccdev$gid"
				addgroup -g "$gid" "$grp" 2> /dev/null || true
			fi
			addgroup evcc "$grp" 2> /dev/null || true
			echo "Serial: granting access to device group ${grp} (gid ${gid})"
		done

		# no explicit gid: su-exec uid:gid wipes supplementary groups (setgroups), the
		# uid-only form keeps them via getgrouplist; primary gid stays PGID from passwd
		RUNAS="su-exec $PUID"
	else
		# started via docker's own `user:`, already non-root: keep HOME on /root so
		# ~/.evcc/evcc.db still resolves to the mount (no passwd entry sets it here)
		RUNAS="env HOME=/root"
	fi

	if [ "$1" = 'evcc' ]; then
		shift
		exec $RUNAS evcc "$@"
	elif expr "$1" : '-.*' > /dev/null; then
		exec $RUNAS evcc "$@"
	else
		exec $RUNAS "$@"
	fi
fi
