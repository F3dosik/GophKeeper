// Command desktop — десктоп-клиент GophKeeper на Wails.
package main

import (
	"context"
	"log"
	"os"

	"github.com/F3dosik/GophKeeper/desktop/backend"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func main() {
	// GOPHKEEPER_DESKTOP_DIR задаёт другой каталог данных: вторая учётка, разработка, тесты.
	dir := os.Getenv("GOPHKEEPER_DESKTOP_DIR")
	if dir == "" {
		var err error
		if dir, err = backend.DefaultDir(); err != nil {
			log.Fatal(err)
		}
	}

	ui := &wailsUI{}
	app := backend.NewApp(backend.Options{Dir: dir, UI: ui})

	err := wails.Run(&options.App{
		Title:     "GophKeeper",
		Width:     1000,
		Height:    700,
		MinWidth:  720,
		MinHeight: 480,
		// GOPHKEEPER_DESKTOP_HIDDEN=1 запускает без окна: для автотестов интерфейса
		// через браузер в режиме wails dev.
		StartHidden: os.Getenv("GOPHKEEPER_DESKTOP_HIDDEN") == "1",
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 24, G: 26, B: 31, A: 1},
		OnStartup: func(ctx context.Context) {
			ui.ctx = ctx
			if err := backend.Startup(app, ctx); err != nil {
				log.Printf("startup: %v", err)
			}
		},
		OnShutdown: func(context.Context) {
			backend.Shutdown(app)
		},
		Bind: []any{app},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// wailsUI реализует backend.UI через runtime Wails.
type wailsUI struct {
	ctx context.Context
}

func (u *wailsUI) OpenFile(title string, filters []backend.FileFilter) (string, error) {
	opts := runtime.OpenDialogOptions{Title: title}
	for _, f := range filters {
		opts.Filters = append(opts.Filters, runtime.FileFilter{DisplayName: f.DisplayName, Pattern: f.Pattern})
	}
	return runtime.OpenFileDialog(u.ctx, opts)
}

func (u *wailsUI) SaveFile(title, defaultName string) (string, error) {
	return runtime.SaveFileDialog(u.ctx, runtime.SaveDialogOptions{Title: title, DefaultFilename: defaultName})
}

func (u *wailsUI) ClipboardSet(text string) error {
	return runtime.ClipboardSetText(u.ctx, text)
}

func (u *wailsUI) ClipboardGet() (string, error) {
	return runtime.ClipboardGetText(u.ctx)
}

func (u *wailsUI) Emit(event string, data ...any) {
	runtime.EventsEmit(u.ctx, event, data...)
}
