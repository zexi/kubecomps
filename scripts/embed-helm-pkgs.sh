#!/bin/bash

rm -rf ./static
helm package ./manifests/helm/monitor-stack \
    --version v1 \
    --destination ./static
helm package ./manifests/helm/fluent-bit \
    --destination ./static
