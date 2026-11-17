package server

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"resume-helper/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	backupFormat          = "resumetrail-backup"
	backupVersion         = 1
	backupManifestFile    = "manifest.json"
	backupResumeDir       = "resumes"
	maxBackupManifestSize = 64 << 20
	maxBackupFileSize     = maxResumeFileSize
)

type backupSetting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type backupResume struct {
	models.Resume
	ArchivePath string `json:"archivePath"`
}

type backupManifest struct {
	Format       string                      `json:"format"`
	Version      int                         `json:"version"`
	AppVersion   string                      `json:"appVersion"`
	ExportedAt   time.Time                   `json:"exportedAt"`
	Applications []models.Application        `json:"applications"`
	Resumes      []backupResume              `json:"resumes"`
	Settings     []backupSetting             `json:"settings"`
	Changes      []models.ApplicationChange  `json:"changes"`
	Trash        []models.DeletedApplication `json:"trash"`
}

// BackupCounts summarises one backup operation for the transfer window.
type BackupCounts struct {
	Applications int   `json:"applications"`
	Resumes      int   `json:"resumes"`
	Settings     int   `json:"settings"`
	Changes      int   `json:"changes"`
	Trash        int   `json:"trash"`
	Files        int   `json:"files"`
	Bytes        int64 `json:"bytes"`
}

// BackupResult is returned by export, inspect and import.
type BackupResult struct {
	Path       string       `json:"path"`
	ExportedAt time.Time    `json:"exportedAt"`
	AppVersion string       `json:"appVersion"`
	Counts     BackupCounts `json:"counts"`
	Warnings   []string     `json:"warnings"`
}

func (s *Server) ExportBackup(target string) (*BackupResult, error) {
	s.backupMu.Lock()
	defer s.backupMu.Unlock()
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, errors.New("backup path is required")
	}
	if !strings.HasSuffix(strings.ToLower(target), ".zip") {
		target += ".zip"
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return nil, fmt.Errorf("create backup folder: %w", err)
	}

	db := s.database()
	manifest := backupManifest{
		Format:       backupFormat,
		Version:      backupVersion,
		AppVersion:   Version,
		ExportedAt:   time.Now(),
		Applications: []models.Application{},
		Resumes:      []backupResume{},
		Settings:     []backupSetting{},
		Changes:      []models.ApplicationChange{},
		Trash:        []models.DeletedApplication{},
	}
	if err := db.Order("id ASC").Find(&manifest.Applications).Error; err != nil {
		return nil, fmt.Errorf("load applications: %w", err)
	}
	if err := db.Order("id ASC").Find(&manifest.Changes).Error; err != nil {
		return nil, fmt.Errorf("load status history: %w", err)
	}
	if err := db.Order("id ASC").Find(&manifest.Trash).Error; err != nil {
		return nil, fmt.Errorf("load recycle bin: %w", err)
	}

	var settings []models.Setting
	if err := db.Order("key ASC").Find(&settings).Error; err != nil {
		return nil, fmt.Errorf("load settings: %w", err)
	}
	for _, setting := range settings {
		// The device identifier is deliberately never exported: it is the
		// resume encryption key and must be created fresh on the target device.
		if setting.Key == deviceIDSettingKey {
			continue
		}
		manifest.Settings = append(manifest.Settings, backupSetting{
			Key:       setting.Key,
			Value:     setting.Value,
			UpdatedAt: setting.UpdatedAt,
		})
	}

	var resumes []models.Resume
	if err := db.Order("id ASC").Find(&resumes).Error; err != nil {
		return nil, fmt.Errorf("load resumes: %w", err)
	}
	for _, resume := range resumes {
		archiveName := path.Join(backupResumeDir, fmt.Sprintf("%d-%s", resume.ID, backupFileName(resume)))
		manifest.Resumes = append(manifest.Resumes, backupResume{Resume: resume, ArchivePath: archiveName})
	}

	temp, err := os.CreateTemp(filepath.Dir(target), ".resumetrail-backup-*.zip")
	if err != nil {
		return nil, fmt.Errorf("create backup file: %w", err)
	}
	tempPath := temp.Name()
	success := false
	defer func() {
		_ = temp.Close()
		if !success {
			_ = os.Remove(tempPath)
		}
	}()

	writer := zip.NewWriter(temp)
	counts := BackupCounts{
		Applications: len(manifest.Applications),
		Resumes:      len(manifest.Resumes),
		Settings:     len(manifest.Settings),
		Changes:      len(manifest.Changes),
		Trash:        len(manifest.Trash),
		Files:        len(manifest.Resumes),
	}
	warnings := []string{}

	manifestHeader := &zip.FileHeader{Name: backupManifestFile, Method: zip.Deflate, Modified: time.Now()}
	manifestWriter, err := writer.CreateHeader(manifestHeader)
	if err != nil {
		return nil, fmt.Errorf("create manifest: %w", err)
	}
	if err := json.NewEncoder(manifestWriter).Encode(manifest); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	for _, resume := range manifest.Resumes {
		content, err := s.readResumeFile(resume.Resume)
		if err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("decrypt resume %q: %w", resume.FileName, err)
		}
		if int64(len(content)) != resume.FileSize && resume.FileSize > 0 {
			warnings = append(warnings, fmt.Sprintf("%s size differs from its metadata", resume.FileName))
		}
		header := &zip.FileHeader{Name: resume.ArchivePath, Method: zip.Deflate, Modified: resume.CreatedAt}
		if header.Modified.IsZero() {
			header.Modified = time.Now()
		}
		entry, err := writer.CreateHeader(header)
		if err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("write resume %q: %w", resume.FileName, err)
		}
		if _, err := entry.Write(content); err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("write resume %q: %w", resume.FileName, err)
		}
		counts.Bytes += int64(len(content))
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finish backup: %w", err)
	}
	if err := temp.Close(); err != nil {
		return nil, fmt.Errorf("close backup: %w", err)
	}
	if err := os.Rename(tempPath, target); err != nil {
		// Windows cannot replace an existing file with Rename.
		if removeErr := os.Remove(target); removeErr == nil {
			if retry := os.Rename(tempPath, target); retry != nil {
				return nil, fmt.Errorf("save backup: %w", retry)
			}
		} else {
			return nil, fmt.Errorf("save backup: %w", err)
		}
	}
	success = true

	s.logInfo("backup exported",
		zap.String("path", target),
		zap.Int("applications", counts.Applications),
		zap.Int("resumes", counts.Resumes),
	)
	return &BackupResult{
		Path:       target,
		ExportedAt: manifest.ExportedAt,
		AppVersion: manifest.AppVersion,
		Counts:     counts,
		Warnings:   warnings,
	}, nil
}

func backupFileName(resume models.Resume) string {
	name := strings.ReplaceAll(resume.FileName, "\\", "/")
	name = path.Base(name)
	if name == "" || name == "." || name == "/" {
		name = fmt.Sprintf("resume-%d.%s", resume.ID, resume.FileType)
	}
	return sanitizeUploadFileName(name)
}

func (s *Server) InspectBackup(source string) (*BackupResult, error) {
	reader, err := zip.OpenReader(strings.TrimSpace(source))
	if err != nil {
		return nil, fmt.Errorf("open backup: %w", err)
	}
	defer reader.Close()

	manifest, err := readBackupManifest(reader.File)
	if err != nil {
		return nil, err
	}
	files := 0
	var bytes int64
	for _, file := range reader.File {
		if strings.HasPrefix(file.Name, backupResumeDir+"/") && !file.FileInfo().IsDir() {
			files++
			bytes += int64(file.UncompressedSize64)
		}
	}
	settings := 0
	for _, setting := range manifest.Settings {
		if setting.Key != deviceIDSettingKey {
			settings++
		}
	}
	return &BackupResult{
		Path:       source,
		ExportedAt: manifest.ExportedAt,
		AppVersion: manifest.AppVersion,
		Counts: BackupCounts{
			Applications: len(manifest.Applications),
			Resumes:      len(manifest.Resumes),
			Settings:     settings,
			Changes:      len(manifest.Changes),
			Trash:        len(manifest.Trash),
			Files:        files,
			Bytes:        bytes,
		},
	}, nil
}

func readBackupManifest(files []*zip.File) (*backupManifest, error) {
	var manifestFile *zip.File
	for _, file := range files {
		if file.Name == backupManifestFile {
			manifestFile = file
			break
		}
	}
	if manifestFile == nil {
		return nil, errors.New("manifest.json is missing from the backup")
	}
	if manifestFile.UncompressedSize64 > maxBackupManifestSize {
		return nil, errors.New("manifest.json is too large")
	}
	handle, err := manifestFile.Open()
	if err != nil {
		return nil, fmt.Errorf("open manifest: %w", err)
	}
	defer handle.Close()

	var manifest backupManifest
	if err := json.NewDecoder(io.LimitReader(handle, maxBackupManifestSize)).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if manifest.Format != backupFormat {
		return nil, errors.New("this file is not a ResumeTrail backup")
	}
	if manifest.Version > backupVersion {
		return nil, fmt.Errorf("the backup version %d is newer than this application supports", manifest.Version)
	}
	return &manifest, nil
}

func (s *Server) ImportBackup(source string) (*BackupResult, error) {
	s.backupMu.Lock()
	defer s.backupMu.Unlock()
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, errors.New("backup path is required")
	}
	reader, err := zip.OpenReader(source)
	if err != nil {
		return nil, fmt.Errorf("open backup: %w", err)
	}
	defer reader.Close()

	manifest, err := readBackupManifest(reader.File)
	if err != nil {
		return nil, err
	}

	deviceID, err := s.ensureDeviceID(false)
	if err != nil {
		return nil, fmt.Errorf("load the device identifier: %w", err)
	}

	entries := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		if !file.FileInfo().IsDir() {
			entries[file.Name] = file
		}
	}

	if err := os.MkdirAll(s.resumeDirectory(), 0o755); err != nil {
		return nil, fmt.Errorf("create resume folder: %w", err)
	}

	type stagedResume struct {
		record models.Resume
		size   int64
	}
	staged := make([]stagedResume, 0, len(manifest.Resumes))
	createdFiles := make([]string, 0, len(manifest.Resumes))
	seenResumes := map[uint]struct{}{}
	cleanup := func() {
		for _, name := range createdFiles {
			_ = os.Remove(filepath.Join(s.resumeDirectory(), name))
		}
	}

	for _, resume := range manifest.Resumes {
		if resume.ID != 0 {
			if _, exists := seenResumes[resume.ID]; exists {
				continue
			}
			seenResumes[resume.ID] = struct{}{}
		}
		if !IsSupportedResumeType(resume.FileType) {
			cleanup()
			return nil, fmt.Errorf("resume %q has an unsupported type", resume.FileName)
		}
		entry, ok := entries[resume.ArchivePath]
		if !ok {
			cleanup()
			return nil, fmt.Errorf("resume file %q is missing from the backup", resume.FileName)
		}
		if entry.UncompressedSize64 > maxBackupFileSize {
			cleanup()
			return nil, fmt.Errorf("resume %q is larger than 50 MB", resume.FileName)
		}
		handle, err := entry.Open()
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("read resume %q: %w", resume.FileName, err)
		}
		content, readErr := io.ReadAll(io.LimitReader(handle, maxBackupFileSize+1))
		_ = handle.Close()
		if readErr != nil {
			cleanup()
			return nil, fmt.Errorf("read resume %q: %w", resume.FileName, readErr)
		}
		if int64(len(content)) > maxBackupFileSize {
			cleanup()
			return nil, fmt.Errorf("resume %q is larger than 50 MB", resume.FileName)
		}

		storedName, err := generateStoredName()
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("prepare resume %q: %w", resume.FileName, err)
		}
		target := filepath.Join(s.resumeDirectory(), storedName)
		if err := s.writeEncryptedResume(target, content); err != nil {
			cleanup()
			return nil, fmt.Errorf("encrypt resume %q: %w", resume.FileName, err)
		}
		createdFiles = append(createdFiles, storedName)

		record := resume.Resume
		record.StoredName = storedName
		record.Encrypted = true
		record.FileSize = int64(len(content))
		if record.Tags == nil {
			record.Tags = []string{}
		}
		staged = append(staged, stagedResume{record: record, size: int64(len(content))})
	}

	// Keep only valid, de-duplicated status milestones so a hand-edited
	// manifest cannot break the unique index during import.
	changes := make([]models.ApplicationChange, 0, len(manifest.Changes))
	seenChanges := map[string]struct{}{}
	for _, change := range manifest.Changes {
		key := fmt.Sprintf("%d|%s", change.ApplicationID, change.Status)
		if _, exists := seenChanges[key]; exists {
			continue
		}
		seenChanges[key] = struct{}{}
		change.ID = 0
		changes = append(changes, change)
	}
	sortChanges(changes)

	applications := make([]models.Application, 0, len(manifest.Applications))
	seenApplications := map[uint]struct{}{}
	for _, application := range manifest.Applications {
		if application.ID == 0 {
			continue
		}
		if _, exists := seenApplications[application.ID]; exists {
			continue
		}
		if !models.IsValidCategory(application.Category) || !models.IsValidStatus(application.Status) {
			cleanup()
			return nil, fmt.Errorf("application %d has an invalid category or status", application.ID)
		}
		seenApplications[application.ID] = struct{}{}
		applications = append(applications, application)
	}

	trash := make([]models.DeletedApplication, 0, len(manifest.Trash))
	for _, item := range manifest.Trash {
		item.ID = 0
		trash = append(trash, item)
	}

	var oldStoredNames []string
	db := s.database()
	if err := db.Model(&models.Resume{}).Pluck("stored_name", &oldStoredNames).Error; err != nil {
		cleanup()
		return nil, fmt.Errorf("load existing resumes: %w", err)
	}

	counts := BackupCounts{
		Applications: len(applications),
		Resumes:      len(staged),
		Changes:      len(changes),
		Trash:        len(trash),
		Files:        len(staged),
	}
	for _, item := range staged {
		counts.Bytes += item.size
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).
			Delete(&models.DeletedApplication{}).Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).
			Delete(&models.ApplicationChange{}).Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).
			Delete(&models.Application{}).Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).
			Delete(&models.Resume{}).Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).
			Delete(&models.Setting{}).Error; err != nil {
			return err
		}

		if err := tx.Save(&models.Setting{Key: deviceIDSettingKey, Value: deviceID}).Error; err != nil {
			return err
		}
		settings := make([]models.Setting, 0, len(manifest.Settings))
		seenSettings := map[string]struct{}{}
		for _, setting := range manifest.Settings {
			if strings.TrimSpace(setting.Key) == "" || setting.Key == deviceIDSettingKey {
				continue
			}
			if _, exists := seenSettings[setting.Key]; exists {
				continue
			}
			seenSettings[setting.Key] = struct{}{}
			settings = append(settings, models.Setting{
				Key:       setting.Key,
				Value:     setting.Value,
				UpdatedAt: setting.UpdatedAt,
			})
		}
		counts.Settings = len(settings)
		if len(settings) > 0 {
			if err := tx.Create(&settings).Error; err != nil {
				return err
			}
		}

		if len(applications) > 0 {
			if err := tx.Create(&applications).Error; err != nil {
				return err
			}
		}
		if len(staged) > 0 {
			records := make([]models.Resume, 0, len(staged))
			for _, item := range staged {
				records = append(records, item.record)
			}
			if err := tx.Create(&records).Error; err != nil {
				return err
			}
		}
		if len(changes) > 0 {
			if err := tx.Create(&changes).Error; err != nil {
				return err
			}
		}
		if len(trash) > 0 {
			if err := tx.Create(&trash).Error; err != nil {
				return err
			}
		}

		// PostgreSQL sequences do not advance when rows are inserted with an
		// explicit id. Move them past the restored ids so the next normal
		// insert cannot collide with an imported row.
		if tx.Dialector.Name() == "postgres" {
			for _, table := range []string{"applications", "resumes"} {
				statement := fmt.Sprintf(
					"SELECT setval(pg_get_serial_sequence('%s','id'), COALESCE((SELECT MAX(id) FROM %s), 0) + 1, false)",
					table,
					table,
				)
				if err := tx.Exec(statement).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("write imported data: %w", err)
	}

	newStored := make(map[string]struct{}, len(staged))
	for _, item := range staged {
		newStored[item.record.StoredName] = struct{}{}
	}
	for _, storedName := range oldStoredNames {
		if _, keep := newStored[storedName]; keep {
			continue
		}
		_ = os.Remove(filepath.Join(s.resumeDirectory(), storedName))
	}
	if dirEntries, err := os.ReadDir(s.resumeDirectory()); err == nil {
		for _, entry := range dirEntries {
			if entry.IsDir() {
				continue
			}
			if _, keep := newStored[entry.Name()]; keep {
				continue
			}
			_ = os.Remove(filepath.Join(s.resumeDirectory(), entry.Name()))
		}
	}

	s.logInfo("backup imported",
		zap.String("path", source),
		zap.Int("applications", counts.Applications),
		zap.Int("resumes", counts.Resumes),
	)
	return &BackupResult{
		Path:       source,
		ExportedAt: manifest.ExportedAt,
		AppVersion: manifest.AppVersion,
		Counts:     counts,
		Warnings:   []string{},
	}, nil
}
