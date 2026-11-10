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
	"syscall"
	"time"

	"resume-helper/internal/config"
	"resume-helper/internal/database"
	"resume-helper/internal/server"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// The frontend is embedded so the production binary is fully self-contained.
// The same embedded filesystem is served by Gin inside the Wails window.
//
//go:embed all:frontend/dist
var embeddedAssets embed.FS

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

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
	log.Printf("database: %s (%s)", cfg.Database.Driver, cfg.SourcePath)

	if isHeadless() {
		runHeadless(httpServer, baseURL)
		return
	}

	app := application.New(application.Options{
		Name:        "ResumeTrail",
		Description: "A beautiful resume application tracker",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(dist),
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "履途 ResumeTrail",
		URL:              baseURL,
		Width:            1360,
		Height:           860,
		MinWidth:         1024,
		MinHeight:        680,
		Frameless:        true,
		BackgroundColour: application.NewRGB(15, 23, 42),
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
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
