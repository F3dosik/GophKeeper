.PHONY:  generate

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