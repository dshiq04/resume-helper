package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"resume-helper/internal/config"
	"resume-helper/internal/database"
	"resume-helper/internal/desktop"
	"resume-helper/internal/server"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// The frontend is embedded so the production binary is fully self-contained.
// The same embedded filesystem is served by Gin inside the Wails window.
//
//go:embed all:frontend/dist
var embeddedAssets embed.FS

//go:embed frontend/public/app-icon.png
var appIconPNG []byte

//go:embed frontend/public/app-icon.ico
var appIconICO []byte

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}
	desktop.Configure(cfg.DataDir, appIconICO)

	db, err := database.Open(cfg.Database)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	dist, err := fs.Sub(embeddedAssets, "frontend/dist")
	if err != nil {
		log.Fatalf("load embedded frontend: %v", err)
	}

	httpServer := server.New(cfg, db, dist)
	baseURL, err := httpServer.Start()
	if err != nil {
		log.Fatalf("start HTTP server: %v", err)
	}
	log.Printf("ResumeTrail %s is serving at %s", server.Version, baseURL)
	log.Printf(
		"database: %s %s (config: %s)",
		database.DatabaseLabel(cfg.Database.Driver),
		database.SafeDSN(cfg.Database.Driver, cfg.Database.DSN),
		cfg.SourcePath,
	)

	if isHeadless() {
		runHeadless(httpServer, baseURL)
		return
	}

	// Serve the frontend through the Wails asset server instead of loading the
	// Gin URL directly. This keeps window APIs (minimise / maximise /
	// fullscreen / close) and resize handling on the Wails origin, while the
	// Gin engine continues to handle both the static files and /api requests.
	app := application.New(application.Options{
		Name:        "ResumeTrail",
		Description: "A beautiful resume application tracker",
		Icon:        appIconPNG,
		Assets: application.AssetOptions{
			Handler: httpServer.Handler(),
		},
	})

	mainWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "履途 ResumeTrail",
		Width:            1360,
		Height:           860,
		MinWidth:         360,
		MinHeight:        480,
		Frameless:        true,
		DisableResize:    false,
		BackgroundColour: application.NewRGB(15, 23, 42),
	})
	shortcutPrompt := &shortcutPromptState{}
	registerShortcutClosePrompt(app, mainWindow, shortcutPrompt)
	httpServer.SetShortcutPromptHandler(func(create bool) error {
		return resolveShortcutPrompt(app, mainWindow, shortcutPrompt, create)
	})
	httpServer.SetQuickEntryOpener(func(category string) {
		openQuickEntryWindow(app, category)
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func openQuickEntryWindow(app *application.App, category string) {
	application.InvokeAsync(func() {
		if existing, ok := app.Window.Get("quick-entry"); ok {
			existing.Show()
			existing.Focus()
			return
		}

		window := app.Window.NewWithOptions(application.WebviewWindowOptions{
			Name:             "quick-entry",
			Title:            "快速录入",
			URL:              "/quick-entry?category=" + category,
			Width:            400,
			Height:           170,
			MinWidth:         400,
			MinHeight:        170,
			MaxWidth:         400,
			MaxHeight:        170,
			Frameless:        true,
			DisableResize:    true,
			AlwaysOnTop:      true,
			InitialPosition:  application.WindowCentered,
			BackgroundColour: application.NewRGB(15, 23, 42),
		})
		window.Show()
		window.Focus()
	})
}

type shortcutPromptState struct {
	allowClose atomic.Bool
	prompting  atomic.Bool
}

func registerShortcutClosePrompt(
	app *application.App,
	window *application.WebviewWindow,
	state *shortcutPromptState,
) {
	if !desktop.ShortcutSupported() {
		return
	}

	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if state.allowClose.Load() {
			return
		}
		exists, err := desktop.ShortcutExists()
		if err != nil || exists {
			return
		}

		event.Cancel()
		if !state.prompting.CompareAndSwap(false, true) {
			application.InvokeAsync(func() {
				if existing, ok := app.Window.Get("shortcut-prompt"); ok {
					existing.Show()
					existing.Focus()
				}
			})
			return
		}
		openShortcutPromptWindow(app, state)
	})
}

func openShortcutPromptWindow(app *application.App, state *shortcutPromptState) {
	application.InvokeAsync(func() {
		if existing, ok := app.Window.Get("shortcut-prompt"); ok {
			existing.Show()
			existing.Focus()
			return
		}

		window := app.Window.NewWithOptions(application.WebviewWindowOptions{
			Name:             "shortcut-prompt",
			Title:            "桌面快捷方式",
			URL:              "/shortcut-prompt",
			Width:            440,
			Height:           220,
			MinWidth:         440,
			MinHeight:        220,
			MaxWidth:         440,
			MaxHeight:        220,
			Frameless:        true,
			DisableResize:    true,
			InitialPosition:  application.WindowCentered,
			BackgroundColour: application.NewRGB(15, 23, 42),
		})
		window.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
			state.prompting.Store(false)
		})
		window.Show()
		window.Focus()
	})
}

func resolveShortcutPrompt(
	app *application.App,
	mainWindow *application.WebviewWindow,
	state *shortcutPromptState,
	create bool,
) error {
	if create {
		if err := desktop.CreateShortcut(); err != nil {
			return err
		}
	}

	application.InvokeAsync(func() {
		if promptWindow, ok := app.Window.Get("shortcut-prompt"); ok {
			promptWindow.Close()
		}
		state.allowClose.Store(true)
		state.prompting.Store(false)
		mainWindow.Close()
	})
	return nil
}

func isHeadless() bool {
	if strings.EqualFold(os.Getenv("RESUME_HELPER_HEADLESS"), "true") || os.Getenv("RESUME_HELPER_HEADLESS") == "1" {
		return true
	}
	for _, arg := range os.Args[1:] {
		if arg == "--headless" || arg == "-headless" {
			return true
		}
	}
	return false
}

func runHeadless(httpServer *server.Server, baseURL string) {
	log.Printf("headless mode enabled; open %s in a browser", baseURL)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Stop(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("shutdown: %v", err)
	}
}
