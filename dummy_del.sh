#!/bin/sh

if [ $# -ne 1 ]; then
	echo "$0 container-id"
	echo "example: $0 pod-foobar"
	exit
fi

export CNI_COMMAND="DEL"
export CNI_CONTAINERID="$1"
export CNI_NETNS="/run/netns/foobar"
export CNI_IFNAME="test1"
export CNI_PATH="$(pwd)"
export CNI_ARGS="K8S_POD_NAMESPACE=default;K8S_POD_NAME=$1"

./whereabouts < dummy_cni.conf
