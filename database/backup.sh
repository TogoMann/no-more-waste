#!/usr/bin/env bash
set -e

DB_FILE="${1:-../backend/nomorewaste.db}"
BACKUP_DIR="${2:-./backups}"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"

mkdir -p "$BACKUP_DIR"

if [ ! -f "$DB_FILE" ]; then
  echo "Base introuvable: $DB_FILE"
  exit 1
fi

if command -v sqlite3 >/dev/null 2>&1; then
  sqlite3 "$DB_FILE" ".backup '$BACKUP_DIR/nomorewaste_$TIMESTAMP.db'"
else
  cp "$DB_FILE" "$BACKUP_DIR/nomorewaste_$TIMESTAMP.db"
fi

echo "Sauvegarde creee: $BACKUP_DIR/nomorewaste_$TIMESTAMP.db"

ls -1t "$BACKUP_DIR"/nomorewaste_*.db | tail -n +8 | xargs -r rm --
