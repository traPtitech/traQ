#!/usr/bin/env bash

set -eu

cd ../../

if type spectral >/dev/null 2>&1; then
    spectral lint -r .spectral.yml -q docs/v3-api.yaml
else
  	docker run --rm -it -v $(pwd):/tmp stoplight/spectral lint -r /tmp/.spectral.yml -q /tmp/docs/v3-api.yaml
fi
