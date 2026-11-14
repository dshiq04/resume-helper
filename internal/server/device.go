package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"resume-helper/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/moov-io/cryptfs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// deviceIDSettingKey stores the identifier generated the first time the
// application starts against a database. The identifier doubles as the seed
// for the AES key that protects resume files at rest.
const deviceIDSettingKey = "security.device_id"

// ensureDeviceID returns the identifier for this installation, creating and
// persisting it in the settings table on first launch. When refresh is true the
// setting is re-read from the current database (used after a database switch),
// but a cached identifier is still persisted when the new database has none so
// already encrypted resumes remain readable.
func (s *Server) ensureDeviceID(refresh bool) (string, error) {
	s.deviceMu.Lock()
	defer s.deviceMu.Unlock()

	if !refresh && s.deviceID != "" {
		return s.deviceID, nil
	}

	db := s.database()
	var setting models.Setting
	err := db.First(&setting, "key = ?", deviceIDSettingKey).Error
	switch {
	case err == nil:
		if value := strings.TrimSpace(setting.Value); value != "" {
			s.deviceID = value
			return value, nil
		}
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return "", err
	}

	id := s.deviceID
	if id == "" {
		generated, genErr := generateDeviceID()
		if genErr != nil {
			return "", genErr
		}
		id = generated
	}
	if err := db.Save(&models.Setting{Key: deviceIDSettingKey, Value: id}).Error; err != nil {
		return "", err
	}
	s.deviceID = id
	return id, nil
}

// generateDeviceID returns a 32-byte random identifier encoded as hex. The
// underlying bytes are used as an AES-256 key after a SHA-256 derivation.
func generateDeviceID() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func (s *Server) resumeCipher() (*cryptfs.FS, error) {
	id, err := s.ensureDeviceID(false)
	if err != nil {
		return nil, err
	}
	key := sha256.Sum256([]byte(id))
	return cryptfs.FromCryptor(cryptfs.NewAESCryptor(key[:]))
}

// writeEncryptedResume encrypts the plaintext with AES-256-GCM through cryptfs
// and writes the ciphertext to target. The file is created exclusively so a
// random name collision can never overwrite an existing resume.
func (s *Server) writeEncryptedResume(target string, plaintext []byte) error {
	cipher, err := s.resumeCipher()
	if err != nil {
		return err
	}
	encrypted, err := cipher.Disfigure(plaintext)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(encrypted); err != nil {
		_ = file.Close()
		_ = os.Remove(target)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(target)
		return err
	}
	return nil
}

// readResumeFile returns the plaintext of a stored resume. Files uploaded
// before encryption was introduced are marked Encrypted=false and are read
// unchanged for backward compatibility.
func (s *Server) readResumeFile(record models.Resume) ([]byte, error) {
	filePath := filepath.Join(s.resumeDirectory(), record.StoredName)
	if !record.Encrypted {
		return os.ReadFile(filePath)
	}
	cipher, err := s.resumeCipher()
	if err != nil {
		return nil, err
	}
	return cipher.ReadFile(filePath)
}

func (s *Server) deviceInfo(c *gin.Context) {
	id, err := s.ensureDeviceID(false)
	if err != nil {
		s.logError("device identifier unavailable", zap.Error(err))
		apiError(c, http.StatusInternalServerError, "device_id_failed", "Failed to load the device identifier.")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"deviceId":      id,
		"algorithm":     "AES-256-GCM",
		"keyDerivation": "SHA-256",
	})
}
