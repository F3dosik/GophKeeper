//go:build !desktop && !dev

package main

import "embed"

// assets пуст вне сборок Wails: так `go build ./...` и `go test ./...` всего репозитория
// работают без собранного фронтенда (frontend/dist не хранится в git).
var assets embed.FS
