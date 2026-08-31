#!/usr/bin/env bash
set -e

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_DIR="$ROOT_DIR/frontend"

if [ -f "$ROOT_DIR/.env" ]; then
  set -a
  . "$ROOT_DIR/.env"
  set +a
fi

API_PORT="${API_PORT:-9081}"
FRONTEND_PORT="${FRONTEND_PORT:-9080}"
DEV_PORT="${DEV_PORT:-5173}"
PUBLIC_URL="${PUBLIC_URL:-http://localhost:$DEV_PORT}"

GREEN='\033[0;32m'
RESET='\033[0m'

info() { echo -e "${GREEN}==> $1${RESET}"; }

require() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Erreur: '$1' est requis mais introuvable."
    exit 1
  fi
}

info "Verification des prerequis"
require go
require node
require npm

info "Compilation du backend Go"
cd "$BACKEND_DIR"
go mod tidy
go build -o nomorewaste-server .

info "Installation des dependances frontend"
cd "$FRONTEND_DIR"
npm install

info "Build du frontend Vue"
npm run build

info "Demarrage du backend sur le port $API_PORT"
cd "$BACKEND_DIR"
PORT="$API_PORT" \
PUBLIC_URL="$PUBLIC_URL" \
DB_PATH="$BACKEND_DIR/nomorewaste.db" \
SCHEMA_PATH="$ROOT_DIR/database/schema.sql" \
SEED_PATH="$ROOT_DIR/database/seed.sql" \
./nomorewaste-server &
BACKEND_PID=$!

trap "kill $BACKEND_PID 2>/dev/null" EXIT

sleep 2

info "Demarrage du frontend (Vite) sur le port $DEV_PORT"
cd "$FRONTEND_DIR"
VITE_API_TARGET="http://localhost:$API_PORT" \
VITE_DEV_PORT="$DEV_PORT" \
npm run dev -- --host --port "$DEV_PORT"

wait $BACKEND_PID
