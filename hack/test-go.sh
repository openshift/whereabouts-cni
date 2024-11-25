#!/usr/bin/env bash
# single test: go test -v ./pkg/storage/
# without cache: go test -count=1 -v ./pkg/storage/
set -eo -x pipefail

while true; do
  case "$1" in
    -u|--use-static-check)
      USE_STATIC_CHECK=true
      break
      ;;
    *)
      echo "define argument -u (use static check)"
      exit 1
  esac
done

GO=${GO:-go}

echo "Running go vet ..."
${GO} vet --tags=test ./cmd/... ./pkg/...

BASEDIR=$(pwd)

#if static check true run this
echo USE_STATIC_CHECK=$USE_STATIC_CHECK

if [ $USE_STATIC_CHECK ]
then
  echo "Installing golang staticcheck ..."
  GOBIN=${BASEDIR}/bin go install honnef.co/go/tools/cmd/staticcheck@latest
  echo "Running golang staticcheck ..."
  ${BASEDIR}/bin/staticcheck --tags=test ./...
fi


