package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"resume-helper/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	maxResumeFileSize = 50 << 20 // 50 MB
	maxResumeTags     = 12
	maxResumeTagRunes = 32
	resumeFolderName  = "resume"
)

// supportedResumeTypes maps the accepted lowercase extensions to the MIME type
// used when the file is streamed back to the preview window.
var supportedResumeTypes = map[string]string{
	"docx":     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"pdf":      "application/pdf",
	"pptx":     "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"png":      "image/png",
	"md":       "text/markdown; charset=utf-8",
	"markdown": "text/markdown; charset=utf-8",
}

type resumeUpdateInput struct {
	Name   *string   `json:"name"`
	Tags   *[]string `json:"tags"`
	Pinned *bool     `json:"pinned"`
}

type resumeListResponse struct {
	Items []models.Resume `json:"items"`
	Total int64           `json:"total"`
}

func (s *Server) resumeDirectory() string {
	return filepath.Join(s.cfg.DataDir, resumeFolderName)
}

func (s *Server) listResumes(c *gin.Context) {
	db := s.database()
	build := func() *gorm.DB {
		query := db.Model(&models.Resume{})
		if search := strings.TrimSpace(c.Query("search")); search != "" {
			like := "%" + strings.ToLower(search) + "%"
			query = query.Where("LOWER(name) LIKE ? OR LOWER(file_name) LIKE ? OR LOWER(tags) LIKE ?", like, like, like)
		}
		return query
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load resumes.")
		return
	}

	items := []models.Resume{}
	if err := build().Order("pinned DESC, created_at DESC, id DESC").Find(&items).Error; err != nil {
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load resumes.")
		return
	}
	for index := range items {
		if items[index].Tags == nil {
			items[index].Tags = []string{}
		}
	}

	c.JSON(http.StatusOK, resumeListResponse{Items: items, Total: total})
}

func (s *Server) getResume(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	record, ok := s.findResume(c, id)
	if !ok {
		return
	}
	if record.Tags == nil {
		record.Tags = []string{}
	}
	c.JSON(http.StatusOK, record)
}

func (s *Server) createResume(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxResumeFileSize+(4<<20))

	header, err := c.FormFile("file")
	if err != nil {
		apiError(c, http.StatusBadRequest, "file_required", "Please choose a resume file.")
		return
	}
	if header.Size > maxResumeFileSize {
		apiError(c, http.StatusBadRequest, "file_too_large", "The file is larger than 50 MB.")
		return
	}
	if header.Size <= 0 {
		apiError(c, http.StatusBadRequest, "file_empty", "The selected file is empty.")
		return
	}

	fileName := sanitizeUploadFileName(header.Filename)
	extension := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	_, supported := supportedResumeTypes[extension]
	if !supported {
		apiError(c, http.StatusBadRequest, "unsupported_file_type", "Only DOCX, PDF, PPTX, PNG and Markdown files are supported.")
		return
	}
	if fileName == "" {
		fileName = "resume." + extension
	}

	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		name = strings.TrimSuffix(fileName, filepath.Ext(fileName))
	}
	if name == "" {
		name = "Resume"
	}
	if utf8.RuneCountInString(name) > 255 {
		apiError(c, http.StatusBadRequest, "name_too_long", "Resume name is too long.")
		return
	}

	tags, code, message := parseResumeTags(c.PostForm("tags"))
	if code != "" {
		apiError(c, http.StatusBadRequest, code, message)
		return
	}

	dir := s.resumeDirectory()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		apiError(c, http.StatusInternalServerError, "storage_unavailable", "Failed to create the resume folder.")
		return
	}

	storedName, err := generateStoredName(extension)
	if err != nil {
		apiError(c, http.StatusInternalServerError, "storage_unavailable", "Failed to prepare the resume file.")
		return
	}
	target := filepath.Join(dir, storedName)
	if err := saveMultipartFile(header, target); err != nil {
		apiError(c, http.StatusInternalServerError, "storage_unavailable", "Failed to save the resume file.")
		return
	}

	record := &models.Resume{
		Name:       name,
		Tags:       tags,
		FileName:   fileName,
		StoredName: storedName,
		FileType:   extension,
		FileSize:   header.Size,
	}
	if err := s.database().Create(record).Error; err != nil {
		_ = os.Remove(target)
		apiError(c, http.StatusInternalServerError, "create_failed", "Failed to create the resume.")
		return
	}

	c.JSON(http.StatusCreated, record)
}

func (s *Server) updateResume(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	record, ok := s.findResume(c, id)
	if !ok {
		return
	}

	var input resumeUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}

	updates := map[string]interface{}{}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			apiError(c, http.StatusBadRequest, "name_required", "Resume name is required.")
			return
		}
		if utf8.RuneCountInString(name) > 255 {
			apiError(c, http.StatusBadRequest, "name_too_long", "Resume name is too long.")
			return
		}
		updates["name"] = name
	}
	if input.Tags != nil {
		tags, code, message := normalizeResumeTags(*input.Tags)
		if code != "" {
			apiError(c, http.StatusBadRequest, code, message)
			return
		}
		encoded, err := json.Marshal(tags)
		if err != nil {
			apiError(c, http.StatusInternalServerError, "update_failed", "Failed to update the resume.")
			return
		}
		updates["tags"] = string(encoded)
	}
	if input.Pinned != nil {
		updates["pinned"] = *input.Pinned
	}

	if len(updates) > 0 {
		if err := s.database().Model(&record).Updates(updates).Error; err != nil {
			apiError(c, http.StatusInternalServerError, "update_failed", "Failed to update the resume.")
			return
		}
	}

	if err := s.database().First(&record, id).Error; err != nil {
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to reload the resume.")
		return
	}
	if record.Tags == nil {
		record.Tags = []string{}
	}
	c.JSON(http.StatusOK, record)
}

func (s *Server) deleteResume(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	record, ok := s.findResume(c, id)
	if !ok {
		return
	}

	if err := s.database().Delete(&models.Resume{}, id).Error; err != nil {
		apiError(c, http.StatusInternalServerError, "delete_failed", "Failed to delete the resume.")
		return
	}
	_ = os.Remove(filepath.Join(s.resumeDirectory(), record.StoredName))
	c.JSON(http.StatusOK, gin.H{"deleted": 1})
}

func (s *Server) serveResumeFile(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	record, ok := s.findResume(c, id)
	if !ok {
		return
	}

	contentType, supported := supportedResumeTypes[record.FileType]
	if !supported {
		contentType = "application/octet-stream"
	}
	filePath := filepath.Join(s.resumeDirectory(), record.StoredName)
	if info, err := os.Stat(filePath); err != nil || info.IsDir() {
		apiError(c, http.StatusNotFound, "file_missing", "The resume file is missing.")
		return
	}

	mode := "inline"
	if c.Query("download") == "1" {
		mode = "attachment"
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", contentDisposition(mode, record.FileName))
	c.Header("Cache-Control", "no-store")
	c.File(filePath)
}

func (s *Server) openResumePreview(c *gin.Context) {
	var input struct {
		ID uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.ID == 0 {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}

	record, ok := s.findResume(c, input.ID)
	if !ok {
		return
	}

	s.resumePreviewMu.RLock()
	opener := s.resumePreviewOpener
	s.resumePreviewMu.RUnlock()
	if opener == nil {
		apiError(c, http.StatusServiceUnavailable, "desktop_only", "The preview window is only available in desktop mode.")
		return
	}

	opener(record.ID, record.Name)
	c.JSON(http.StatusOK, gin.H{"opened": true})
}

func (s *Server) findResume(c *gin.Context, id uint) (models.Resume, bool) {
	var record models.Resume
	if err := s.database().First(&record, id).Error; err != nil {
		if isRecordNotFound(err) {
			apiError(c, http.StatusNotFound, "not_found", "Resume not found.")
			return record, false
		}
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load the resume.")
		return record, false
	}
	return record, true
}

func parseResumeTags(raw string) ([]string, string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}, "", ""
	}
	var tags []string
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &tags); err != nil {
			return nil, "invalid_tags", "Resume tags are invalid."
		}
	} else {
		tags = strings.Split(raw, ",")
	}
	return normalizeResumeTags(tags)
}

func normalizeResumeTags(tags []string) ([]string, string, string) {
	result := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if utf8.RuneCountInString(tag) > maxResumeTagRunes {
			return nil, "tag_too_long", "A resume tag is too long."
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, tag)
		if len(result) > maxResumeTags {
			return nil, "too_many_tags", "Too many resume tags."
		}
	}
	return result, "", ""
}

func sanitizeUploadFileName(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	value = path.Base(value)
	value = strings.TrimSpace(value)
	if value == "." || value == "/" {
		return ""
	}
	if utf8.RuneCountInString(value) > 255 {
		extension := filepath.Ext(value)
		base := strings.TrimSuffix(value, extension)
		runes := []rune(base)
		maxBase := 255 - utf8.RuneCountInString(extension)
		if maxBase < 0 {
			maxBase = 0
		}
		if len(runes) > maxBase {
			runes = runes[:maxBase]
		}
		value = string(runes) + extension
	}
	return value
}

func generateStoredName(extension string) (string, error) {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return time.Now().Format("20060102-150405") + "-" + hex.EncodeToString(raw) + "." + extension, nil
}

func saveMultipartFile(header *multipart.FileHeader, target string) error {
	source, err := header.Open()
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer destination.Close()

	written, err := io.Copy(destination, io.LimitReader(source, maxResumeFileSize+1))
	if err != nil {
		_ = os.Remove(target)
		return err
	}
	if written > maxResumeFileSize {
		_ = os.Remove(target)
		return fmt.Errorf("resume file exceeds %d bytes", maxResumeFileSize)
	}
	return nil
}

func contentDisposition(mode, fileName string) string {
	fallback := strings.Map(func(r rune) rune {
		if r < 32 || r > 126 || r == '"' || r == '\\' {
			return '_'
		}
		return r
	}, fileName)
	if strings.TrimSpace(fallback) == "" {
		fallback = "resume"
	}
	return fmt.Sprintf("%s; filename=\"%s\"; filename*=UTF-8''%s", mode, fallback, url.PathEscape(fileName))
}
