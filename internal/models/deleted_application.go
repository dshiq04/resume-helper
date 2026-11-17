package models

import "time"

// DeletedApplication is the recycle-bin copy of a deleted application. The
// record keeps every column of the original row (including its id and
// timestamps) so restoring it recreates the row exactly, together with the
// status timeline that belonged to it.
type DeletedApplication struct {
	ID          uint                `gorm:"primaryKey" json:"id"`
	OriginalID  uint                `gorm:"not null;index" json:"originalId"`
	Category    string              `gorm:"type:varchar(32);not null" json:"category"`
	CompanyName string              `gorm:"type:varchar(255);not null" json:"companyName"`
	Link        string              `gorm:"type:text" json:"link"`
	Status      string              `gorm:"type:varchar(32);not null" json:"status"`
	Pinned      bool                `gorm:"not null;default:false" json:"pinned"`
	AppliedAt   time.Time           `json:"appliedAt"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
	History     []ApplicationChange `gorm:"serializer:json;type:text" json:"history"`
	DeletedAt   time.Time           `gorm:"not null;index" json:"deletedAt"`
}
