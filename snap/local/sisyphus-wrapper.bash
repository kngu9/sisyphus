#!/bin/bash
                               
set -eu

export HOME=$SNAP_USER_COMMON

CONFIG=""

CONFIG_LOCATIONS="$SNAP_USER_COMMON/config.yaml $SNAP_COMMON/config.yaml"
for loc in $CONFIG_LOCATIONS; do
	if [ -f "$loc" ]; then
		export CONFIG=$loc
		break
	fi
done

if [ -z "$CONFIG" ]; then
	echo "config not found; searched: $CONFIG_LOCATIONS"
	exit 1
fi

exec $SNAP/bin/sisyphus
