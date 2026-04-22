# GophKeeper

Клиент-серверный менеджер паролей и приватных данных. Сервер работает в Docker, клиент — кроссплатформенный CLI-бинарь.

## Возможности

- Регистрация и аутентификация пользователей (JWT).
- Хранение секретов четырёх типов: `credentials`, `text`, `card`, `binary`.
- Клиентское шифрование: AES-256-GCM с ключом, производным от пароля (Argon2id + HKDF); сервер видит только шифротекст.
- Поиск по имени через blind index (HMAC-SHA256).
- CRUD-операции над секретами, листинг с фильтрацией по типу.

## Структура

- `cmd/client`, `cmd/server` — точки входа.
- `internal/client` — CLI (Cobra), gRPC-клиент, сервисный слой, сессия.
- `internal/server` — gRPC-хендлеры, сервисы, репозитории (pgx), JWT, логгер.
- `internal/domain` — модели и доменные ошибки.
- `pkg/crypto` — примитивы шифрования.
- `proto/` — схемы gRPC, `proto/gen` — сгенерированный код.
- `migrations/` — SQL-миграции (golang-migrate).
- `tests/e2e` — интеграционные тесты на testcontainers-go.

## Переменные окружения

### Сервер (`.env`)

| Переменная     | Описание                                         | Пример                                                      |
| -------------- | ------------------------------------------------ | ----------------------------------------------------------- |
| `DATABASE_URL` | DSN PostgreSQL                                   | `postgresql://gophkeeper:secret@db/gophkeeper?sslmode=disable` |
| `JWT_SECRET`   | Секрет для подписи JWT (≥ 32 символа)            | `your-super-secret-key-min-32-chars`                        |
| `SERVER_PORT`  | Порт gRPC-сервера (по умолчанию `50051`)         | `50051`                                                     |
| `LOG_LEVEL`    | `development` или `production`                   | `development`                                               |
| `TOKEN_TTL`    | Время жизни JWT (по умолчанию `24h`)             | `1h`, `30m`, `24h`                                          |

Пример — см. [`.env_example`](.env_example).

### Клиент

| Переменная             | Описание                                     | По умолчанию            |
| ---------------------- | -------------------------------------------- | ----------------------- |
| `GOPHKEEPER_SERVER`    | Адрес gRPC-сервера                           | `localhost:50051`       |
| `GOPHKEEPER_SESSION`   | Путь к файлу сессии (хранит логин и токен)   | `~/.gophkeeper/session` |
| `GOPHKEEPER_TLS_CERT`  | Путь к CA-сертификату для TLS (опционально)  | —                       |

Если `GOPHKEEPER_TLS_CERT` не задан, клиент подключается без TLS.

## Запуск сервера

```bash
cp .env_example .env  # заполнить значения
make docker-up        # поднимает postgres, применяет миграции, запускает сервер
make docker-down
```

## Сборка клиента

```bash
make build-client       # для текущей ОС → bin/gophkeeper
make build-client-all   # linux/darwin/windows → bin/gophkeeper-<os>-<arch>
./bin/gophkeeper version
```

В бинарь инжектятся `Version` (из `git describe`) и `BuildDate`.

## Автодополнение (bash)

```bash
make install-completion
source ~/.bashrc
```

## Использование

```bash
# регистрация и вход
gophkeeper auth register alice
gophkeeper auth login alice
gophkeeper auth logout

# создание секретов
gophkeeper secret create --name gmail --type credentials
gophkeeper secret create --name passport --type text
gophkeeper secret create --name visa --type card
gophkeeper secret create --name keyfile --type binary

# чтение
gophkeeper secret get --name gmail --type credentials
gophkeeper secret get --name gmail --type credentials --json
gophkeeper secret get --name keyfile --type binary --output ./keyfile.bin

# обновление и удаление
gophkeeper secret update --name gmail --type credentials
gophkeeper secret delete --name gmail --type credentials --yes

# список
gophkeeper secret list
gophkeeper secret list --type card
```

При входе в систему и операциях с секретами клиент интерактивно запрашивает мастер-пароль.

## Разработка

```bash
make generate     # перегенерировать gRPC-код из proto/
make docs         # сгенерировать docs/api.md из .proto (нужен protoc-gen-doc)
make test         # unit-тесты
make test-e2e     # интеграционные тесты (требует Docker)
make test-cover   # покрытие (без mocks/, proto/gen/, cmd/)
make clean        # удалить bin/ и coverage*.out
```

Документация gRPC API собирается из комментариев в `proto/*.proto` и лежит в [`docs/api.md`](docs/api.md).
Установка генератора:

```bash
go install github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@latest
```

Для запуска миграции вниз на один шаг:

```bash
docker compose --profile tools run --rm migrate-down
```
