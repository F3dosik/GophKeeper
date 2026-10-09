# Разработка

## Структура

| Каталог | Содержимое |
| --- | --- |
| `cmd/server`, `cmd/client` | Точки входа сервера и CLI |
| `internal/server` | gRPC-хендлеры, сервисы, репозитории (pgx), JWT, middleware (аутентификация, лимиты) |
| `internal/client/command` | Команды CLI (Cobra) |
| `internal/client/service` | Регистрация, вход, операции с секретами, смена пароля |
| `internal/client/vault` | Разблокированное хранилище для приложения: ключи в памяти, автоблокировка |
| `internal/client/grpcclient`, `session`, `config` | Подключение к серверу, файл сессии, настройки клиента |
| `internal/client/export` | Зашифрованный экспорт и импорт |
| `internal/client/passgen` | Генератор паролей и парольных фраз |
| `internal/domain` | Модели и доменные ошибки |
| `pkg/crypto` | Argon2id, HKDF, AES-GCM, blind index |
| `desktop/` | Десктоп-приложение: `backend` (Go), `frontend` (Svelte 5 + TypeScript) |
| `proto/` | Схемы gRPC, `proto/gen` — сгенерированный код |
| `migrations/` | SQL-миграции (golang-migrate) |
| `scripts/backup.sh` | Резервное копирование по расписанию |
| `tests/e2e` | Интеграционные тесты на testcontainers-go |
| `.github/workflows` | CI, релизы, еженедельный `govulncheck` |

## Сборка

CLI:

```bash
make build-client       # для текущей ОС → bin/gophkeeper
make build-client-all   # linux, darwin, windows → bin/gophkeeper-<ос>-<архитектура>
./bin/gophkeeper version
```

Версия (`git describe`) и дата сборки встраиваются в бинарь.

Десктоп-приложение собирается [Wails](https://wails.io). Нужны Node.js с npm и Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev    # Linux

make desktop-dev             # режим разработки с горячей перезагрузкой
make desktop-build           # для текущей ОС → desktop/build/bin/gophkeeper-desktop
make desktop-build-windows   # .exe для Windows, собирается на Linux
```

На Ubuntu 24.04+ есть только WebKitGTK 4.1, поэтому сборка идёт с тегом `webkit2_41`, а `wails doctor` сообщает об отсутствии `libwebkit` — это ожидаемо. Приложение для macOS собирается только на macOS.

Фронтенд встраивается в бинарь только с тегами `desktop` или `dev`, поэтому `go build ./...` и `go test ./...` работают без сборки фронтенда.

Переменные для отладки приложения: `GOPHKEEPER_DESKTOP_DIR` — каталог данных (настройки, `ca.crt`, сессия), `GOPHKEEPER_DESKTOP_HIDDEN=1` — запуск без окна (для автотестов интерфейса).

## Тесты

```bash
make test         # unit-тесты
make test-e2e     # интеграционные: сервер и PostgreSQL в testcontainers, нужен Docker
make test-cover   # покрытие без mocks/, proto/gen/, cmd/

cd desktop/frontend && npm run check    # проверка типов Svelte и TypeScript
```

Образ PostgreSQL для e2e задаётся `E2E_POSTGRES_IMAGE` (по умолчанию `postgres:18-alpine`) — пригодится, если Docker Hub ограничивает скачивание.

## Генерация кода

```bash
make generate     # gRPC-код из proto/
make docs         # docs/api.md из комментариев в proto/*.proto
make clean        # удалить bin/ и файлы покрытия
```

Для `make docs` нужен `protoc-gen-doc`:

```bash
go install github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@latest
```

Новая миграция добавляется парой файлов `migrations/00000N_имя.{up,down}.sql` и применяется при запуске стека. Чтобы проверить изменения сервера в Docker до слияния, соберите образ из исходников: `make docker-up-local`. Откат на один шаг — `docker compose --profile tools run --rm migrate-down`.

## CI и релизы

`.github/workflows/ci.yml` запускается на каждый PR и push в `main`:

- Go: `go vet`, unit-тесты с `-race`, e2e-тесты, `govulncheck`;
- фронтенд: `svelte-check --fail-on-warnings` и сборка;
- сборки CLI для Linux, macOS и Windows и приложения на трёх ОС — результаты лежат в артефактах запуска;
- образ сервера для amd64 и arm64. В PR он только собирается, после push в `main` и по тегу публикуется в GHCR (теги — в [docs/deployment.md](deployment.md#образ-сервера)).

`govulncheck.yml` раз в неделю проверяет `main`: уязвимости в зависимостях публикуются постоянно, и код может стать уязвимым без единого коммита.

Релиз выпускается по тегу. CI собирает всё заново, публикует образ сервера с тегами версии и `latest` и GitHub Release с бинарями CLI, архивами приложения и `SHA256SUMS`:

```bash
git tag v1.1.0 && git push origin v1.1.0
```
