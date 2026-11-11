package server

import (
	"net/http"
	"os"
	"strings"
	"time"

	"resume-helper/internal/config"
	"resume-helper/internal/database"

	"github.com/gin-gonic/gin"
)

type databaseInput struct {
	database.Settings
	// UseDSN switches from the graphical fields to the raw DSN fallback.
	UseDSN bool `json:"useDsn"`
}

func (s *Server) getDatabaseSettings(c *gin.Context) {
	s.dbMu.RLock()
	driver := s.cfg.Database.Driver
	dsn := s.cfg.Database.DSN
	configPath := s.cfg.SourcePath
	dataDir := s.cfg.DataDir
	s.dbMu.RUnlock()

	settings := database.ParseSettings(driver, dsn, dataDir)
	location := ""
	if driver == "sqlite" {
		location = dsn
	}
	c.JSON(http.StatusOK, gin.H{
		"settings":    settings,
		"configPath":  configPath,
		"dataDir":     dataDir,
		"location":    location,
		"envOverride": databaseEnvOverride(),
	})
}

func (s *Server) testDatabaseSettings(c *gin.Context) {
	var input databaseInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}

	_, runtimeCfg, err := s.resolveDatabaseInput(input)
	if err != nil {
		apiError(c, http.StatusBadRequest, "invalid_database", err.Error())
		return
	}
	if err := database.TestConnection(runtimeCfg); err != nil {
		apiError(c, http.StatusBadRequest, "connection_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"message": "Connection successful.",
	})
}

func (s *Server) updateDatabaseSettings(c *gin.Context) {
	var input databaseInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}

	s.updateMu.Lock()
	defer s.updateMu.Unlock()

	storageCfg, runtimeCfg, err := s.resolveDatabaseInput(input)
	if err != nil {
		apiError(c, http.StatusBadRequest, "invalid_database", err.Error())
		return
	}

	newDB, err := database.Open(runtimeCfg)
	if err != nil {
		apiError(c, http.StatusBadRequest, "connection_failed", err.Error())
		return
	}

	s.dbMu.RLock()
	updated := *s.cfg
	s.dbMu.RUnlock()
	updated.Database = storageCfg
	if err := updated.Save(); err != nil {
		if sqlDB, dbErr := newDB.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		apiError(c, http.StatusInternalServerError, "config_save_failed", err.Error())
		return
	}

	s.dbMu.Lock()
	oldDB := s.db
	s.db = newDB
	s.cfg.Database = runtimeCfg
	s.dbMu.Unlock()
	s.auth.setDB(newDB)

	if oldDB != nil {
		go func() {
			// Give in-flight requests a moment to finish before closing the
			// previous pool.
			time.Sleep(time.Second)
			if sqlDB, dbErr := oldDB.DB(); dbErr == nil {
				_ = sqlDB.Close()
			}
		}()
	}

	s.dbMu.RLock()
	configPath := s.cfg.SourcePath
	dataDir := s.cfg.DataDir
	s.dbMu.RUnlock()

	settings := database.ParseSettings(runtimeCfg.Driver, runtimeCfg.DSN, dataDir)
	location := ""
	if runtimeCfg.Driver == "sqlite" {
		location = runtimeCfg.DSN
	}
	envOverride := databaseEnvOverride()
	warning := ""
	if envOverride {
		warning = "RESUME_HELPER_DB_DRIVER or RESUME_HELPER_DB_DSN is set. After restarting, the environment variable will override this file."
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"settings":    settings,
		"configPath":  configPath,
		"dataDir":     dataDir,
		"location":    location,
		"envOverride": envOverride,
		"warning":     warning,
	})
}

func (s *Server) resolveDatabaseInput(input databaseInput) (config.DatabaseConfig, config.DatabaseConfig, error) {
	driver, err := config.NormalizeDriver(input.Driver)
	if err != nil {
		return config.DatabaseConfig{}, config.DatabaseConfig{}, err
	}

	dsn := ""
	if input.UseDSN {
		dsn = strings.TrimSpace(input.DSN)
	} else {
		settings := input.Settings
		settings.Driver = driver
		dsn, err = database.BuildDSN(settings)
		if err != nil {
			return config.DatabaseConfig{}, config.DatabaseConfig{}, err
		}
	}

	s.dbMu.RLock()
	dataDir := s.cfg.DataDir
	s.dbMu.RUnlock()
	return config.NormalizeDatabase(dataDir, driver, dsn)
}

func databaseEnvOverride() bool {
	return strings.TrimSpace(os.Getenv("RESUME_HELPER_DB_DRIVER")) != "" ||
		strings.TrimSpace(os.Getenv("RESUME_HELPER_DB_DSN")) != ""
}
