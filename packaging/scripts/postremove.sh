#!/bin/sh
set -e

if [ -d /run/systemd/system ]; then
	systemctl --system daemon-reload >/dev/null || true
fi

if [ "$1" = "remove" ]; then
	if [ -x "/usr/bin/deb-systemd-helper" ]; then
		deb-systemd-helper mask evcc.service >/dev/null || true
	fi
fi

if [ "$1" = "purge" ]; then
	if [ -x "/usr/bin/deb-systemd-helper" ]; then
		deb-systemd-helper purge evcc.service >/dev/null || true
		deb-systemd-helper unmask evcc.service >/dev/null || true
	fi
fi

# if interactive: call `/usr/bin/evcc checkconfig` and check the return code (newer version)
# if return code is 0, do nothing
# else: Ask user if he wants to keep the old version (working) or the new version (not working) 
# Remember the choice with /tmp/.evccrollback and fail new-postrm failed-upgrade old-version new-version to initiate dpkg's rollback
if [ "$1" = "upgrade" ] && [ -t 0 ]; then
	if ! /usr/bin/evcc checkconfig > /dev/null; then
		echo "--------------------------------------------------------------------------------"
		echo "ERROR: your configuration is not compatible with the new version:"
		/usr/bin/evcc checkconfig --log error || true
		echo "Please consult the release notes: https://github.com/evcc-io/evcc/releases"
		echo "--------------------------------------------------------------------------------"

		while true; do
			echo "Do you want to keep your old (working) version? [Y/n]: "
			read choice
			case "$choice" in
				n*|N*|"")
					echo "We will keep the new version. Your configuration stays untouched!"
					break
					;;
				y*|Y*)
					echo "The old version will be restored. Your configuration stays untouched! Following errors are intended:"
					touch /tmp/.evccrollback
					exit 1
					break
					;;
				*)
					;;
			esac
		done
	fi
fi 

# if upgrade goal fails, new-postrm failed-upgrade old-version new-version is called. It should fail to initiate rollback
if [ "$1" = "failed-upgrade" ]; then
	if [ -f "/tmp/.evccrollback" ]; then
		rm "/tmp/.evccrollback"
		exit 1
	fi
fi