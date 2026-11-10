package models

import "time"

// Setting is a small persisted key/value store for application preferences
// such as the bcrypt password hash.
type Setting struct {
	Key       string    `gorm:"primaryKey;type:varchar(64)" json:"key"`
	Value     string    `gorm:"type:text" json:"-"`
	UpdatedAt time.Time `json:"updatedAt"`
}
