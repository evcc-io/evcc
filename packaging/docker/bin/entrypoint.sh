#!/bin/sh
set -e

# started as hassio addon
HASSIO_OPTIONSFILE=/data/options.json

if [ -f ${HASSIO_OPTIONSFILE} ]; then
	CONFIG=$(grep -o '"config_file": "[^"]*' ${HASSIO_OPTIONSFILE} | grep -o '[^"]*$')
	SQLITE_FILE=$(grep -o '"sqlite_file": "[^"]*' ${HASSIO_OPTIONSFILE} | grep -o '[^"]*$')

	# Config File Migration
	# If there is no config file found in '/config' we copy it from '/homeassistant' and rename the old config file to .migrated
	if [ ! -f "${CONFIG}" ]; then
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

	echo "Using config file: ${CONFIG}"
	if [ ! -f "${CONFIG}" ]; then
		echo "Config not found. Please create a config under ${CONFIG}."
		echo "For details see evcc documentation at https://github.com/evcc-io/evcc#readme."
	else
		if [ "${SQLITE_FILE}" ]; then
			echo "starting evcc: 'EVCC_DATABASE_DSN=${SQLITE_FILE} evcc --config ${CONFIG}'"
			exec env EVCC_DATABASE_DSN="${SQLITE_FILE}" evcc --config "${CONFIG}"
		else
			echo "starting evcc: 'evcc --config ${CONFIG}'"
			exec evcc --config "${CONFIG}"
		fi
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
