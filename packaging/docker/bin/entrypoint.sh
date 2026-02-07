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
	if [ "$1" = 'evcc' ]; then
		shift
		exec evcc "$@"
	elif expr "$1" : '-.*' > /dev/null; then
		exec evcc "$@"
	else
		exec "$@"
	fi
fi
