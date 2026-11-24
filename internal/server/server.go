package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"resume-helper/internal/appinfo"
	"resume-helper/internal/config"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Server struct {
	cfg    *config.Config
	assets fs.FS
	auth   *authManager
	engine *gin.Engine
	http   *http.Server

	// apiToken is generated once per process and injected into the served
	// index.html. Every /api request (except the health check) must carry it,
	// which stops arbitrary web pages and other local processes from reaching
	// the API even when password protection is disabled.
	apiToken string

	loggerMu sync.RWMutex
	logger   *zap.Logger

	dbMu     sync.RWMutex
	db       *gorm.DB
	listener net.Listener
	updateMu sync.Mutex

	quickEntryMu     sync.RWMutex
	quickEntryOpener func(category string)

	applicationEditMu     sync.RWMutex
	applicationEditOpener func(id uint)

	applicationViewMu     sync.RWMutex
	applicationViewOpener func(id uint)

	shortcutPromptMu      sync.RWMutex
	shortcutPromptHandler func(create bool) error

	resumePreviewMu     sync.RWMutex
	resumePreviewOpener func(id uint, title string)

	dataTransferMu     sync.RWMutex
	dataTransferOpener func(mode string)

	appInfoMu      sync.RWMutex
	appInfoHandler func()

	closePromptMu      sync.RWMutex
	closePromptHandler func(action string, remember bool) error

	backupMu sync.Mutex

	deviceMu sync.Mutex
	deviceID string
}

func New(cfg *config.Config, db *gorm.DB, assets fs.FS) *Server {
	gin.SetMode(gin.ReleaseMode)
	s := &Server{
		cfg:      cfg,
		db:       db,
		assets:   assets,
		auth:     newAuthManager(db),
		apiToken: newAPIToken(),
	}
	s.engine = s.newEngine()
	return s
}

// newAPIToken returns a random per-process bearer value. crypto/rand should
// always succeed; the hash fallback only keeps the server usable if the
// operating system entropy source is broken.
func newAPIToken() string {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		sum := sha256.Sum256([]byte(fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())))
		return hex.EncodeToString(sum[:])
	}
	return hex.EncodeToString(raw)
}

// APIToken exposes the process token so the desktop layer and tests can
// authenticate requests sent outside the embedded frontend.
func (s *Server) APIToken() string {
	return s.apiToken
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

// SetApplicationEditOpener wires the desktop application-edit window opener
// into the HTTP layer. Headless/browser mode leaves it nil.
func (s *Server) SetApplicationEditOpener(opener func(id uint)) {
	s.applicationEditMu.Lock()
	defer s.applicationEditMu.Unlock()
	s.applicationEditOpener = opener
}

// SetApplicationViewOpener wires the read-only detail window opener into the
// HTTP layer. Headless/browser mode leaves it nil.
func (s *Server) SetApplicationViewOpener(opener func(id uint)) {
	s.applicationViewMu.Lock()
	defer s.applicationViewMu.Unlock()
	s.applicationViewOpener = opener
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

// SetDataTransferOpener wires the independent backup window opener into the
// HTTP layer.
func (s *Server) SetDataTransferOpener(opener func(mode string)) {
	s.dataTransferMu.Lock()
	defer s.dataTransferMu.Unlock()
	s.dataTransferOpener = opener
}

// SetClosePromptHandler wires the close-window decision back into the desktop
// layer.
func (s *Server) SetClosePromptHandler(handler func(action string, remember bool) error) {
	s.closePromptMu.Lock()
	defer s.closePromptMu.Unlock()
	s.closePromptHandler = handler
}

// SetLogger wires a zap logger into the HTTP layer. Without it the server
// falls back to the standard library logger, which keeps tests quiet.
func (s *Server) SetLogger(logger *zap.Logger) {
	s.loggerMu.Lock()
	defer s.loggerMu.Unlock()
	s.logger = logger
}

func (s *Server) logDebug(message string, fields ...zap.Field) {
	s.loggerMu.RLock()
	logger := s.logger
	s.loggerMu.RUnlock()
	if logger != nil {
		logger.Debug(message, fields...)
	}
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

func (s *Server) logError(message string, fields ...zap.Field) {
	s.loggerMu.RLock()
	logger := s.logger
	s.loggerMu.RUnlock()
	if logger != nil {
		logger.Error(message, fields...)
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
		switch {
		case c.Writer.Status() >= 500:
			s.logError("http request failed", fields...)
		case c.Writer.Status() >= 400:
			s.logWarn("http request rejected", fields...)
		default:
			s.logInfo("http request", fields...)
		}
	}
}

func (s *Server) newEngine() *gin.Engine {
	r := gin.New()
	r.Use(s.requestLogger(), gin.Recovery(), apiErrorHandler(), s.corsMiddleware(), s.apiTokenAuth(), securityHeaders())

	api := r.Group("/api")
	{
		api.GET("/health", s.health)
		// The frontend reports the values from frontend/.env right after it
		// loads, before the lock screen, so this endpoint must stay public.
		api.POST("/app-info", s.updateAppInfo)
		api.GET("/auth/status", s.authStatus)
		api.POST("/auth/login", s.authLogin)
		api.POST("/auth/setup", s.authSetup)
		// The shortcut prompt is shown while the main window is closing, which
		// can happen after the session has been locked. Both buttons must keep
		// working without a valid session.
		api.POST("/system/shortcut/prompt", s.resolveShortcutPrompt)
		// The close prompt must work even when the app is locked, otherwise the
		// user could not choose "minimise to tray" or "quit".
		api.POST("/system/close-prompt", s.resolveClosePrompt)
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
		protected.PATCH("/applications/batch-category", s.batchUpdateCategory)
		protected.GET("/applications/companies", s.companySuggestions)
		protected.GET("/application/:id", s.getApplication)
		protected.PUT("/application/:id", s.updateApplication)
		protected.DELETE("/application/:id", s.deleteApplication)
		protected.GET("/application/:id/history", s.getApplicationHistory)
		protected.POST("/application/:id/history", s.createApplicationHistory)
		protected.POST("/applications/import", s.importApplications)
		protected.GET("/applications/template", s.downloadTemplate)
		protected.GET("/trash", s.listTrash)
		protected.POST("/trash/:id/restore", s.restoreTrash)
		protected.DELETE("/trash/:id", s.deleteTrash)
		protected.DELETE("/trash", s.emptyTrash)
		protected.GET("/resumes", s.listResumes)
		protected.POST("/resumes", s.createResume)
		protected.GET("/resume/:id", s.getResume)
		protected.PUT("/resume/:id", s.updateResume)
		protected.DELETE("/resume/:id", s.deleteResume)
		protected.GET("/resume/:id/file", s.serveResumeFile)
		protected.GET("/stats", s.stats)
		protected.GET("/stats/daily", s.dailyStats)
		protected.GET("/stats/funnel", s.applicationFunnel)
		protected.GET("/system/info", s.systemInfo)
		protected.GET("/system/device", s.deviceInfo)
		protected.GET("/system/platform", s.platformInfo)
		protected.GET("/system/preferences", s.getPreferences)
		protected.PUT("/system/preferences", s.updatePreferences)
		protected.POST("/system/data-transfer", s.openDataTransfer)
		protected.GET("/system/database", s.getDatabaseSettings)
		protected.POST("/system/database/test", s.testDatabaseSettings)
		protected.PUT("/system/database", s.updateDatabaseSettings)
		protected.GET("/system/shortcut", s.getShortcutStatus)
		protected.POST("/system/shortcut", s.createDesktopShortcut)
		protected.PUT("/system/shortcut/prompt-setting", s.updateShortcutPromptSetting)
		protected.POST("/system/quick-entry", s.openQuickEntry)
		protected.GET("/system/quick-entry/defaults", s.quickEntryDefaults)
		protected.POST("/system/application-edit", s.openApplicationEdit)
		protected.POST("/system/application-view", s.openApplicationView)
		protected.POST("/system/resume-preview", s.openResumePreview)
	}

	s.registerStaticRoutes(r)
	return r
}

// Start binds the HTTP listener and serves in a background goroutine. It
// returns the URL clients should use. If the configured port is unavailable,
// an ephemeral port is selected instead.
func (s *Server) Start() (string, error) {
	// Create the installation identifier on first launch. It is written to the
	// settings table and used to derive the resume encryption key.
	if _, err := s.ensureDeviceID(false); err != nil {
		return "", fmt.Errorf("initialize device identifier: %w", err)
	}

	host := strings.TrimSpace(s.cfg.Server.Host)
	if host == "" {
		host = "127.0.0.1"
	}

	listen := func(port int) (net.Listener, error) {
		return net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	}

	listener, err := listen(s.cfg.Server.Port)
	if err != nil {
		// Sharing the SQLite file and resume folder between two processes is
		// unsafe, so refuse to start instead of silently moving to a random
		// port when the configured one is already taken.
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
		"version": appinfo.Version(),
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

	location := dsn
	c.JSON(http.StatusOK, gin.H{
		"version":     appinfo.Version(),
		"goVersion":   runtime.Version(),
		"driver":      driver,
		"database":    databaseLabel(driver),
		"location":    location,
		"configPath":  configPath,
		"dataDir":     dataDir,
		"docsVersion": 1,
	})
}

func databaseLabel(_ string) string {
	return "SQLite"
}

// corsMiddleware only grants cross-origin access to the Wails webview and the
// application's own loopback origin. Reflecting arbitrary Origins turned the
// local API into a cross-site data leak, so unknown origins are rejected for
// API calls and never receive CORS headers.
func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		allowed := origin != "" && s.allowedOrigin(origin)
		if origin != "" && !allowed && strings.HasPrefix(c.Request.URL.Path, "/api/") {
			apiError(c, http.StatusForbidden, "origin_not_allowed", "This origin is not allowed to access the local API.")
			return
		}
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-ResumeTrail-Token")
			c.Header("Access-Control-Max-Age", "86400")
		}
		if c.Request.Method == http.MethodOptions {
			if !allowed {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (s *Server) allowedOrigin(origin string) bool {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	switch scheme {
	case "wails":
		return host == "wails" || host == "localhost" || host == "wails.localhost"
	case "http", "https":
		if host == "wails.localhost" {
			return true
		}
		if host != "127.0.0.1" && host != "localhost" && host != "::1" {
			return false
		}
		return s.originPortAllowed(parsed.Port())
	default:
		return false
	}
}

// originPortAllowed matches the loopback origin against the actual listener
// port. Before Start (tests) the listener is unknown, so any loopback port is
// accepted and the per-process API token still provides the real boundary.
func (s *Server) originPortAllowed(port string) bool {
	if s.listener == nil {
		return true
	}
	address, ok := s.listener.Addr().(*net.TCPAddr)
	if !ok {
		return true
	}
	if port == "" {
		return address.Port == 80 || address.Port == 443
	}
	return port == strconv.Itoa(address.Port)
}

// apiTokenAuth requires the per-process token on every API request except the
// public health check. The frontend receives the value through the injected
// index.html bootstrap script, so browser pages on other origins cannot read
// it (CORS) or guess it.
func (s *Server) apiTokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if !strings.HasPrefix(path, "/api/") || path == "/api/health" {
			c.Next()
			return
		}
		token := strings.TrimSpace(c.GetHeader("X-ResumeTrail-Token"))
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(s.apiToken)) != 1 {
			apiError(c, http.StatusForbidden, "invalid_app_token", "Missing or invalid application token.")
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

// apiFailure is an error that can be raised from a handler and rendered by the
// global apiErrorHandler middleware. Keeping the response in one place lets
// every endpoint share the same JSON error shape.
type apiFailure struct {
	status  int
	code    string
	message string
}

func (e *apiFailure) Error() string {
	return e.message
}

// fail raises an application error for the global interceptor instead of
// writing the response inline.
func fail(c *gin.Context, status int, code, message string) {
	_ = c.Error(&apiFailure{status: status, code: code, message: message})
	c.Abort()
}

func apiErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Written() || len(c.Errors) == 0 {
			return
		}

		failure := &apiFailure{
			status:  http.StatusInternalServerError,
			code:    "internal_error",
			message: "Something went wrong.",
		}
		for _, err := range c.Errors {
			var candidate *apiFailure
			if errors.As(err, &candidate) {
				failure = candidate
				break
			}
		}
		c.AbortWithStatusJSON(failure.status, gin.H{
			"error": failure.message,
			"code":  failure.code,
		})
	}
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
