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
	if [ -d "/tmp/oldevcc" ]; then
		rm -rf "/tmp/oldevcc"
	fi
	if [ -x "/usr/bin/deb-systemd-helper" ]; then
		deb-systemd-helper purge evcc.service >/dev/null || true
		deb-systemd-helper unmask evcc.service >/dev/null || true
	fi
fi

# Call /usr/bin/evcc checkconfig and capture the output
# if exit code is 0, then remove /tmp/oldevcc
# if exit code is not 0, then fail installation with error message and copy /tmp/oldevcc back to /etc/evcc
# if /tmp/oldevcc does not exist, then do nothing
if [ "$1" = "upgrade" ]; then
failInstallation=0
INTERACTIVE=0
# is shell script interactive?
if [ -t 0 ]; then
  INTERACTIVE=1
else
  INTERACTIVE=0
fi

if [ -f /tmp/oldevcc ] && [ $INTERACTIVE -eq 1 ]; then
    checkConfigOutput=$(/usr/bin/evcc checkconfig 2>&1 || true)
	oldevccversion=$(/tmp/oldevcc -v)
	newevccversion=$(/usr/bin/evcc -v)
	if echo "$checkConfigOutput" | grep -q "config valid"; then	
		if [ "$oldevccversion" = "$newevccversion" ]; then
			echo "--------------------------------------------------------------------------------"
			echo "Old evcc version detected. To apply the new version, please reinstall evcc. (e.g. apt-get reinstall evcc)"
			echo "--------------------------------------------------------------------------------"
		else 
			rm -rf /tmp/oldevcc
		fi
	else
		if [ "$oldevccversion" = "$newevccversion" ]; then
			echo "--------------------------------------------------------------------------------"
			echo "Old evcc version detected. To apply the new version, please reinstall evcc. (e.g. apt-get reinstall evcc)"
			echo "--------------------------------------------------------------------------------"
		else 
			echo "--------------------------------------------------------------------------------"
			echo "ERROR: your evcc configuration is not compatible with the new version. Please consider reading the release notes: https://github.com/evcc-io/evcc/releases"
			echo "checkconfig Output:" 
			echo "$checkConfigOutput"
			echo "--------------------------------------------------------------------------------"
    
			while true; do
				echo "Do you want to keep your old (working) evcc version? [Y/n]: "
				read choice
				case "$choice" in
					n*|N*|"")
						echo "We will keep the new version. Your evcc configuration stays untouched!"
						break
						;;
					y*|Y*)
						echo "The old version will be restored. Your evcc configuration stays untouched! Consider reinstalling the new version after fixing your configuration. (e.g. 	apt-get reinstall evcc)"
						cp -r /tmp/oldevcc /usr/bin/evcc
						failInstallation=1
						break
						;;
					*)
						;;
				esac
	   		done
		fi
	fi
fi 
# Fail installation if checkconfig command failed and the user decided to keep the old version to inform package manager about outcome
if [ $failInstallation -eq 1 ]; then
	exit 1
fi
fi
