#!/usr/bin/env bash
# Build and push Sentinel images to reg.mailtarget.dev/mailtarget/
#
# Usage:
#   ./scripts/docker-push.sh              # push :latest
#   TAG=v1.0.0 ./scripts/docker-push.sh   # push :v1.0.0 and :latest
#
# Login first if needed:
#   docker login reg.mailtarget.dev

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REGISTRY="${REGISTRY:-reg.mailtarget.dev/mailtarget}"
TAG="${TAG:-latest}"

cd "${ROOT}"

echo "==> Building ${REGISTRY}/sentinel:${TAG}"
docker build -t "${REGISTRY}/sentinel:${TAG}" -f Dockerfile .

echo "==> Building ${REGISTRY}/dashboard-sentinel:${TAG}"
docker build -t "${REGISTRY}/dashboard-sentinel:${TAG}" -f Dockerfile.dashboard .

if [[ "${TAG}" != "latest" ]]; then
  docker tag "${REGISTRY}/sentinel:${TAG}" "${REGISTRY}/sentinel:latest"
  docker tag "${REGISTRY}/dashboard-sentinel:${TAG}" "${REGISTRY}/dashboard-sentinel:latest"
fi

echo "==> Pushing ${REGISTRY}/sentinel:${TAG}"
docker push "${REGISTRY}/sentinel:${TAG}"

echo "==> Pushing ${REGISTRY}/dashboard-sentinel:${TAG}"
docker push "${REGISTRY}/dashboard-sentinel:${TAG}"

if [[ "${TAG}" != "latest" ]]; then
  echo "==> Pushing ${REGISTRY}/sentinel:latest"
  docker push "${REGISTRY}/sentinel:latest"
  echo "==> Pushing ${REGISTRY}/dashboard-sentinel:latest"
  docker push "${REGISTRY}/dashboard-sentinel:latest"
fi

echo "==> Done."
echo "    sentinel:           ${REGISTRY}/sentinel:${TAG}"
echo "    dashboard-sentinel:   ${REGISTRY}/dashboard-sentinel:${TAG}"
