#!/bin/sh

cd "$(dirname "$0")/../pkg/apis"

[ -d googleapis ] || git clone https://github.com/googleapis/googleapis

[ -z "$(command -v protoc-gen-openapi)" ] && go install github.com/google/gnostic/cmd/protoc-gen-openapi@v0.7.0

protoc \
  -I/opt/homebrew/opt/protobuf/include \
  -I./googleapis \
  --proto_path=. \
  --go_out=. \
  --go_opt=paths=source_relative \
  --openapi_out=../../docs \
  --openapi_opt=title="" \
  --openapi_opt=version=beta1 \
  config.proto

yq '
del(
    .components.schemas.Status,
    .components.schemas.GoogleProtobufAny,
    .tags
),
.paths={}' -i ../../docs/openapi.yaml
