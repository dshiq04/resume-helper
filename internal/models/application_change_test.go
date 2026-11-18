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
		{StatusPending, []string{StatusScreening}},
		{StatusScreening, []string{StatusScreening}},
		{StatusWritten, []string{StatusScreening, StatusWritten}},
		{StatusInterview, []string{StatusScreening, StatusWritten, StatusInterview}},
		{StatusOffer, []string{StatusScreening, StatusWritten, StatusInterview, StatusOffer}},
		{StatusRejected, []string{StatusScreening, StatusWritten, StatusInterview, StatusRejected}},
	}

	for _, test := range tests {
		if got := StageChainForStatus(test.status); !reflect.DeepEqual(got, test.want) {
			t.Errorf("StageChainForStatus(%q) = %v, want %v", test.status, got, test.want)
		}
	}
}
