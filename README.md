# GophKeeper

[![CI](https://github.com/F3dosik/GophKeeper/actions/workflows/ci.yml/badge.svg)](https://github.com/F3dosik/GophKeeper/actions/workflows/ci.yml)

Менеджер паролей и приватных данных с собственным сервером. Секреты шифруются на устройстве пользователя: сервер и его база видят только шифротекст, а мастер-пароль никогда не покидает клиент.

## Возможности

- **Четыре типа секретов:** логины и пароли, заметки, банковские карты, файлы.
- **Клиентское шифрование:** AES-256-GCM, ключи выводятся из мастер-пароля через Argon2id и HKDF. На сервер уходит только ключ аутентификации, из которого нельзя получить ключ шифрования.
- **Клиенты:** CLI для Linux, macOS и Windows и десктоп-приложение (Wails). Работают с одним сервером и одними учётками, секреты доступны на всех устройствах.
- **Генератор паролей и парольных фраз** (словарь EFF).
- **Смена мастер-пароля** с перешифровкой всех секретов в одной транзакции.
- **Сессии:** короткоживущие токены, выход на одном или на всех устройствах.
- **Экспорт хранилища** в зашифрованный файл и импорт в любую учётку.
- **Для администратора:** закрытая регистрация, учётки с временным паролем, список пользователей, завершение сессий, удаление; резервные копии базы по расписанию.
- **TLS** между клиентом и сервером, лимиты частоты запросов и квоты на хранение.

Подробности о защите — в [docs/security.md](docs/security.md).

## Быстрый старт

### Сервер

Нужны Docker с Compose и OpenSSL.

```bash
git clone https://github.com/F3dosik/GophKeeper.git && cd GophKeeper
cp .env_example .env        # заполнить: пароль БД, DATABASE_URL, JWT_SECRET (openssl rand -hex 32)

# сертификат для всех адресов, по которым клиенты будут подключаться
make certs CERT_HOSTS=localhost,127.0.0.1,192.168.1.5

make docker-up              # PostgreSQL, миграции, сервер, резервное копирование
```

Сервер слушает порт `50051`. Настройка, регистрация пользователей, резервные копии и обновление — в [docs/deployment.md](docs/deployment.md).

### Клиент

Готовые сборки — в [Releases](https://github.com/F3dosik/GophKeeper/releases): CLI `gophkeeper-<ос>-<архитектура>` и архивы приложения `gophkeeper-desktop-*`. Контрольные суммы — в `SHA256SUMS`. Собрать самому: `make build-client` и `make desktop-build`.

На каждое устройство нужен файл `certs/ca.crt` с сервера.

```bash
export GOPHKEEPER_SERVER=192.168.1.5          # порт по умолчанию 50051
export GOPHKEEPER_TLS_CERT=/path/to/ca.crt

gophkeeper auth register alice
gophkeeper auth login alice
gophkeeper secret create --name github --type credentials --generate
gophkeeper secret get --name github --type credentials
```

В приложении адрес сервера и `ca.crt` указываются при первом запуске.

Бинари не подписаны. Windows покажет SmartScreen («Подробнее» → «Выполнить в любом случае»), macOS попросит открыть приложение через контекстное меню. Linux-приложению нужен WebKitGTK 4.1 (`libwebkit2gtk-4.1-0`).

## CLI

Мастер-пароль запрашивается интерактивно и проверяется на сервере перед каждой операцией с секретами.

```bash
# учётка
gophkeeper auth register alice
gophkeeper auth login alice
gophkeeper auth passwd                  # сменить мастер-пароль (все секреты перешифровываются)
gophkeeper auth logout                  # выйти на этом устройстве
gophkeeper auth logout --all            # выйти на всех устройствах (например, при потере телефона)
gophkeeper auth delete-account          # удалить учётку со всеми секретами

# секреты: --type credentials | text | card | binary
gophkeeper secret create --name gmail --type credentials
gophkeeper secret create --name gmail --type credentials --generate   # сгенерировать пароль
gophkeeper secret get    --name gmail --type credentials [--json]
gophkeeper secret get    --name key   --type binary --output ./key.bin
gophkeeper secret update --name gmail --type credentials
gophkeeper secret delete --name gmail --type credentials
gophkeeper secret list [--type card]

# генератор (на клиенте, никуда не отправляется)
gophkeeper generate                     # 20 символов
gophkeeper generate --length 32 --no-symbols
gophkeeper generate --words 6           # парольная фраза — удобна для мастер-пароля

# экспорт и импорт (файл зашифрован отдельным паролем экспорта)
gophkeeper export -o vault.gkx
gophkeeper import vault.gkx [--overwrite]
```

| Переменная | Описание | По умолчанию |
| --- | --- | --- |
| `GOPHKEEPER_SERVER` | Адрес сервера, порт необязателен | `localhost:50051` |
| `GOPHKEEPER_TLS_CERT` | CA-сертификат сервера (`ca.crt`) | системные сертификаты |
| `GOPHKEEPER_SESSION` | Файл сессии (логин и токен) | `~/.gophkeeper/session` |
| `GOPHKEEPER_INSECURE` | `true` — без TLS, только для локальной разработки | `false` |

Автодополнение для bash: `make install-completion`; для других оболочек — `gophkeeper completion --help`.

## Десктоп-приложение

То же, что CLI, с графическим интерфейсом: список с поиском и фильтром по типу, просмотр и копирование, формы для всех типов секретов, генератор паролей, смена пароля, экспорт и импорт, выход на всех устройствах.

- Мастер-пароль вводится один раз. Хранилище блокируется вручную, при сворачивании окна, после бездействия (по умолчанию 5 минут) и при отзыве сессии с другого устройства.
- Пароли и CVV скрыты, пока их не показали. Скопированное значение стирается из буфера обмена через 30 секунд.
- Подключённое к административному порту, приложение открывает панель администратора ([docs/deployment.md](docs/deployment.md#регистрация-и-администрирование)).

Данные приложения хранятся в `~/.config/GophKeeper`, `%AppData%\GophKeeper` или `~/Library/Application Support/GophKeeper`; другой каталог задаёт `GOPHKEEPER_DESKTOP_DIR`.

## Документация

- [docs/deployment.md](docs/deployment.md) — установка и настройка сервера, TLS, регистрация и администрирование, резервные копии, обновление.
- [docs/security.md](docs/security.md) — схема ключей, модель угроз, сессии, известные ограничения.
- [docs/development.md](docs/development.md) — структура кода, сборка, тесты, CI и релизы.
- [docs/api.md](docs/api.md) — gRPC API (генерируется из `proto/`).
