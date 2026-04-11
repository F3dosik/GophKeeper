.PHONY: generate

# Кодогенерация из .proto файлов
generate:
	mkdir -p proto/gen
	protoc \
		--go_out=proto/gen \
		--go_opt=paths=source_relative \
		--go-grpc_out=proto/gen \
		--go-grpc_opt=paths=source_relative \
		--go_opt=default_api_level=API_OPAQUE \
		proto/*.proto