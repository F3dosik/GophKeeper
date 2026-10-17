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
| `mobile/` | Клиент для Android на Go, собирается gomobile в `shron.aar` |
| `android/` | Android-приложение (Kotlin, Jetpack Compose) поверх `shron.aar` |
| `proto/` | Схемы gRPC, `proto/gen` — сгенерированный код |
| `migrations/` | SQL-миграции (golang-migrate) |
| `scripts/backup.sh` | Резервное копирование по расписанию |
| `tests/e2e` | Интеграционные тесты на testcontainers-go |
| `.github/workflows` | CI, релизы, еженедельный `govulncheck` |

## Сборка

CLI:

```bash
make build-client       # для текущей ОС → bin/shron
make build-client-all   # linux, darwin, windows → bin/shron-<ос>-<архитектура>
./bin/shron version
```

Версия (`git describe`) и дата сборки встраиваются в бинарь.

Десктоп-приложение собирается [Wails](https://wails.io). Нужны Node.js с npm и Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev    # Linux

make desktop-dev             # режим разработки с горячей перезагрузкой
make desktop-build           # для текущей ОС → desktop/build/bin/shron-desktop
make desktop-build-windows   # .exe для Windows, собирается на Linux
```

На Ubuntu 24.04+ есть только WebKitGTK 4.1, поэтому сборка идёт с тегом `webkit2_41`, а `wails doctor` сообщает об отсутствии `libwebkit` — это ожидаемо. Приложение для macOS собирается только на macOS.

Фронтенд встраивается в бинарь только с тегами `desktop` или `dev`, поэтому `go build ./...` и `go test ./...` работают без сборки фронтенда.

Переменные для отладки приложения: `SHRON_DESKTOP_DIR` — каталог данных (настройки, `ca.crt`, сессия), `SHRON_DESKTOP_HIDDEN=1` — запуск без окна (для автотестов интерфейса).

### Android

Нужны Android SDK с NDK и JDK 25 — проще всего поставить Android Studio, в ней есть и SDK Manager, и JDK. Сначала собирается Go-ядро, затем приложение:

```bash
export ANDROID_HOME=~/Android/Sdk                 # NDK ищется в $ANDROID_HOME/ndk
export JAVA_HOME=/snap/android-studio/current/jbr # JDK из Android Studio; путь зависит от установки
make android-aar                                  # → android/app/libs/shron.aar (arm64 и x86_64)

cd android
./gradlew assembleDebug                           # → app/build/outputs/apk/debug/app-debug.apk
./gradlew installDebug                            # на подключённое устройство или эмулятор
```

`shron.aar` в репозиторий не попадает: после изменений в `mobile/` и после `git clone` его нужно пересобрать. В Android Studio открывается каталог `android/`, а не корень репозитория.

`./gradlew assembleDebug -Pshron.insecure=true` собирает приложение без TLS — только для локальной разработки. Тексты интерфейса лежат в `android/app/src/main/res/values/strings.xml`.

## Тесты

```bash
make test         # unit-тесты
make test-e2e     # интеграционные: сервер и PostgreSQL в testcontainers, нужен Docker
make test-cover   # покрытие без mocks/, proto/gen/, cmd/

cd desktop/frontend && npm run check    # проверка типов Svelte и TypeScript

cd android
./gradlew testDebugUnitTest             # JVM-тесты Kotlin
./gradlew connectedDebugAndroidTest     # на эмуляторе или устройстве, с настоящим Go-ядром

./gradlew assembleRelease -Pshron.debugSignedRelease=true
python3 scripts/release_smoke.py app/build/outputs/apk/release/app-release.apk   # release с R8
```

Тесты на эмуляторе удаляют приложение с устройства вместе с его данными, smoke-тест тоже.

Инструментальные тесты идут на debug-сборке, где R8 не работает, поэтому ошибки оптимизации ловит smoke-тест release-сборки: он проходит через интерфейс сценарий, задевающий Go-ядро (классы gomobile вызываются из нативного кода по именам, и R8 не должен их трогать — правила в `android/app/src/main/keepRules`). `-Pshron.debugSignedRelease=true` подписывает release отладочным ключом — только для такой проверки.

Инструментальным тестам Android сервер не нужен. Каждый тест идёт в новом процессе с пустыми данными приложения (Test Orchestrator): Go-ядро одно на процесс и хранит настройки и сессию в `filesDir`.

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
- Android: `shron.aar`, сборка, lint и JVM-тесты; затем на эмуляторе инструментальные тесты и smoke-тест release-сборки с R8. Отладочный APK лежит в артефактах запуска;
- сборки CLI для Linux, macOS и Windows и приложения на трёх ОС — результаты лежат в артефактах запуска;
- образ сервера для amd64 и arm64. В PR он только собирается, после push в `main` и по тегу публикуется в GHCR (теги — в [docs/deployment.md](deployment.md#образ-сервера)).

`govulncheck.yml` раз в неделю проверяет `main`: уязвимости в зависимостях публикуются постоянно, и код может стать уязвимым без единого коммита.

Релиз выпускается по тегу. CI собирает всё заново, публикует образ сервера с тегами версии и `latest` и GitHub Release с бинарями CLI, архивами приложения, APK для Android и `SHA256SUMS`:

```bash
git tag v1.1.0 && git push origin v1.1.0
```

APK подписывается ключом релизов из секретов репозитория: `ANDROID_KEYSTORE_BASE64` — хранилище ключей PKCS12 в base64, `ANDROID_KEYSTORE_PASSWORD` — его пароль (псевдоним ключа — `shron`). Версия берётся из тега: `v1.3.0` → `versionName` 1.3.0, `versionCode` 10300. Android ставит обновление поверх установленного приложения, только если оно подписано тем же ключом, поэтому ключ один на все релизы и хранится в резервной копии вне репозитория: без него следующую версию придётся ставить с удалением предыдущей.

Подписанный APK можно собрать и локально:

```bash
export SHRON_KEYSTORE=~/.android/shron-release.jks SHRON_KEYSTORE_PASSWORD=...
./gradlew assembleRelease -Pshron.versionName=1.3.0 -Pshron.versionCode=10300
```
