package server

import (
	"archive/zip"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
	"time"

	"resume-helper/internal/models"
)

// writeBackupZip creates a backup whose resume entry uses the given raw zip
// name and optional Info-ZIP Unicode Path extra field. Windows' built-in zip
// tool stores non-ASCII names in the local code page and records the UTF-8
// spelling in that extra field.
func writeBackupZip(t *testing.T, target string, resume backupResume, rawEntryName string, unicodeEntryName string, content []byte) {
	t.Helper()
	manifest := backupManifest{
		Format:        backupFormat,
		FormatVersion: backupFormatVersion,
		ExportedAt:    time.Now(),
		Applications:  []models.Application{},
		Resumes:       []backupResume{resume},
		Settings:      []backupSetting{},
		Changes:       []models.ApplicationChange{},
		Trash:         []models.DeletedApplication{},
	}

	file, err := os.Create(target)
	if err != nil {
		t.Fatalf("create backup: %v", err)
	}
	writer := zip.NewWriter(file)

	manifestWriter, err := writer.Create(backupManifestFile)
	if err != nil {
		t.Fatalf("create manifest entry: %v", err)
	}
	if err := json.NewEncoder(manifestWriter).Encode(manifest); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	header := &zip.FileHeader{Name: rawEntryName, Method: zip.Deflate, Modified: time.Now()}
	if unicodeEntryName != "" {
		extra := make([]byte, 4+1+4+len(unicodeEntryName))
		binary.LittleEndian.PutUint16(extra[0:2], unicodePathExtraID)
		binary.LittleEndian.PutUint16(extra[2:4], uint16(1+4+len(unicodeEntryName)))
		extra[4] = 1
		binary.LittleEndian.PutUint32(extra[5:9], crc32.ChecksumIEEE([]byte(rawEntryName)))
		copy(extra[9:], unicodeEntryName)
		header.Extra = extra
	}
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatalf("create resume entry: %v", err)
	}
	if _, err := entry.Write(content); err != nil {
		t.Fatalf("write resume entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close backup: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close backup file: %v", err)
	}
}

func TestImportBackupFindsResumeWithWindowsZipEncoding(t *testing.T) {
	// GBK bytes for "简历7.pdf", as written by the Windows compressed folder
	// tool on a Chinese system.
	gbkName := "resumes/\xbc\xf2\xc0\xfa7.pdf"
	content := []byte("%PDF-1.4 fake resume")

	app := newTestApp(t)
	target := filepath.Join(t.TempDir(), "windows.zip")
	writeBackupZip(t, target, backupResume{
		Resume: models.Resume{
			ID:       5,
			Name:     "简历",
			FileName: "简历7.pdf",
			FileType: "pdf",
			FileSize: int64(len(content)),
			Tags:     []string{},
		},
		// The archive entry lost the "<id>-" prefix, as happens when a backup
		// is unpacked and compressed again.
		ArchivePath: "resumes/5-简历7.pdf",
	}, gbkName, "resumes/简历7.pdf", content)

	result, err := app.server.ImportBackup(target)
	if err != nil {
		t.Fatalf("import backup: %v", err)
	}
	if result.Counts.Resumes != 1 {
		t.Fatalf("imported resumes = %d, want 1", result.Counts.Resumes)
	}

	var resumes []models.Resume
	if err := app.server.database().Find(&resumes).Error; err != nil {
		t.Fatalf("load imported resumes: %v", err)
	}
	if len(resumes) != 1 {
		t.Fatalf("resume count = %d, want 1", len(resumes))
	}
	if resumes[0].FileName != "简历7.pdf" {
		t.Fatalf("file name = %q, want %q", resumes[0].FileName, "简历7.pdf")
	}
	if !resumes[0].Encrypted {
		t.Fatal("imported resume should be encrypted")
	}
	imported, err := app.server.readResumeFile(resumes[0])
	if err != nil {
		t.Fatalf("read imported resume: %v", err)
	}
	if string(imported) != string(content) {
		t.Fatalf("imported content = %q, want %q", imported, content)
	}
}

func TestImportBackupMatchesResumeByFileName(t *testing.T) {
	content := []byte("resume body")

	app := newTestApp(t)
	target := filepath.Join(t.TempDir(), "renamed.zip")
	writeBackupZip(t, target, backupResume{
		Resume: models.Resume{
			ID:       9,
			Name:     "Plain",
			FileName: "plain.md",
			FileType: "md",
			FileSize: int64(len(content)),
			Tags:     []string{},
		},
		ArchivePath: "resumes/9-plain.md",
	}, "resumes/plain.md", "", content)

	if _, err := app.server.ImportBackup(target); err != nil {
		t.Fatalf("import backup: %v", err)
	}

	var resumes []models.Resume
	if err := app.server.database().Find(&resumes).Error; err != nil {
		t.Fatalf("load imported resumes: %v", err)
	}
	if len(resumes) != 1 {
		t.Fatalf("resume count = %d, want 1", len(resumes))
	}
	imported, err := app.server.readResumeFile(resumes[0])
	if err != nil {
		t.Fatalf("read imported resume: %v", err)
	}
	if string(imported) != string(content) {
		t.Fatalf("imported content = %q, want %q", imported, content)
	}
}

func TestUnicodePathNameIgnoresStaleChecksum(t *testing.T) {
	rawName := "resumes/\xbc\xf2\xc0\xfa7.pdf"
	extra := make([]byte, 4+1+4+len("resumes/简历7.pdf"))
	binary.LittleEndian.PutUint16(extra[0:2], unicodePathExtraID)
	binary.LittleEndian.PutUint16(extra[2:4], uint16(len(extra)-4))
	extra[4] = 1
	binary.LittleEndian.PutUint32(extra[5:9], crc32.ChecksumIEEE([]byte("different name")))
	copy(extra[9:], "resumes/简历7.pdf")

	if name, ok := unicodePathName(extra, rawName); ok {
		t.Fatalf("stale extra field accepted as %q", name)
	}
}
