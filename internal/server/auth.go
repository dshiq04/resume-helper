package server

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"resume-helper/internal/models"
	"resume-helper/internal/security"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	passwordSettingKey = "auth.password_hash"
	sessionTTL         = 24 * time.Hour
)

type authManager struct {
	dbMu     sync.RWMutex
	db       *gorm.DB
	mu       sync.Mutex
	sessions map[string]time.Time
}

func newAuthManager(db *gorm.DB) *authManager {
	return &authManager{
		db:       db,
		sessions: make(map[string]time.Time),
	}
}

func (a *authManager) setDB(db *gorm.DB) {
	a.dbMu.Lock()
	defer a.dbMu.Unlock()
	a.db = db
}

func (a *authManager) database() *gorm.DB {
	a.dbMu.RLock()
	defer a.dbMu.RUnlock()
	return a.db
}

func (a *authManager) passwordHash() (string, error) {
	var setting models.Setting
	err := a.database().First(&setting, "key = ?", passwordSettingKey).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

func (a *authManager) enabled() (bool, error) {
	hash, err := a.passwordHash()
	if err != nil {
		return false, err
	}
	return hash != "", nil
}

func (a *authManager) savePassword(password string) error {
	hash, err := security.HashPassword(password)
	if err != nil {
		return err
	}
	return a.database().Save(&models.Setting{
		Key:   passwordSettingKey,
		Value: hash,
	}).Error
}

func (a *authManager) clearPassword() error {
	return a.database().Delete(&models.Setting{}, "key = ?", passwordSettingKey).Error
}

func (a *authManager) verifyPassword(password string) (bool, error) {
	hash, err := a.passwordHash()
	if err != nil {
		return false, err
	}
	if hash == "" {
		return false, nil
	}
	return security.VerifyPassword(hash, password), nil
}

func (a *authManager) createSession() (string, time.Time, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token := hex.EncodeToString(raw)
	expiresAt := time.Now().Add(sessionTTL)

	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	for existing, expiry := range a.sessions {
		if now.After(expiry) {
			delete(a.sessions, existing)
		}
	}
	a.sessions[token] = expiresAt
	return token, expiresAt, nil
}

func (a *authManager) validateSession(token string) bool {
	if token == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	expiresAt, ok := a.sessions[token]
	if !ok {
		return false
	}
	if time.Now().After(expiresAt) {
		delete(a.sessions, token)
		return false
	}
	return true
}

func (a *authManager) invalidateSession(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessions, token)
}

func (a *authManager) invalidateAllSessions() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessions = make(map[string]time.Time)
}

func (a *authManager) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		enabled, err := a.enabled()
		if err != nil {
			apiError(c, http.StatusInternalServerError, "auth_failed", "Failed to verify the password setting.")
			return
		}
		if !enabled {
			c.Next()
			return
		}
		token := bearerToken(c)
		if !a.validateSession(token) {
			apiError(c, http.StatusUnauthorized, "unauthorized", "Password required.")
			return
		}
		c.Set("authToken", token)
		c.Next()
	}
}

func (s *Server) authStatus(c *gin.Context) {
	enabled, err := s.auth.enabled()
	if err != nil {
		apiError(c, http.StatusInternalServerError, "auth_failed", "Failed to verify the password setting.")
		return
	}
	authenticated := !enabled
	if enabled {
		authenticated = s.auth.validateSession(bearerToken(c))
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled":       enabled,
		"authenticated": authenticated,
	})
}

func (s *Server) authLogin(c *gin.Context) {
	enabled, err := s.auth.enabled()
	if err != nil {
		apiError(c, http.StatusInternalServerError, "auth_failed", "Failed to verify the password setting.")
		return
	}
	if !enabled {
		apiError(c, http.StatusBadRequest, "password_not_enabled", "Password protection is not enabled.")
		return
	}

	var input struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}
	ok, err := s.auth.verifyPassword(input.Password)
	if err != nil {
		apiError(c, http.StatusInternalServerError, "auth_failed", "Failed to verify the password.")
		return
	}
	if !ok {
		apiError(c, http.StatusUnauthorized, "invalid_password", "Incorrect password.")
		return
	}

	token, expiresAt, err := s.auth.createSession()
	if err != nil {
		apiError(c, http.StatusInternalServerError, "auth_failed", "Failed to create a session.")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":     token,
		"expiresAt": expiresAt,
		"enabled":   true,
	})
}

func (s *Server) authSetup(c *gin.Context) {
	enabled, err := s.auth.enabled()
	if err != nil {
		apiError(c, http.StatusInternalServerError, "auth_failed", "Failed to verify the password setting.")
		return
	}
	if enabled {
		apiError(c, http.StatusConflict, "password_already_enabled", "Password protection is already enabled.")
		return
	}

	var input struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}
	if code, message := security.ValidatePassword(input.Password); code != "" {
		apiError(c, http.StatusBadRequest, code, message)
		return
	}
	if err := s.auth.savePassword(input.Password); err != nil {
		apiError(c, http.StatusInternalServerError, "auth_failed", "Failed to save the password.")
		return
	}

	token, expiresAt, err := s.auth.createSession()
	if err != nil {
		apiError(c, http.StatusInternalServerError, "auth_failed", "Password saved but session creation failed.")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":     token,
		"expiresAt": expiresAt,
		"enabled":   true,
	})
}

func (s *Server) authLogout(c *gin.Context) {
	token, _ := c.Get("authToken")
	if value, ok := token.(string); ok {
		s.auth.invalidateSession(value)
	}
	c.JSON(http.StatusOK, gin.H{"loggedOut": true})
}

func (s *Server) authChangePassword(c *gin.Context) {
	enabled, err := s.auth.enabled()
	if err != nil {
		apiError(c, http.StatusInternalServerError, "auth_failed", "Failed to verify the password setting.")
		return
	}
	if !enabled {
		apiError(c, http.StatusBadRequest, "password_not_enabled", "Password protection is not enabled.")
		return
	}

	var input struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}
	ok, err := s.auth.verifyPassword(input.CurrentPassword)
	if err != nil {
		apiError(c, http.StatusInternalServerError, "auth_failed", "Failed to verify the password.")
		return
	}
	if !ok {
		apiError(c, http.StatusUnauthorized, "invalid_password", "Current password is incorrect.")
		return
	}
	if code, message := security.ValidatePassword(input.NewPassword); code != "" {
		apiError(c, http.StatusBadRequest, code, message)
		return
	}
	if err := s.auth.savePassword(input.NewPassword); err != nil {
		apiError(c, http.StatusInternalServerError, "auth_failed", "Failed to save the password.")
		return
	}

	// Rotate every session and keep the current client signed in.
	s.auth.invalidateAllSessions()
	token, expiresAt, err := s.auth.createSession()
	if err != nil {
		apiError(c, http.StatusInternalServerError, "auth_failed", "Password changed but session creation failed.")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":     token,
		"expiresAt": expiresAt,
	})
}

func (s *Server) authDisable(c *gin.Context) {
	enabled, err := s.auth.enabled()
	if err != nil {
		apiError(c, http.StatusInternalServerError, "auth_failed", "Failed to verify the password setting.")
		return
	}
	if !enabled {
		apiError(c, http.StatusBadRequest, "password_not_enabled", "Password protection is not enabled.")
		return
	}

	var input struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}
	ok, err := s.auth.verifyPassword(input.Password)
	if err != nil {
		apiError(c, http.StatusInternalServerError, "auth_failed", "Failed to verify the password.")
		return
	}
	if !ok {
		apiError(c, http.StatusUnauthorized, "invalid_password", "Incorrect password.")
		return
	}
	if err := s.auth.clearPassword(); err != nil {
		apiError(c, http.StatusInternalServerError, "auth_failed", "Failed to disable the password.")
		return
	}
	s.auth.invalidateAllSessions()
	c.JSON(http.StatusOK, gin.H{"enabled": false})
}

func bearerToken(c *gin.Context) string {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if header == "" {
		return ""
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}
