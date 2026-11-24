package server

import (
	"archive/zip"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"resume-helper/internal/appinfo"
	"resume-helper/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	backupFormat          = "resumetrail-backup"
	backupFormatVersion   = 1
	backupManifestFile    = "manifest.json"
	backupResumeDir       = "resumes"
	maxBackupManifestSize = 64 << 20
	maxBackupFileSize     = maxResumeFileSize
)

// backupVersion is the ResumeTrail version that produced a backup. It is the
// first key in manifest.json so tools can check compatibility without reading
// the rest of the file. Older backups used the same key for a numeric format
// version, which decodes to an empty value and falls back to appVersion.
type backupVersion string

func (v *backupVersion) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*v = ""
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*v = backupVersion(strings.TrimSpace(text))
		return nil
	}
	var legacy int
	if err := json.Unmarshal(data, &legacy); err == nil {
		*v = ""
		return nil
	}
	return errors.New("backup version must be a string")
}

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
	Version       backupVersion               `json:"version"`
	Format        string                      `json:"format"`
	FormatVersion int                         `json:"formatVersion"`
	AppVersion    string                      `json:"appVersion"`
	ExportedAt    time.Time                   `json:"exportedAt"`
	Applications  []models.Application        `json:"applications"`
	Resumes       []backupResume              `json:"resumes"`
	Settings      []backupSetting             `json:"settings"`
	Changes       []models.ApplicationChange  `json:"changes"`
	Trash         []models.DeletedApplication `json:"trash"`
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
	Path          string       `json:"path"`
	ExportedAt    time.Time    `json:"exportedAt"`
	Version       string       `json:"version"`
	AppVersion    string       `json:"appVersion"`
	ImportBlocked bool         `json:"importBlocked"`
	Counts        BackupCounts `json:"counts"`
	Warnings      []string     `json:"warnings"`
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
		Version:       backupVersion(appinfo.Version()),
		Format:        backupFormat,
		FormatVersion: backupFormatVersion,
		AppVersion:    appinfo.Version(),
		ExportedAt:    time.Now().UTC(),
		Applications:  []models.Application{},
		Resumes:       []backupResume{},
		Settings:      []backupSetting{},
		Changes:       []models.ApplicationChange{},
		Trash:         []models.DeletedApplication{},
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
	type resumeArchive struct {
		resume  backupResume
		content []byte
	}
	warnings := []string{}
	archives := make([]resumeArchive, 0, len(resumes))
	for _, resume := range resumes {
		content, err := s.readResumeFile(resume)
		if err != nil {
			// A single missing or undecryptable file must not make the whole
			// backup unusable. Skip it and surface a warning to the UI.
			warnings = append(warnings, fmt.Sprintf("skipped resume %q: %v", resume.FileName, err))
			continue
		}
		if int64(len(content)) != resume.FileSize && resume.FileSize > 0 {
			warnings = append(warnings, fmt.Sprintf("%s size differs from its metadata", resume.FileName))
		}
		archiveName := path.Join(backupResumeDir, fmt.Sprintf("%d-%s", resume.ID, backupFileName(resume)))
		archive := backupResume{Resume: resume, ArchivePath: archiveName}
		manifest.Resumes = append(manifest.Resumes, archive)
		archives = append(archives, resumeArchive{resume: archive, content: content})
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
		Resumes:      len(archives),
		Settings:     len(manifest.Settings),
		Changes:      len(manifest.Changes),
		Trash:        len(manifest.Trash),
		Files:        len(archives),
	}

	manifestHeader := &zip.FileHeader{Name: backupManifestFile, Method: zip.Deflate, Modified: time.Now()}
	manifestWriter, err := writer.CreateHeader(manifestHeader)
	if err != nil {
		return nil, fmt.Errorf("create manifest: %w", err)
	}
	if err := json.NewEncoder(manifestWriter).Encode(manifest); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	for _, archive := range archives {
		header := &zip.FileHeader{Name: archive.resume.ArchivePath, Method: zip.Deflate, Modified: archive.resume.CreatedAt}
		if header.Modified.IsZero() {
			header.Modified = time.Now()
		}
		entry, err := writer.CreateHeader(header)
		if err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("write resume %q: %w", archive.resume.FileName, err)
		}
		if _, err := entry.Write(archive.content); err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("write resume %q: %w", archive.resume.FileName, err)
		}
		counts.Bytes += int64(len(archive.content))
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finish backup: %w", err)
	}
	if err := temp.Close(); err != nil {
		return nil, fmt.Errorf("close backup: %w", err)
	}
	if err := os.Rename(tempPath, target); err != nil {
		// Windows cannot replace an existing file with Rename. Move the old
		// backup aside, install the new one, and only then remove the old
		// file, so a failed replace never destroys both copies.
		if _, statErr := os.Stat(target); statErr != nil {
			return nil, fmt.Errorf("save backup: %w", err)
		}
		backupPath := target + ".old-" + strconv.FormatInt(time.Now().UnixNano(), 10)
		if moveErr := os.Rename(target, backupPath); moveErr != nil {
			return nil, fmt.Errorf("save backup: %w", err)
		}
		if retry := os.Rename(tempPath, target); retry != nil {
			if restoreErr := os.Rename(backupPath, target); restoreErr != nil {
				return nil, fmt.Errorf("save backup: %w (the previous backup was kept at %s)", retry, backupPath)
			}
			return nil, fmt.Errorf("save backup: %w", retry)
		}
		_ = os.Remove(backupPath)
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
		Version:    string(manifest.Version),
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

// unicodePathExtraID is the Info-ZIP Unicode Path extra field. Archivers that
// store names in a local ANSI code page (for example the Windows compressed
// folder tool) add it with the UTF-8 spelling of the name.
const unicodePathExtraID = 0x7075

// backupEntryName returns the name of a zip entry, preferring the UTF-8 name
// from the Info-ZIP Unicode Path extra field. archive/zip ignores that field,
// so a backup re-zipped on Windows can otherwise expose GBK file names.
func backupEntryName(file *zip.File) string {
	if name, ok := unicodePathName(file.Extra, file.Name); ok {
		return name
	}
	return file.Name
}

func unicodePathName(extra []byte, rawName string) (string, bool) {
	for len(extra) >= 4 {
		id := binary.LittleEndian.Uint16(extra[0:2])
		size := int(binary.LittleEndian.Uint16(extra[2:4]))
		if size+4 > len(extra) {
			return "", false
		}
		data := extra[4 : 4+size]
		if id == unicodePathExtraID && size >= 5 && data[0] == 1 {
			// The checksum covers the raw name bytes, so a stale extra field
			// left over from a rename is ignored.
			if binary.LittleEndian.Uint32(data[1:5]) == crc32.ChecksumIEEE([]byte(rawName)) {
				name := string(data[5:])
				if utf8.ValidString(name) && name != "" {
					return name, true
				}
			}
		}
		extra = extra[4+size:]
	}
	return "", false
}

func normalizeBackupEntryName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Clean(name)
	name = strings.TrimPrefix(name, "/")
	if name == "." {
		return ""
	}
	return name
}

// findBackupResumeEntry locates the archive entry that holds a resume. The
// manifest path is used first; when an archiver dropped or added a folder or
// id prefix, the entry is matched by its file name instead.
func findBackupResumeEntry(entries map[string]*zip.File, byFileName map[string][]*zip.File, resume backupResume) *zip.File {
	archivePath := normalizeBackupEntryName(resume.ArchivePath)
	if entry, ok := entries[archivePath]; ok {
		return entry
	}
	baseName := path.Base(archivePath)
	candidates := byFileName[baseName]
	if len(candidates) == 0 {
		fileName := normalizeBackupEntryName(sanitizeUploadFileName(resume.FileName))
		if fileName != "" && fileName != baseName {
			candidates = byFileName[fileName]
		}
	}
	if len(candidates) == 1 {
		return candidates[0]
	}
	return nil
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
		if file.FileInfo().IsDir() {
			continue
		}
		name := normalizeBackupEntryName(backupEntryName(file))
		if strings.HasPrefix(name, backupResumeDir+"/") {
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
		Path:          source,
		ExportedAt:    manifest.ExportedAt,
		Version:       backupSourceVersion(manifest),
		AppVersion:    manifest.AppVersion,
		ImportBlocked: backupImportBlocked(manifest),
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
	// Match manifest.json at any folder depth so a backup that was unpacked
	// and compressed again (backup/manifest.json) still imports.
	var candidates []*zip.File
	var rootLevel []*zip.File
	for _, file := range files {
		if file.FileInfo().IsDir() {
			continue
		}
		name := normalizeBackupEntryName(backupEntryName(file))
		if path.Base(name) != backupManifestFile {
			continue
		}
		candidates = append(candidates, file)
		if name == backupManifestFile {
			rootLevel = append(rootLevel, file)
		}
	}
	if len(candidates) == 0 {
		return nil, errors.New("manifest.json is missing from the backup")
	}
	if len(candidates) > 1 {
		// A single root-level manifest wins; otherwise the archive is
		// ambiguous and the user must repack it deterministically.
		if len(rootLevel) != 1 {
			return nil, errors.New("the backup contains multiple manifest.json files")
		}
		candidates = rootLevel
	}
	manifestFile := candidates[0]
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
	if manifest.FormatVersion > backupFormatVersion {
		return nil, fmt.Errorf("the backup format %d is newer than this application supports", manifest.FormatVersion)
	}
	if err := checkBackupVersion(&manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// checkBackupVersion refuses backups exported by a newer application than the
// one currently running. Older backups are always accepted.
func checkBackupVersion(manifest *backupManifest) error {
	source := backupSourceVersion(manifest)
	client := strings.TrimSpace(appinfo.Version())
	if source == "" || client == "" {
		return nil
	}
	sourceVersion, ok := parseAppVersion(source)
	if !ok {
		return nil
	}
	clientVersion, ok := parseAppVersion(client)
	if !ok {
		return nil
	}
	if compareAppVersions(clientVersion, sourceVersion) < 0 {
		return fmt.Errorf(
			"this backup was exported from version %s, but the current version is %s; upgrade %s to %s or newer before importing",
			source,
			client,
			appinfo.Name(),
			source,
		)
	}
	return nil
}

func backupSourceVersion(manifest *backupManifest) string {
	if version := strings.TrimSpace(string(manifest.Version)); version != "" {
		return version
	}
	return strings.TrimSpace(manifest.AppVersion)
}

// isDevOrBetaVersion reports whether a version was produced by an unstable
// development or beta build. Those backups support export only.
func isDevOrBetaVersion(version string) bool {
	normalized := strings.ToLower(strings.TrimSpace(version))
	return strings.Contains(normalized, "dev") || strings.Contains(normalized, "beta")
}

// backupImportBlocked reports whether a backup may not be imported. Backups
// from dev/beta builds are export-only unless they come from exactly the same
// version that is running now.
func backupImportBlocked(manifest *backupManifest) bool {
	source := backupSourceVersion(manifest)
	if !isDevOrBetaVersion(source) {
		return false
	}
	return !sameAppVersion(source, appinfo.Version())
}

func sameAppVersion(left, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == right {
		return true
	}
	leftVersion, ok := parseAppVersion(left)
	if !ok {
		return false
	}
	rightVersion, ok := parseAppVersion(right)
	if !ok {
		return false
	}
	return compareAppVersions(leftVersion, rightVersion) == 0
}

type appVersion struct {
	core []int
	pre  string
}

// parseAppVersion understands versions such as 1.2, v1.2.0, 1.2-beta and
// 1.2.0+build. Unparseable values are ignored so a custom version string never
// locks users out of their own backups.
func parseAppVersion(value string) (appVersion, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "v")
	if value == "" {
		return appVersion{}, false
	}
	if index := strings.IndexByte(value, '+'); index >= 0 {
		value = value[:index]
	}
	pre := ""
	if index := strings.IndexByte(value, '-'); index >= 0 {
		pre = value[index+1:]
		value = value[:index]
	}
	parts := strings.Split(value, ".")
	core := make([]int, 0, len(parts))
	for _, part := range parts {
		number, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return appVersion{}, false
		}
		core = append(core, number)
	}
	if len(core) == 0 {
		return appVersion{}, false
	}
	return appVersion{core: core, pre: pre}, true
}

// compareAppVersions returns -1, 0 or 1 depending on whether a is older than,
// equal to or newer than b. A release is newer than its pre-releases, so
// 1.2 > 1.2-beta and 1.2-beta > 1.2-alpha.
func compareAppVersions(a, b appVersion) int {
	length := len(a.core)
	if len(b.core) > length {
		length = len(b.core)
	}
	for index := 0; index < length; index++ {
		left, right := 0, 0
		if index < len(a.core) {
			left = a.core[index]
		}
		if index < len(b.core) {
			right = b.core[index]
		}
		if left < right {
			return -1
		}
		if left > right {
			return 1
		}
	}
	switch {
	case a.pre == "" && b.pre == "":
		return 0
	case a.pre == "":
		return 1
	case b.pre == "":
		return -1
	default:
		return comparePrerelease(a.pre, b.pre)
	}
}

// comparePrerelease implements the semver precedence rules for dot-separated
// identifiers: numeric identifiers compare numerically, alphanumeric ones
// compare lexically, and numeric identifiers have lower precedence than
// alphanumeric ones. This makes beta.10 newer than beta.2.
func comparePrerelease(a, b string) int {
	if a == b {
		return 0
	}
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	for index := 0; index < len(aParts) || index < len(bParts); index++ {
		if index >= len(aParts) {
			return -1
		}
		if index >= len(bParts) {
			return 1
		}
		left, right := aParts[index], bParts[index]
		leftNumber, leftErr := strconv.Atoi(left)
		rightNumber, rightErr := strconv.Atoi(right)
		switch {
		case leftErr == nil && rightErr == nil:
			if leftNumber < rightNumber {
				return -1
			}
			if leftNumber > rightNumber {
				return 1
			}
		case leftErr == nil:
			return -1
		case rightErr == nil:
			return 1
		default:
			if left < right {
				return -1
			}
			if left > right {
				return 1
			}
		}
	}
	return 0
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
	if version := backupSourceVersion(manifest); backupImportBlocked(manifest) {
		return nil, fmt.Errorf(
			"this backup was exported by dev/beta version %s and cannot be imported by version %s; dev and beta versions are for development and testing only and support export only",
			version,
			appinfo.Version(),
		)
	}

	deviceID, err := s.ensureDeviceID(false)
	if err != nil {
		return nil, fmt.Errorf("load the device identifier: %w", err)
	}

	entries := make(map[string]*zip.File, len(reader.File))
	resumeEntries := make(map[string][]*zip.File)
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		name := normalizeBackupEntryName(backupEntryName(file))
		if name == "" {
			continue
		}
		if _, exists := entries[name]; !exists {
			entries[name] = file
		}
		if strings.HasPrefix(name, backupResumeDir+"/") {
			baseName := path.Base(name)
			resumeEntries[baseName] = append(resumeEntries[baseName], file)
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
		entry := findBackupResumeEntry(entries, resumeEntries, resume)
		if entry == nil {
			cleanup()
			return nil, fmt.Errorf("resume file %q is missing from the backup (expected %q)", resume.FileName, resume.ArchivePath)
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
		if !models.IsValidCategory(item.Category) || !models.IsValidStatus(item.Status) {
			cleanup()
			return nil, fmt.Errorf("recycle bin item %q has an invalid category or status", item.CompanyName)
		}
		if strings.TrimSpace(item.CompanyName) == "" {
			cleanup()
			return nil, fmt.Errorf("recycle bin item %d is missing a company name", item.OriginalID)
		}
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
	// Only delete files that the previous database actually referenced. Files
	// the user placed in resume/ manually, or that belong to another database,
	// must be left alone.
	for _, storedName := range oldStoredNames {
		if _, keep := newStored[storedName]; keep {
			continue
		}
		_ = os.Remove(filepath.Join(s.resumeDirectory(), storedName))
	}

	// Importing replaces the settings table, including the password hash, so
	// every in-memory session must be invalidated. The frontend re-checks the
	// auth status and shows the lock screen when the imported backup has a
	// password.
	s.auth.invalidateAllSessions()

	s.logInfo("backup imported",
		zap.String("path", source),
		zap.Int("applications", counts.Applications),
		zap.Int("resumes", counts.Resumes),
	)
	return &BackupResult{
		Path:       source,
		ExportedAt: manifest.ExportedAt,
		Version:    backupSourceVersion(manifest),
		AppVersion: manifest.AppVersion,
		Counts:     counts,
		Warnings:   []string{},
	}, nil
}
