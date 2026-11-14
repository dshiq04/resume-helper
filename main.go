package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"resume-helper/internal/config"
	"resume-helper/internal/database"
	"resume-helper/internal/desktop"
	"resume-helper/internal/logging"
	"resume-helper/internal/models"
	"resume-helper/internal/server"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"go.uber.org/zap"
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

	appLogger, logPath, closeLog, err := logging.Setup(cfg.DataDir)
	if err != nil {
		log.Fatalf("setup logging: %v", err)
	}
	defer closeLog()
	appLogger.Info("ResumeTrail starting", zap.String("logFile", logPath))
	appLogger.Debug("configuration loaded",
		zap.String("dataDir", cfg.DataDir),
		zap.String("config", cfg.SourcePath),
		zap.String("driver", cfg.Database.Driver),
	)

	db, err := database.Open(cfg.Database)
	if err != nil {
		appLogger.Fatal("open database", zap.Error(err))
	}
	appLogger.Debug("database connection ready",
		zap.String("driver", cfg.Database.Driver),
		zap.String("dsn", database.SafeDSN(cfg.Database.Driver, cfg.Database.DSN)),
	)

	dist, err := fs.Sub(embeddedAssets, "frontend/dist")
	if err != nil {
		appLogger.Fatal("load embedded frontend", zap.Error(err))
	}

	httpServer := server.New(cfg, db, dist)
	httpServer.SetLogger(appLogger)
	baseURL, err := httpServer.Start()
	if err != nil {
		appLogger.Fatal("start HTTP server", zap.Error(err))
	}
	appLogger.Info("ResumeTrail is serving",
		zap.String("version", server.Version),
		zap.String("url", baseURL),
		zap.String("database", database.DatabaseLabel(cfg.Database.Driver)),
		zap.String("dsn", database.SafeDSN(cfg.Database.Driver, cfg.Database.DSN)),
		zap.String("config", cfg.SourcePath),
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
		Services: []application.Service{
			application.NewService(&DropImporter{server: httpServer}),
		},
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
		EnableFileDrop:   true,
		BackgroundColour: application.NewRGB(15, 23, 42),
	})
	shortcutPrompt := &shortcutPromptState{}
	registerShortcutClosePrompt(app, mainWindow, shortcutPrompt, httpServer.ShortcutPromptEnabled)
	registerFileDropEvent(appLogger, mainWindow)
	httpServer.SetShortcutPromptHandler(func(create bool) error {
		return resolveShortcutPrompt(app, mainWindow, shortcutPrompt, create)
	})
	httpServer.SetQuickEntryOpener(func(category string) {
		openQuickEntryWindow(app, category)
	})
	httpServer.SetApplicationEditOpener(func(id uint) {
		openApplicationEditWindow(app, id)
	})
	httpServer.SetResumePreviewOpener(func(id uint, title string) {
		openResumePreviewWindow(app, id, title)
	})

	if err := app.Run(); err != nil {
		appLogger.Fatal("application stopped", zap.Error(err))
	}
}

// DropImporter exposes resume import helpers to the frontend through Wails
// bindings. Drag-and-drop from the operating system only provides file paths,
// so the file has to be copied on the Go side.
type DropImporter struct {
	server *server.Server
}

func (d *DropImporter) ProbeResumeFile(path string) (*server.ResumeFileInfo, error) {
	return d.server.ProbeResumeFile(path)
}

func (d *DropImporter) ImportResume(path string, name string, tags []string) (*models.Resume, error) {
	return d.server.ImportResumeFromPath(path, name, tags)
}

func (d *DropImporter) ProbeExcelFile(path string) (*server.ExcelFileInfo, error) {
	return d.server.ProbeExcelFile(path)
}

func (d *DropImporter) ImportExcel(path string) (*server.ImportResult, error) {
	return d.server.ImportExcelFromPath(path)
}

// registerFileDropEvent forwards file drops that land on a known upload area
// to the frontend. Wails native drops carry full paths, which the frontend
// cannot read directly, so imports happen through Wails bindings.
func registerFileDropEvent(logger *zap.Logger, mainWindow *application.WebviewWindow) {
	mainWindow.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		paths := event.Context().DroppedFiles()
		if len(paths) == 0 {
			return
		}
		details := event.Context().DropTargetDetails()
		if details == nil {
			return
		}
		target := details.Attributes["data-file-drop-target"]
		if target != server.ResumeDropTarget && target != server.ExcelDropTarget {
			return
		}
		mainWindow.EmitEvent("resumetrail:file-dropped", map[string]any{
			"target": target,
			"files":  paths,
		})
		logger.Debug("files dropped",
			zap.String("target", target),
			zap.Strings("files", paths),
		)
	})
}

func openResumePreviewWindow(app *application.App, id uint, title string) {
	application.InvokeAsync(func() {
		target := "/resume-preview?id=" + strconv.FormatUint(uint64(id), 10)
		windowTitle := "简历预览"
		if trimmed := strings.TrimSpace(title); trimmed != "" {
			windowTitle += " · " + trimmed
		}

		if existing, ok := app.Window.Get("resume-preview"); ok {
			existing.SetTitle(windowTitle)
			existing.SetAlwaysOnTop(false)
			existing.SetURL(target)
			existing.Show()
			existing.Focus()
			return
		}

		window := app.Window.NewWithOptions(application.WebviewWindowOptions{
			Name:             "resume-preview",
			Title:            windowTitle,
			URL:              target,
			Width:            1120,
			Height:           840,
			MinWidth:         520,
			MinHeight:        420,
			Frameless:        true,
			DisableResize:    false,
			InitialPosition:  application.WindowCentered,
			BackgroundColour: application.NewRGB(15, 23, 42),
		})
		window.Show()
		window.Focus()
	})
}

func openApplicationEditWindow(app *application.App, id uint) {
	application.InvokeAsync(func() {
		target := "/application-edit?id=" + strconv.FormatUint(uint64(id), 10)
		windowTitle := "修改投递"

		if existing, ok := app.Window.Get("application-edit"); ok {
			existing.SetTitle(windowTitle)
			existing.SetURL(target)
			existing.Show()
			existing.Focus()
			return
		}

		window := app.Window.NewWithOptions(application.WebviewWindowOptions{
			Name:             "application-edit",
			Title:            windowTitle,
			URL:              target,
			Width:            520,
			Height:           560,
			MinWidth:         420,
			MinHeight:        430,
			Frameless:        true,
			DisableResize:    false,
			InitialPosition:  application.WindowCentered,
			BackgroundColour: application.NewRGB(15, 23, 42),
		})
		window.Show()
		window.Focus()
	})
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
	promptEnabled func() bool,
) {
	if !desktop.ShortcutSupported() {
		return
	}

	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if state.allowClose.Load() {
			return
		}
		if !promptEnabled() {
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
			Height:           300,
			MinWidth:         440,
			MinHeight:        300,
			MaxWidth:         440,
			MaxHeight:        300,
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
