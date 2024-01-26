#!/bin/sh
set -e

USER_CHOICE_CONFIG="/etc/evcc-userchoices.sh"
ETC_SERVICE="/etc/systemd/system/evcc.service"
USR_LOCAL_BIN="/usr/local/bin/evcc"
RESTART_FLAG_FILE="/tmp/.restartEvccOnUpgrade"

# Usage: askUserKeepFile <file>
# Return: 1 = keep, 0 = delete
askUserKeepFile() {
	while true; do
		echo "Shall '$1' be deleted? [Y/n]: "
		read answer
		case "$answer" in
			n*|N*)
				echo "Ok. We will keep that file. Keep in mind that you may need to alter it if any changes are done upstream. Your answer is saved for the future."
				return 1
				;;
			y*|Y*|"")
				echo "The file will be deleted."
				return 0
				;;
			*)
				;;
		esac
	done
}

if [ "$1" = "configure" ]; then
	KEEP_ETC_SERVICE=0
	KEEP_USR_LOCAL_BIN=0
	# If the user once said that he wants to keep the files
	# this choice file will include that information
	# and the user will no longer be asked if he wants to keep it
	if [ -f "$USER_CHOICE_CONFIG" ]; then
		. "$USER_CHOICE_CONFIG"
	fi

	# If the user previously decided that he doesn't want to keep
	# the files or if it's the first time, aks whether he want's to
	# keep the file
	if [ -f "$ETC_SERVICE" ] && [ "$KEEP_ETC_SERVICE" -eq 0 ]; then
		echo "An alternate service file was detected under '$ETC_SERVICE'."
		echo "This is probably due to a previous manual installation."
		echo "You probably want to delete this file now. Your evcc configuration stays untouched!"
		askUserKeepFile "$ETC_SERVICE" || KEEP_ETC_SERVICE=$?
	fi
	if [ -f "$USR_LOCAL_BIN" ] && [ "$KEEP_USR_LOCAL_BIN" -eq 0 ]; then
		echo "An alternate evcc binary was detected under '$USR_LOCAL_BIN'."
		echo "This is probably due to a previous manual installation."
		echo "You probably want to delete this file now. Your evcc configuration stays untouched!"
		askUserKeepFile "$USR_LOCAL_BIN" || KEEP_USR_LOCAL_BIN=$?
	fi
	# Save the user decision
	cat > "$USER_CHOICE_CONFIG" <<EOF
#!/bin/sh
KEEP_ETC_SERVICE=$KEEP_ETC_SERVICE
KEEP_USR_LOCAL_BIN=$KEEP_USR_LOCAL_BIN
EOF

	# Execute the user decision
	if [ -f "$ETC_SERVICE" ] && [ "$KEEP_ETC_SERVICE" -eq 0 ]; then
		echo "Deleting old service file '$ETC_SERVICE'"
		rm -v "$ETC_SERVICE"
	fi

	if [ -f "$USR_LOCAL_BIN" ] && [ "$KEEP_USR_LOCAL_BIN" -eq 0 ]; then
		echo "Deleting old evcc binary '$USR_LOCAL_BIN'"
		rm -v "$USR_LOCAL_BIN"
	fi
fi

if [ "$1" = "configure" ] || [ "$1" = "abort-upgrade" ] || [ "$1" = "abort-deconfigure" ] || [ "$1" = "abort-remove" ] ; then
	# This will only remove masks created by d-s-h on package removal.
	deb-systemd-helper unmask evcc.service >/dev/null || true

	# was-enabled defaults to true, so new installations run enable.
	if deb-systemd-helper --quiet was-enabled evcc.service; then
		# Enables the unit on first installation, creates new
		# symlinks on upgrades if the unit file has changed.
		deb-systemd-helper enable evcc.service >/dev/null || true
	else
		# Update the statefile to add new symlinks (if any), which need to be
		# cleaned up on purge. Also remove old symlinks.
		deb-systemd-helper update-state evcc.service >/dev/null || true
	fi

	# Restart only if it was already started
	if [ -d /run/systemd/system ]; then
		systemctl --system daemon-reload >/dev/null || true
		if [ -f $RESTART_FLAG_FILE ]; then
			deb-systemd-invoke start evcc.service >/dev/null || true
			rm $RESTART_FLAG_FILE
		elif [ -n "$2" ]; then
			deb-systemd-invoke try-restart evcc.service >/dev/null || true
		else
			deb-systemd-invoke start evcc.service >/dev/null || true
		fi
	fi
fi
