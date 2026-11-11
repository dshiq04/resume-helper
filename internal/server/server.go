package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"resume-helper/internal/config"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const Version = "1.0.0"

type Server struct {
	cfg    *config.Config
	assets fs.FS
	auth   *authManager
	engine *gin.Engine
	http   *http.Server

	loggerMu sync.RWMutex
	logger   *zap.Logger

	dbMu     sync.RWMutex
	db       *gorm.DB
	listener net.Listener
	updateMu sync.Mutex

	quickEntryMu     sync.RWMutex
	quickEntryOpener func(category string)

	shortcutPromptMu      sync.RWMutex
	shortcutPromptHandler func(create bool) error

	resumePreviewMu     sync.RWMutex
	resumePreviewOpener func(id uint, title string)
}

func New(cfg *config.Config, db *gorm.DB, assets fs.FS) *Server {
	gin.SetMode(gin.ReleaseMode)
	s := &Server{
		cfg:    cfg,
		db:     db,
		assets: assets,
		auth:   newAuthManager(db),
	}
	s.engine = s.newEngine()
	return s
}

// Handler exposes the Gin engine so the Wails asset server can serve both the
// embedded frontend and /api requests from the same origin. Keeping the
// frontend on the Wails origin is required for /wails/runtime calls (window
// minimise, maximise, fullscreen and close) to reach the desktop host.
func (s *Server) Handler() http.Handler {
	return s.engine
}

func (s *Server) database() *gorm.DB {
	s.dbMu.RLock()
	defer s.dbMu.RUnlock()
	return s.db
}

// SetQuickEntryOpener wires the desktop window opener into the HTTP layer.
// Headless/browser mode leaves it nil.
func (s *Server) SetQuickEntryOpener(opener func(category string)) {
	s.quickEntryMu.Lock()
	defer s.quickEntryMu.Unlock()
	s.quickEntryOpener = opener
}

// SetShortcutPromptHandler wires the shortcut-prompt window decision back
// into the desktop layer.
func (s *Server) SetShortcutPromptHandler(handler func(create bool) error) {
	s.shortcutPromptMu.Lock()
	defer s.shortcutPromptMu.Unlock()
	s.shortcutPromptHandler = handler
}

// SetResumePreviewOpener wires the resume preview window opener into the HTTP
// layer. Headless/browser mode leaves it nil.
func (s *Server) SetResumePreviewOpener(opener func(id uint, title string)) {
	s.resumePreviewMu.Lock()
	defer s.resumePreviewMu.Unlock()
	s.resumePreviewOpener = opener
}

// SetLogger wires a zap logger into the HTTP layer. Without it the server
// falls back to the standard library logger, which keeps tests quiet.
func (s *Server) SetLogger(logger *zap.Logger) {
	s.loggerMu.Lock()
	defer s.loggerMu.Unlock()
	s.logger = logger
}

func (s *Server) logInfo(message string, fields ...zap.Field) {
	s.loggerMu.RLock()
	logger := s.logger
	s.loggerMu.RUnlock()
	if logger != nil {
		logger.Info(message, fields...)
		return
	}
	log.Println(message)
}

func (s *Server) logWarn(message string, fields ...zap.Field) {
	s.loggerMu.RLock()
	logger := s.logger
	s.loggerMu.RUnlock()
	if logger != nil {
		logger.Warn(message, fields...)
		return
	}
	log.Println(message)
}

func (s *Server) requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.Request.URL.Path
		if !strings.HasPrefix(path, "/api/") {
			return
		}

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.ClientIP()),
		}
		if c.Writer.Status() >= 500 {
			s.logWarn("http request failed", fields...)
			return
		}
		s.logInfo("http request", fields...)
	}
}

func (s *Server) newEngine() *gin.Engine {
	r := gin.New()
	r.Use(s.requestLogger(), gin.Recovery(), corsMiddleware(), securityHeaders())

	api := r.Group("/api")
	{
		api.GET("/health", s.health)
		api.GET("/auth/status", s.authStatus)
		api.POST("/auth/login", s.authLogin)
		api.POST("/auth/setup", s.authSetup)
	}

	protected := api.Group("")
	protected.Use(s.auth.requireAuth())
	{
		protected.POST("/auth/logout", s.authLogout)
		protected.POST("/auth/change-password", s.authChangePassword)
		protected.POST("/auth/disable", s.authDisable)

		protected.GET("/applications", s.listApplications)
		protected.POST("/applications", s.createApplication)
		protected.POST("/applications/batch-delete", s.batchDeleteApplications)
		protected.PATCH("/applications/batch-status", s.batchUpdateStatus)
		protected.PUT("/application/:id", s.updateApplication)
		protected.DELETE("/application/:id", s.deleteApplication)
		protected.POST("/applications/import", s.importApplications)
		protected.GET("/applications/template", s.downloadTemplate)
		protected.GET("/resumes", s.listResumes)
		protected.POST("/resumes", s.createResume)
		protected.GET("/resume/:id", s.getResume)
		protected.PUT("/resume/:id", s.updateResume)
		protected.DELETE("/resume/:id", s.deleteResume)
		protected.GET("/resume/:id/file", s.serveResumeFile)
		protected.GET("/stats", s.stats)
		protected.GET("/system/info", s.systemInfo)
		protected.GET("/system/database", s.getDatabaseSettings)
		protected.POST("/system/database/test", s.testDatabaseSettings)
		protected.PUT("/system/database", s.updateDatabaseSettings)
		protected.GET("/system/shortcut", s.getShortcutStatus)
		protected.POST("/system/shortcut", s.createDesktopShortcut)
		protected.POST("/system/shortcut/prompt", s.resolveShortcutPrompt)
		protected.POST("/system/quick-entry", s.openQuickEntry)
		protected.POST("/system/resume-preview", s.openResumePreview)
	}

	s.registerStaticRoutes(r)
	return r
}

// Start binds the HTTP listener and serves in a background goroutine. It
// returns the URL clients should use. If the configured port is unavailable,
// an ephemeral port is selected instead.
func (s *Server) Start() (string, error) {
	host := strings.TrimSpace(s.cfg.Server.Host)
	if host == "" {
		host = "127.0.0.1"
	}

	listen := func(port int) (net.Listener, error) {
		return net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	}

	listener, err := listen(s.cfg.Server.Port)
	if err != nil && s.cfg.Server.Port != 0 {
		s.logWarn("port is unavailable; falling back to an ephemeral port",
			zap.Int("port", s.cfg.Server.Port), zap.Error(err))
		listener, err = listen(0)
	}
	if err != nil {
		return "", fmt.Errorf("listen on %s: %w", host, err)
	}
	s.listener = listener
	s.http = &http.Server{
		Handler:           s.engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := s.http.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logWarn("http server stopped", zap.Error(err))
		}
	}()

	actualPort := listener.Addr().(*net.TCPAddr).Port
	displayHost := host
	if displayHost == "0.0.0.0" || displayHost == "::" || displayHost == "[::]" {
		displayHost = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s", net.JoinHostPort(displayHost, strconv.Itoa(actualPort))), nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.http == nil {
		return nil
	}
	return s.http.Shutdown(ctx)
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": Version,
		"time":    time.Now(),
	})
}

func (s *Server) systemInfo(c *gin.Context) {
	s.dbMu.RLock()
	driver := s.cfg.Database.Driver
	dsn := s.cfg.Database.DSN
	configPath := s.cfg.SourcePath
	dataDir := s.cfg.DataDir
	s.dbMu.RUnlock()

	location := ""
	if driver == "sqlite" {
		location = dsn
	}
	c.JSON(http.StatusOK, gin.H{
		"version":     Version,
		"goVersion":   runtime.Version(),
		"driver":      driver,
		"database":    databaseLabel(driver),
		"location":    location,
		"configPath":  configPath,
		"dataDir":     dataDir,
		"docsVersion": 1,
	})
}

func databaseLabel(driver string) string {
	switch driver {
	case "mysql":
		return "MySQL"
	case "postgres":
		return "PostgreSQL"
	default:
		return "SQLite"
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}

func apiError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": message,
		"code":  code,
	})
}

func parseID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || value == 0 {
		apiError(c, http.StatusBadRequest, "invalid_id", "Invalid record id.")
		return 0, false
	}
	return uint(value), true
}

func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
