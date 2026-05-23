#!/bin/bash
# whereabout downstream entrypoint.sh to invoke the ip-control-loop binary

set -e

/usr/src/whereabouts/bin/ip-control-loop $@
