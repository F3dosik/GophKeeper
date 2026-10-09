.PHONY: help generate docs build-client build-client-all build-server test test-e2e test-cover docker-up docker-down certs ca-encrypt desktop-dev desktop-build desktop-build-windows clean

# Версия и дата сборки — подставляются в бинарь клиента через -ldflags.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/F3dosik/GophKeeper/internal/client/command.Version=$(VERSION) \
           -X github.com/F3dosik/GophKeeper/internal/client/command.BuildDate=$(BUILD_DATE)

SHELL := /bin/bash

BIN_DIR := bin

# Десктоп-клиент (Wails). WAILS — путь к CLI: go install github.com/wailsapp/wails/v2/cmd/wails@latest
# На Ubuntu 24.04+ есть только WebKitGTK 4.1, для него нужен тег webkit2_41.
WAILS       ?= wails
WAILS_TAGS  ?= webkit2_41
DESKTOP_DIR := desktop

COMPLETION_FILE := $(HOME)/.gophkeeper_completion

# TLS: каталог с сертификатами и список имён/IP, по которым клиенты обращаются к серверу.
# Пример: make certs CERT_HOSTS=localhost,127.0.0.1,203.0.113.10,keeper.example.com
CERTS_DIR  := certs
CA_DIR     := ca
CERT_HOSTS ?= localhost,127.0.0.1
CERT_DAYS  ?= 825

help:
	@echo "Доступные цели:"
	@echo "  generate          — кодогенерация из proto-файлов"
	@echo "  docs              — сгенерировать документацию API (docs/api.md)"
	@echo "  build-client      — бинарь клиента для текущей ОС"
	@echo "  build-client-all  — бинарники клиента для linux/macOS/windows"
	@echo "  build-server      — бинарь сервера (обычно запускается через docker)"
	@echo "  test              — unit-тесты"
	@echo "  test-e2e          — e2e-тесты (требует Docker)"
	@echo "  test-cover        — покрытие тестами"
	@echo "  docker-up         — поднять сервер в docker-compose"
	@echo "  docker-down       — остановить docker-compose"
	@echo "  certs             — сгенерировать CA и TLS-сертификат сервера (CERT_HOSTS=...)"
	@echo "  ca-encrypt        — зашифровать паролем существующий ключ CA"
	@echo "  desktop-dev       — десктоп-клиент в режиме разработки (горячая перезагрузка)"
	@echo "  desktop-build     — десктоп-клиент для текущей ОС → desktop/build/bin"
	@echo "  desktop-build-windows — десктоп-клиент для Windows (кросс-сборка)"
	@echo "  clean             — удалить bin/"

# Кодогенерация из .proto файлов
generate:
	mkdir -p proto/gen
	protoc \
		--go_out=proto/gen \
		--go_opt=module=github.com/F3dosik/GophKeeper/proto/gen \
		--go_opt=default_api_level=API_OPAQUE \
		--go-grpc_out=proto/gen \
		--go-grpc_opt=module=github.com/F3dosik/GophKeeper/proto/gen \
		proto/*.proto

# Документация API из .proto (требует protoc-gen-doc в PATH).
# Установка: go install github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@latest
# protoc-gen-doc пока не поддерживает editions 2023, поэтому временно конвертируем
# .proto в syntax=proto3 только для генерации документации.
docs:
	mkdir -p docs
	rm -rf .tmp-proto
	mkdir -p .tmp-proto
	for f in proto/*.proto; do \
	    sed 's/^edition = "2023";/syntax = "proto3";/' "$$f" > ".tmp-proto/$$(basename $$f)"; \
	done
	protoc --proto_path=.tmp-proto --doc_out=docs --doc_opt=markdown,api.md .tmp-proto/*.proto
	rm -rf .tmp-proto

build-client:
	mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/gophkeeper ./cmd/client

build-client-all:
	mkdir -p $(BIN_DIR)
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/gophkeeper-linux-amd64       ./cmd/client
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/gophkeeper-darwin-amd64      ./cmd/client
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/gophkeeper-darwin-arm64      ./cmd/client
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/gophkeeper-windows-amd64.exe ./cmd/client

install-completion:
	@$(BIN_DIR)/gophkeeper completion bash > $(COMPLETION_FILE)
	@echo "complete -F __start_gophkeeper $(BIN_DIR)/gophkeeper" >> $(COMPLETION_FILE)
	@grep -qxF "source $(COMPLETION_FILE)" $(HOME)/.bashrc || \
		echo "source $(COMPLETION_FILE)" >> $(HOME)/.bashrc
	@echo "Автодополнение установлено. Выполни: source ~/.bashrc"

build-server:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/gophkeeper-server ./cmd/server

test:
	go test ./...

test-e2e:
	go test -tags=e2e -v ./tests/e2e/...

test-cover:
	go test -coverpkg=./... -coverprofile=coverage.out ./...
	@grep -v -E "(mocks/|proto/gen/|cmd/)" coverage.out > coverage.filtered.out
	@head -1 coverage.out > coverage.final.out && tail -n +2 coverage.filtered.out >> coverage.final.out
	go tool cover -func=coverage.final.out | tail -1

# Генерирует собственный CA и подписанный им сертификат сервера.
# certs/ca.crt раздаётся клиентам (GOPHKEEPER_TLS_CERT), certs/server.* монтируются в контейнер.
# Ключ CA лежит отдельно в ca/ (не монтируется в контейнер) и зашифрован паролем:
# с ним можно выпустить сертификат, которому поверят все клиенты, поэтому после выпуска
# его лучше убрать с сервера (см. README) и возвращать только для перевыпуска.
# Пароль ключа CA спрашивается интерактивно или берётся из переменной CA_PASS.
# server.key получает права 0644, т.к. сервер в контейнере работает под отдельным UID;
# каталоги certs/ и ca/ не коммитятся (см. .gitignore).
CA_PASS_OUT = $(if $(CA_PASS),-passout env:CA_PASS)
CA_PASS_IN  = $(if $(CA_PASS),-passin env:CA_PASS)

certs:
	@if [ -f $(CERTS_DIR)/ca.key ]; then \
		echo "Ключ CA лежит в $(CERTS_DIR)/, а этот каталог монтируется в контейнер."; \
		echo "Перенесите его: mkdir -p $(CA_DIR) && mv $(CERTS_DIR)/ca.key $(CA_DIR)/ && make ca-encrypt"; \
		exit 1; \
	fi
	@mkdir -p $(CERTS_DIR) $(CA_DIR) && chmod 700 $(CA_DIR)
	@test -f $(CA_DIR)/ca.key || { \
		echo "Создаётся CA. Задайте пароль ключа CA, он понадобится при перевыпуске сертификата."; \
		openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 $(CA_PASS_OUT) \
			-keyout $(CA_DIR)/ca.key -out $(CERTS_DIR)/ca.crt -days $(CERT_DAYS) \
			-subj "/CN=GophKeeper CA" 2>/dev/null; }
	@SAN=$$(echo "$(CERT_HOSTS)" | tr ',' '\n' | while read h; do \
		if echo "$$h" | grep -Eq '^[0-9.]+$$|:'; then echo "IP:$$h"; else echo "DNS:$$h"; fi; \
	done | paste -sd, -); \
	openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes \
		-keyout $(CERTS_DIR)/server.key.new -out $(CERTS_DIR)/server.csr \
		-subj "/CN=gophkeeper-server" 2>/dev/null && \
	echo "Подпись сертификата сервера ключом CA:" && \
	openssl x509 -req -in $(CERTS_DIR)/server.csr -CA $(CERTS_DIR)/ca.crt -CAkey $(CA_DIR)/ca.key $(CA_PASS_IN) \
		-CAcreateserial -CAserial $(CA_DIR)/ca.srl -out $(CERTS_DIR)/server.crt.new -days $(CERT_DAYS) \
		-extfile <(printf "subjectAltName=$$SAN\nextendedKeyUsage=serverAuth") 2>/dev/null; \
	status=$$?; rm -f $(CERTS_DIR)/server.csr $(CA_DIR)/ca.srl; \
	if [ $$status -ne 0 ]; then \
		rm -f $(CERTS_DIR)/server.key.new $(CERTS_DIR)/server.crt.new; \
		echo "Не удалось подписать сертификат (неверный пароль CA?). Текущий сертификат не изменён."; \
		exit 1; \
	fi; \
	mv $(CERTS_DIR)/server.key.new $(CERTS_DIR)/server.key && \
	mv $(CERTS_DIR)/server.crt.new $(CERTS_DIR)/server.crt && \
	chmod 600 $(CA_DIR)/ca.key && chmod 644 $(CERTS_DIR)/server.key && \
	echo "Сертификат сервера выпущен для: $$SAN" && \
	echo "Клиентам передать $(CERTS_DIR)/ca.crt и задать GOPHKEEPER_TLS_CERT=<путь к ca.crt>" && \
	echo "Ключ CA ($(CA_DIR)/ca.key) после выпуска лучше убрать с сервера."

# Шифрует паролем ранее созданный незашифрованный ключ CA.
ca-encrypt:
	@if grep -q ENCRYPTED $(CA_DIR)/ca.key; then echo "$(CA_DIR)/ca.key уже зашифрован"; exit 0; fi; \
	openssl pkey -in $(CA_DIR)/ca.key -aes256 $(CA_PASS_OUT) -out $(CA_DIR)/ca.key.enc && \
	mv $(CA_DIR)/ca.key.enc $(CA_DIR)/ca.key && chmod 600 $(CA_DIR)/ca.key && \
	echo "$(CA_DIR)/ca.key зашифрован"

desktop-dev:
	cd $(DESKTOP_DIR) && $(WAILS) dev -tags $(WAILS_TAGS)

desktop-build:
	cd $(DESKTOP_DIR) && $(WAILS) build -tags $(WAILS_TAGS) -clean

desktop-build-windows:
	cd $(DESKTOP_DIR) && $(WAILS) build -platform windows/amd64

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

clean:
	rm -rf $(BIN_DIR) coverage*.out .tmp-proto
