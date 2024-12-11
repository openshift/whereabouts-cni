#!/bin/bash
# whereabout downstream entrypoint.sh to invoke corresponding
# ip-control-loop binary, based on RHEL version

set -e

function log()
{
	echo "$(date --iso-8601=seconds) [entrypoint.sh] ${1}"
}

# Collect host OS information
. /etc/os-release
rhelmajor=
# detect which version we're using in order to copy the proper binaries
case "${ID}" in
	rhcos|scos)
		rhelmajor=$(echo ${RHEL_VERSION} | cut -f 1 -d .)
		;;
	rhel|centos)
		rhelmajor=$(echo "${VERSION_ID}" | cut -f 1 -d .)
		;;
	fedora)
		if [ "${VARIANT_ID}" == "coreos" ]; then
			rhelmajor=8
		else
			log "FATAL ERROR: Unsupported Fedora variant=${VARIANT_ID}"
			exit 1
		fi
		;;
	*) log "FATAL ERROR: Unsupported OS ID=${ID}"; exit 1
		;;
esac

/usr/src/whereabouts/rhel${rhelmajor}/bin/ip-control-loop $@
