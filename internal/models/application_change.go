package models

import "time"

// ApplicationChange is one recorded status milestone of an application. A
// record is appended whenever an application reaches a status it has never
// reached before, which keeps the timeline stable when users toggle back and
// forth between the same two statuses.
type ApplicationChange struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ApplicationID uint      `gorm:"not null;uniqueIndex:idx_application_change_status" json:"applicationId"`
	Status        string    `gorm:"type:varchar(32);not null;uniqueIndex:idx_application_change_status" json:"status"`
	ChangedAt     time.Time `gorm:"not null;index" json:"changedAt"`
	CreatedAt     time.Time `json:"createdAt"`
}

// StageOrder is the linear progression used by the conversion funnel and by
// the "missing history" prompt for records created before history tracking.
var StageOrder = []string{StatusScreening, StatusWritten, StatusInterview, StatusOffer}

// StageChainForStatus returns the statuses that must have happened before (and
// including) the given status. Side states such as pending and rejected only
// imply that screening happened, because they can occur at any stage.
func StageChainForStatus(status string) []string {
	for index, stage := range StageOrder {
		if stage == status {
			return append([]string{}, StageOrder[:index+1]...)
		}
	}
	if status == StatusPending || status == StatusRejected {
		return []string{StatusScreening}
	}
	return []string{StatusScreening}
}

// StatusReached reports whether an application whose current status is
// current ever reached the given stage. It is the fallback used for records
// created before status history existed.
func StatusReached(current, stage string) bool {
	for _, item := range StageChainForStatus(current) {
		if item == stage {
			return true
		}
	}
	return false
}
