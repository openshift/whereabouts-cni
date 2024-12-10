#!/usr/bin/env bash

echo "Warning: This will not work without a running cluster / kubeconfig"

#can we just use the whereabouts kubeconfig here?
export KUBECONFIG=${KUBECONFIG:-${HOME}/} #need to find how to get the kubeconfig from ci cluster


FOCUS=$(echo ${@:1} | sed 's/ /\\s/g')
SKPPED_TESTS=""

pushd e2e
go mod download
go test -test.timeout 180m -v . \
        -ginko.v \
        -gingko.focus ${FOCUS:-.} \
        -ginkgo.timeout 3h \
        -ginkgo.flake-attempts ${FLAKE_ATTEMPTS:-2} \
        -gingko.skip="${SKIPPED_TESTS}" \
        -gingko.junit-report=${E2E_REPORT_DIR}/junit_${E2E_REPORT_PREFIX}report.xml \
        -provider skeleton \
        -kubeconfig ${KUBECONFIG} \
        ${NUM_NODES:+"--num-nodes=${NUM_NODES}"} \
        ${E2E_REPORT_DIR:+"--report-dir=${E2E_REPORT_DIR}"}
popd
