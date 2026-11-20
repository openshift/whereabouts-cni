#!/usr/bin/env bash
# OCPBUGS-99204: exercise CNO rendering of whereabouts-reconciler and verify
# the DaemonSet pods become Ready using the payload whereabouts image.
#
# CNO does not watch NAD CRs. RenderWhereaboutsAuxillary is set only when
# Network.operator spec.additionalNetworks contains a Raw CNI config with
# ipam.type=whereabouts (pkg/network/multus_ipam.go). A NAD alone will not
# deploy the DaemonSet.
set -euo pipefail

NS=openshift-multus
DS=whereabouts-reconciler
TIMEOUT_SECS=${TIMEOUT_SECS:-600}
NETWORK_NAME=whereabouts-ci

log() { echo "$(date -Iseconds) $*"; }
fail() {
  log "ERROR: $*"
  log ">>> debug: network.operator additionalNetworks"
  oc get network.operator cluster -o jsonpath='{.spec.additionalNetworks}' || true
  echo
  log ">>> debug: ${NS} daemonsets/pods"
  oc -n "${NS}" get ds,pods || true
  log ">>> debug: ${DS} describe"
  oc -n "${NS}" describe "ds/${DS}" || true
  log ">>> debug: cluster-network-operator logs (tail)"
  oc -n openshift-network-operator logs deploy/network-operator --tail=80 || true
  exit 1
}

wait_for_ds() {
  local deadline=$((SECONDS + TIMEOUT_SECS))
  while ! oc -n "${NS}" get "ds/${DS}" >/dev/null 2>&1; do
    if (( SECONDS >= deadline )); then
      fail "ds/${DS} was not created in ${NS} within ${TIMEOUT_SECS}s"
    fi
    sleep 10
  done
}

log ">>> adding whereabouts additionalNetwork to network.operator/cluster"
existing="$(oc get network.operator cluster -o jsonpath="{.spec.additionalNetworks[?(@.name==\"${NETWORK_NAME}\")].name}" || true)"
if [[ -n "${existing}" ]]; then
  log "additionalNetwork ${NETWORK_NAME} already present"
else
  current="$(oc get network.operator cluster -o jsonpath='{.spec.additionalNetworks}' || true)"
  # rawCNIConfig is a JSON string. Chain format matches CNO unit tests for
  # detectIPAMTypeRaw (ipam.type=whereabouts).
  if [[ -z "${current}" || "${current}" == "null" || "${current}" == "[]" ]]; then
    oc patch network.operator cluster --type=merge --patch "$(cat <<'EOF'
{"spec":{"additionalNetworks":[{"name":"whereabouts-ci","namespace":"default","type":"Raw","rawCNIConfig":"{\"cniVersion\":\"0.4.0\",\"name\":\"whereabouts-ci\",\"plugins\":[{\"type\":\"bridge\",\"ipam\":{\"type\":\"whereabouts\",\"range\":\"192.0.2.0/24\"}}]}"}]}}
EOF
)"
  else
    oc patch network.operator cluster --type=json --patch "$(cat <<'EOF'
[{"op":"add","path":"/spec/additionalNetworks/-","value":{"name":"whereabouts-ci","namespace":"default","type":"Raw","rawCNIConfig":"{\"cniVersion\":\"0.4.0\",\"name\":\"whereabouts-ci\",\"plugins\":[{\"type\":\"bridge\",\"ipam\":{\"type\":\"whereabouts\",\"range\":\"192.0.2.0/24\"}}]}"}}]
EOF
)"
  fi
fi

log ">>> waiting for ${DS} DaemonSet in ${NS}"
wait_for_ds

log ">>> waiting for ${DS} rollout"
oc -n "${NS}" rollout status "ds/${DS}" --timeout="${TIMEOUT_SECS}s" \
  || fail "DaemonSet ${DS} did not roll out"

log ">>> waiting for ${DS} pods to become Ready"
oc -n "${NS}" wait pod -l "name=${DS}" --for=condition=Ready --timeout="${TIMEOUT_SECS}s" \
  || fail "whereabouts-reconciler pods did not become Ready"

waiting="$(oc -n "${NS}" get pods -l "name=${DS}" \
  -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.status.containerStatuses[*].state.waiting.reason}{"\n"}{end}' || true)"
if echo "${waiting}" | grep -q CrashLoopBackOff; then
  fail "whereabouts-reconciler pod is CrashLoopBackOff: ${waiting}"
fi

log ">>> checking whereabouts container logs for missing binaries"
for pod in $(oc -n "${NS}" get pods -l "name=${DS}" -o jsonpath='{.items[*].metadata.name}'); do
  logs="$(oc -n "${NS}" logs "${pod}" -c whereabouts --tail=200 || true)"
  if echo "${logs}" | grep -Eiq 'no such file or directory|exec format error|executable file not found'; then
    echo "${logs}"
    fail "pod ${pod} logs indicate a missing/failed binary (entrypoint/image layout regression)"
  fi
done

image="$(oc -n "${NS}" get ds "${DS}" -o jsonpath='{.spec.template.spec.containers[0].image}')"
log ">>> whereabouts-reconciler image: ${image}"
log "SUCCESS: whereabouts-reconciler is healthy"
