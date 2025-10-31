#!/bin/sh
set -e

# started as hassio addon
HASSIO_OPTIONSFILE=/data/options.json

if [ -f ${HASSIO_OPTIONSFILE} ]; then
	CONFIG=$(grep -o '"config_file": "[^"]*' ${HASSIO_OPTIONSFILE} | grep -o '[^"]*$')
	SQLITE_FILE=$(grep -o '"sqlite_file": "[^"]*' ${HASSIO_OPTIONSFILE} | grep -o '[^"]*$')
	SEMP_BASE_URL=$(grep -o '"SEMP_BASE_URL": "[^"]*' ${HASSIO_OPTIONSFILE} | grep -o '[^"]*$')

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
	
	# Determine evcc startup command based on available options
	EVCC_CMD="evcc"
	ENV_VARS=""
	
	# Add config if file exists
	if [ -f "${CONFIG}" ]; then
		EVCC_CMD="${EVCC_CMD} --config ${CONFIG}"
		echo "Config file found: ${CONFIG}"
	else
		echo "Config file not found at ${CONFIG}. Starting evcc without configuration file."
		echo "Setup required via web interface. For details see evcc documentation at https://github.com/evcc-io/evcc#readme."
	fi
	
	# Add database if specified
	if [ "${SQLITE_FILE}" ]; then
		ENV_VARS="${ENV_VARS} EVCC_DATABASE_DSN=${SQLITE_FILE}"
	fi
	
	# Add SEMP base URL if specified
	if [ "${SEMP_BASE_URL}" ]; then
		ENV_VARS="${ENV_VARS} SEMP_BASE_URL=${SEMP_BASE_URL}"
	fi
	
	# Execute evcc with collected parameters
	if [ "${ENV_VARS}" ]; then
		echo "starting evcc: 'env${ENV_VARS} ${EVCC_CMD}'"
		exec env${ENV_VARS} ${EVCC_CMD}
	else
		echo "starting evcc: '${EVCC_CMD}'"
		exec ${EVCC_CMD}
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
