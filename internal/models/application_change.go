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

// StageOrder is the linear progression used by the conversion funnel.
var StageOrder = []string{StatusScreening, StatusWritten, StatusInterview, StatusOffer}

// StageChainForStatus returns the statuses that must have happened before (and
// including) the given status. The chain follows the timeline order
// screening → written → interview and then branches into two parallel terminal
// stages: offer or rejected. A record that reached one terminal stage never
// requires the other one, which keeps the "complete the history" form focused
// on the outcome that actually happened. Pending is a side state that can
// occur at any point, so it only implies that screening happened.
func StageChainForStatus(status string) []string {
	if status == StatusRejected {
		return []string{StatusScreening, StatusWritten, StatusInterview, StatusRejected}
	}
	for index, stage := range StageOrder {
		if stage == status {
			return append([]string{}, StageOrder[:index+1]...)
		}
	}
	if status == StatusPending {
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
