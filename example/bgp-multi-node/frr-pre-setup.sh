#!/bin/bash
set -e

REPO="https://github.com/FRRouting/frr.git"
VERSION="frr-10.6.2"
DIR="frr"

echo "==> Cloning FRR ${VERSION}..."

if [ -d "${DIR}" ]; then
    echo "==> ${DIR} already exists, skipping clone"
else
    git clone --branch "${VERSION}" --depth 1 "${REPO}" "${DIR}"
fi

echo "==> Building frr:${VERSION#frr-}..."

cd "${DIR}"

docker build -f docker/alpine/Dockerfile -t "frr:${VERSION#frr-}" .

echo "==> Done"
echo "    Image: frr:${VERSION#frr-}"
