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
	"gorm.io/gorm"
)

const Version = "1.0.0"

type Server struct {
	cfg    *config.Config
	assets fs.FS
	auth   *authManager
	engine *gin.Engine
	http   *http.Server

	dbMu     sync.RWMutex
	db       *gorm.DB
	listener net.Listener
	updateMu sync.Mutex
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

func (s *Server) newEngine() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), corsMiddleware(), securityHeaders())

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
		protected.GET("/stats", s.stats)
		protected.GET("/system/info", s.systemInfo)
		protected.GET("/system/database", s.getDatabaseSettings)
		protected.POST("/system/database/test", s.testDatabaseSettings)
		protected.PUT("/system/database", s.updateDatabaseSettings)
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
		log.Printf("port %d is unavailable (%v); falling back to an ephemeral port", s.cfg.Server.Port, err)
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
			log.Printf("http server stopped: %v", err)
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
