#!/bin/bash
# 打包, linux 64 & mac os

VERSION=$(git describe --tags)
echo "build $VERSION..."

mkdir dist
mkdir releases
gox -osarch="linux/amd64" -osarch="darwin/amd64" -ldflags "-X main.Version=$VERSION" -output "dist/{{.OS}}_{{.Arch}}/burrow-exporter"

for i in dist/* ; do
  if [ -d "$i" ]; then
   ARCH=$(basename "$i")
   zip releases/burrow-exporter_$VERSION_$ARCH.zip dist/$ARCH/burrow-exporter
  fi
done

