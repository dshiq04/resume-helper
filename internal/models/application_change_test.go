package models

import (
	"reflect"
	"testing"
)

func TestStageChainForStatus(t *testing.T) {
	tests := []struct {
		status string
		want   []string
	}{
		{StatusPending, []string{StatusPending}},
		{StatusScreening, []string{StatusPending, StatusScreening}},
		{StatusWritten, []string{StatusPending, StatusScreening, StatusWritten}},
		{StatusInterview, []string{StatusPending, StatusScreening, StatusWritten, StatusInterview}},
		{StatusOffer, []string{StatusPending, StatusScreening, StatusWritten, StatusInterview, StatusOffer}},
		{StatusRejected, []string{StatusPending, StatusScreening, StatusRejected}},
	}

	for _, test := range tests {
		if got := StageChainForStatus(test.status); !reflect.DeepEqual(got, test.want) {
			t.Errorf("StageChainForStatus(%q) = %v, want %v", test.status, got, test.want)
		}
	}
}
