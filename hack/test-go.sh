#!/usr/bin/env bash
# single test: go test -v ./pkg/storage/
# without cache: go test -count=1 -v ./pkg/storage/
set -eox pipefail

#temp just for testing
echo "Starting Test"
echo "go path ($GOPATH)"
echo "go root ($GOROOT)"

GO=${GO:-go}
SKIP_STATIC_CHECK=$false
#parse args
while [[ $# -gt 0 ]]; do
  case "$1" in
    -s|--skip-static-check)
      SKIP_STATIC_CHECK=true
      shift
      ;;
    *)
      echo "Invalid arguement: $1"
      echo "Usage: $0 [-s|--skip-static-check]"
      exit 1
  esac
done



BASEDIR=$(pwd)

if [ $SKIP_STATIC_CHECK ]
then
    echo "Skipped golang staticcheck"
    echo "Skipped go vet"
else
  echo "Installing golang staticcheck ..."
  GOBIN=${BASEDIR}/bin go install honnef.co/go/tools/cmd/staticcheck@latest
  echo "Running golang staticcheck ..."
  ${BASEDIR}/bin/staticcheck --tags=test ./...

  echo "Running go vet ..."
  ${GO} vet --tags=test ./cmd/... ./pkg/...
fi

echo "Running go tests..."
KUBEBUILDER_ASSETS="$(pwd)/bin" ${GO} test \
    --tags=test \
    -v \
    -covermode=count \
    -coverprofile=coverage.out \
    $(${GO} list ./... | grep -v e2e | tr "\n" " ")
