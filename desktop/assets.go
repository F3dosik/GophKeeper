//go:build desktop || dev

package main

import "embed"

// assets — собранный фронтенд. Встраивается только в сборках через Wails (теги desktop
// и dev), где frontend/dist создаётся перед компиляцией.
//
//go:embed all:frontend/dist
var assets embed.FS
