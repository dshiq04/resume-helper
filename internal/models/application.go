package models

import (
	"strings"
	"time"
)

const (
	CategorySpring     = "spring"
	CategoryAutumn     = "autumn"
	CategoryInternship = "internship"
	CategorySocial     = "social"

	StatusScreening = "screening"
	StatusWritten   = "written"
	StatusInterview = "interview"
	StatusPending   = "pending"
	StatusOffer     = "offer"
	StatusRejected  = "rejected"
)

var (
	Categories = []string{CategorySpring, CategoryAutumn, CategoryInternship, CategorySocial}
	Statuses   = []string{StatusScreening, StatusWritten, StatusInterview, StatusPending, StatusOffer, StatusRejected}
)

// Application is one resume submission.
type Application struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Category    string    `gorm:"type:varchar(32);not null;index" json:"category"`
	CompanyName string    `gorm:"type:varchar(255);not null;index" json:"companyName"`
	Link        string    `gorm:"type:text" json:"link"`
	Status      string    `gorm:"type:varchar(32);not null;default:screening;index" json:"status"`
	Pinned      bool      `gorm:"not null;default:false;index" json:"pinned"`
	AppliedAt   time.Time `gorm:"index" json:"appliedAt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func IsValidCategory(value string) bool {
	return contains(Categories, value)
}

// DefaultCategoryForTime picks the recruiting season that matches the server
// clock: autumn hiring runs August through November, spring hiring February
// through May, and the remaining summer/winter months default to internships.
func DefaultCategoryForTime(t time.Time) string {
	switch month := t.Month(); {
	case month >= time.August && month <= time.November:
		return CategoryAutumn
	case month >= time.February && month <= time.May:
		return CategorySpring
	default:
		return CategoryInternship
	}
}

func IsValidStatus(value string) bool {
	return contains(Statuses, value)
}

func NormalizeCategory(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func NormalizeStatus(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
