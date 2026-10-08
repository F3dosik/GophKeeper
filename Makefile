.PHONY: help generate docs build-client build-client-all build-server test test-e2e test-cover docker-up docker-down certs clean

# Версия и дата сборки — подставляются в бинарь клиента через -ldflags.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/F3dosik/GophKeeper/internal/client/command.Version=$(VERSION) \
           -X github.com/F3dosik/GophKeeper/internal/client/command.BuildDate=$(BUILD_DATE)

SHELL := /bin/bash

BIN_DIR := bin

COMPLETION_FILE := $(HOME)/.gophkeeper_completion

# TLS: каталог с сертификатами и список имён/IP, по которым клиенты обращаются к серверу.
# Пример: make certs CERT_HOSTS=localhost,127.0.0.1,203.0.113.10,keeper.example.com
CERTS_DIR  := certs
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
# certs/ca.crt раздаётся клиентам (GOPHKEEPER_TLS_CERT), ca.key и server.key остаются на сервере.
# server.key получает права 0644, т.к. сервер в контейнере работает под отдельным UID;
# каталог certs/ не коммитится (см. .gitignore) и не должен быть доступен посторонним.
certs:
	@mkdir -p $(CERTS_DIR)
	@test -f $(CERTS_DIR)/ca.key || openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes \
		-keyout $(CERTS_DIR)/ca.key -out $(CERTS_DIR)/ca.crt -days $(CERT_DAYS) \
		-subj "/CN=GophKeeper CA" 2>/dev/null
	@SAN=$$(echo "$(CERT_HOSTS)" | tr ',' '\n' | while read h; do \
		if echo "$$h" | grep -Eq '^[0-9.]+$$|:'; then echo "IP:$$h"; else echo "DNS:$$h"; fi; \
	done | paste -sd, -); \
	openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes \
		-keyout $(CERTS_DIR)/server.key -out $(CERTS_DIR)/server.csr \
		-subj "/CN=gophkeeper-server" 2>/dev/null && \
	openssl x509 -req -in $(CERTS_DIR)/server.csr -CA $(CERTS_DIR)/ca.crt -CAkey $(CERTS_DIR)/ca.key \
		-CAcreateserial -out $(CERTS_DIR)/server.crt -days $(CERT_DAYS) \
		-extfile <(printf "subjectAltName=$$SAN\nextendedKeyUsage=serverAuth") 2>/dev/null && \
	rm -f $(CERTS_DIR)/server.csr $(CERTS_DIR)/ca.srl && \
	chmod 600 $(CERTS_DIR)/ca.key && chmod 644 $(CERTS_DIR)/server.key && \
	echo "Сертификат сервера выпущен для: $$SAN" && \
	echo "Клиентам передать $(CERTS_DIR)/ca.crt и задать GOPHKEEPER_TLS_CERT=<путь к ca.crt>"

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

clean:
	rm -rf $(BIN_DIR) coverage*.out .tmp-proto
