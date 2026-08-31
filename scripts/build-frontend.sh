#!/usr/bin/env bash
set -e

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> Compilation du frontend dans un conteneur jetable"
docker build --no-cache -f "$ROOT_DIR/frontend/Dockerfile" --target build -t nmw-frontend-builder "$ROOT_DIR"

echo "==> Extraction du dossier dist"
rm -rf "$ROOT_DIR/frontend/dist"
CONTAINER_ID="$(docker create nmw-frontend-builder)"
docker cp "$CONTAINER_ID:/app/dist" "$ROOT_DIR/frontend/dist"
docker rm "$CONTAINER_ID" >/dev/null
docker rmi nmw-frontend-builder >/dev/null 2>&1 || true

echo "==> Termine : $(find "$ROOT_DIR/frontend/dist" -type f | wc -l) fichiers dans frontend/dist"
