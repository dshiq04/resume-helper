package models

import "time"

// Resume is one stored resume file. The binary lives in the portable
// "<data dir>/resume" folder; this record keeps the metadata and the randomly
// generated on-disk name.
type Resume struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"type:varchar(255);not null;index" json:"name"`
	Tags       []string  `gorm:"serializer:json;type:text" json:"tags"`
	FileName   string    `gorm:"type:varchar(255);not null" json:"fileName"`
	StoredName string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"-"`
	FileType   string    `gorm:"type:varchar(16);not null" json:"fileType"`
	FileSize   int64     `gorm:"not null;default:0" json:"fileSize"`
	Pinned     bool      `gorm:"not null;default:false;index" json:"pinned"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
