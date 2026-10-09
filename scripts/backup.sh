#!/bin/sh
# Периодическое резервное копирование базы GophKeeper (сервис backup в docker-compose).
#
# Каждые BACKUP_INTERVAL (формат sleep: 30m, 6h, 1d; по умолчанию 24h) делает pg_dump
# в формате custom (-Fc, сжатый, восстанавливается pg_restore) в /backups и оставляет
# BACKUP_KEEP последних копий (по умолчанию 14). Первая копия — сразу при запуске.
#
# Секреты в копиях зашифрованы ключами пользователей, поэтому копии можно хранить
# где угодно: утечка копии равна утечке БД (возможен только офлайн-перебор паролей).
set -eu
# Копии содержат хеши ключей аутентификации и соли: читать их должен только владелец.
umask 077

interval="${BACKUP_INTERVAL:-24h}"
keep="${BACKUP_KEEP:-14}"
dir=/backups

case "$keep" in
  ''|*[!0-9]*) echo "BACKUP_KEEP must be a positive number" >&2; exit 1 ;;
esac
[ "$keep" -ge 1 ] || { echo "BACKUP_KEEP must be at least 1" >&2; exit 1; }

export PGPASSWORD="$POSTGRES_PASSWORD"

backup() {
  ts=$(date -u +%Y%m%dT%H%M%SZ)
  tmp="$dir/.gophkeeper-$ts.dump.tmp"
  # Сначала во временный файл: оборванная копия не должна выглядеть готовой
  # и вытеснять старые при ротации.
  if pg_dump -h db -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc -f "$tmp"; then
    mv "$tmp" "$dir/gophkeeper-$ts.dump"
    echo "backup: gophkeeper-$ts.dump ($(wc -c < "$dir/gophkeeper-$ts.dump") bytes)"
  else
    rm -f "$tmp"
    echo "backup: pg_dump failed" >&2
    return 1
  fi

  # Ротация: оставить keep самых новых (имена сортируются по времени).
  ls -1 "$dir"/gophkeeper-*.dump 2>/dev/null | sort -r | tail -n +"$((keep + 1))" | while read -r old; do
    rm -f "$old" && echo "backup: removed $(basename "$old")"
  done
}

echo "backup: every $interval, keeping $keep copies in $dir"
while true; do
  backup || true
  sleep "$interval"
done
