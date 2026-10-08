# GophKeeper

Клиент-серверный менеджер паролей и приватных данных. Сервер работает в Docker, клиент — кроссплатформенный CLI-бинарь.

## Возможности

- Регистрация и аутентификация пользователей (JWT).
- Хранение секретов четырёх типов: `credentials`, `text`, `card`, `binary`.
- Клиентское шифрование: AES-256-GCM с ключом, производным от пароля (Argon2id + HKDF); сервер видит только шифротекст.
- Разделение ключей: на сервер уходит только ключ аутентификации, из которого нельзя получить ключ шифрования (см. [Схема ключей](#схема-ключей)).
- TLS между клиентом и сервером.
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
| `TOKEN_TTL`    | Время жизни JWT (по умолчанию `1h`)              | `30m`, `1h`                                                 |
| `TLS_CERT_FILE`| Сертификат сервера (PEM), путь внутри контейнера | `/certs/server.crt`                                         |
| `TLS_KEY_FILE` | Приватный ключ сервера (PEM)                     | `/certs/server.key`                                         |

| Переменная         | Описание                                                        | По умолчанию |
| ------------------ | --------------------------------------------------------------- | ------------ |
| `AUTH_RATE_LIMIT`  | Запросов к `GetSalt`/`CreateUser`/`Login` в минуту с одного IP  | `30`         |
| `AUTH_RATE_BURST`  | Таких запросов подряд с одного IP                               | `10`         |
| `SECRET_MAX_SIZE`  | Максимальный размер зашифрованного секрета, байт                | `1048576`    |
| `SECRET_MAX_COUNT` | Максимум секретов у одного пользователя                         | `1000`       |

Лимит частоты замедляет онлайн-перебор паролей и массовую регистрацию, квоты не дают заполнить диск сервера. Каждая операция с секретами проверяет мастер-пароль и делает 2 запроса к `Auth`, поэтому при значениях по умолчанию с одного IP можно выполнить 5 операций подряд и около 15 в минуту. Бинарные секреты хранятся в base64 внутри JSON, поэтому при `SECRET_MAX_SIZE` = 1 MiB максимальный размер файла около 750 КБ.

`TLS_CERT_FILE` и `TLS_KEY_FILE` задаются вместе. Если оба пусты, сервер работает без TLS и пишет предупреждение в лог. Так можно делать только при локальной разработке.

Пример — см. [`.env_example`](.env_example).

### Клиент

| Переменная             | Описание                                     | По умолчанию            |
| ---------------------- | -------------------------------------------- | ----------------------- |
| `GOPHKEEPER_SERVER`    | Адрес gRPC-сервера                           | `localhost:50051`       |
| `GOPHKEEPER_SESSION`   | Путь к файлу сессии (хранит логин и токен)   | `~/.gophkeeper/session` |
| `GOPHKEEPER_TLS_CERT`  | Путь к CA-сертификату для TLS (опционально)  | —                       |
| `GOPHKEEPER_INSECURE`  | `true` — подключаться без TLS                | `false`                 |

Клиент всегда подключается по TLS. Если задан `GOPHKEEPER_TLS_CERT`, сертификат сервера проверяется этим CA, иначе системными корневыми сертификатами. Без TLS клиент подключается только при `GOPHKEEPER_INSECURE=true` и в этом случае выводит предупреждение. Такой режим подходит только для локальной разработки: ключ аутентификации и токен идут открытым текстом. Одновременно задать `GOPHKEEPER_INSECURE` и `GOPHKEEPER_TLS_CERT` нельзя.

## Запуск сервера

```bash
cp .env_example .env  # заполнить значения
make docker-up        # поднимает postgres, применяет миграции, запускает сервер
make docker-down
```

## TLS и удалённый доступ

Если сервер доступен по сети, включите TLS. Сертификаты выпускает собственный CA проекта:

```bash
# перечислите все имена и IP, по которым клиенты будут обращаться к серверу
make certs CERT_HOSTS=localhost,127.0.0.1,203.0.113.10,keeper.example.com
```

В `certs/` появятся:

- `ca.crt` — корневой сертификат. Его нужно передать на клиентские машины.
- `ca.key` — ключ CA. Он остаётся на сервере и нужен только для перевыпуска сертификата.
- `server.crt`, `server.key` — монтируются в контейнер (`./certs:/certs:ro`).

Повторный `make certs` перевыпускает сертификат сервера тем же CA, поэтому `ca.crt` у клиентов менять не придётся.

На удалённой машине:

```bash
export GOPHKEEPER_SERVER=203.0.113.10:50051     # адрес должен входить в CERT_HOSTS
export GOPHKEEPER_TLS_CERT=/path/to/ca.crt
gophkeeper auth login alice
gophkeeper secret get --name gmail --type credentials
```

Соль хранится на сервере, ключи выводятся из пароля, поэтому секреты, созданные на одной машине, читаются на любой другой под той же учёткой.

## Сессии и отзыв токенов

После `auth login` клиент хранит JWT в файле сессии. Каждая операция с секретами проверяет мастер-пароль на сервере и получает новый токен, поэтому срок жизни токена можно держать коротким (`TOKEN_TTL`, по умолчанию 1 час): если файл сессии украдут, токен скоро истечёт сам.

Токены можно отозвать раньше срока:

- `auth logout` отзывает токен текущего устройства (его `jti` попадает в таблицу `revoked_tokens`);
- `auth logout --all` отзывает все токены пользователя (увеличивается `users.token_version`). Используйте эту команду, если устройство потеряно или файл сессии мог утечь.

Сервер проверяет отзыв при каждом запросе.

## Схема ключей

```
masterKey = Argon2id(пароль, соль)          # только на клиенте
authKey   = HKDF(masterKey, "auth")         # отправляется на сервер при register/login
encKey    = HKDF(masterKey, "encryption")   # AES-256-GCM, только на клиенте
hmacKey   = HKDF(masterKey, "blind-index")  # blind index, только на клиенте
```

Сервер хранит `SHA-256(authKey)`. Из `authKey` нельзя восстановить ни `masterKey`, ни `encKey`. Поэтому ни сервер, ни утёкшая БД, ни перехваченный `authKey` не позволяют расшифровать секреты.

Миграция `000002_auth_key_hash` переводит существующих пользователей на эту схему, пароли менять не нужно. Миграция необратима: её `down` ничего не делает, и старая версия сервера после неё не сможет аутентифицировать пользователей.

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
gophkeeper auth logout          # отозвать токен этого устройства
gophkeeper auth logout --all    # отозвать токены на всех устройствах

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
