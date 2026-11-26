package api

import "testing"

func TestScoreEnabled(t *testing.T) {
	for _, tc := range []struct {
		scope    string
		campaign int64
		want     bool
	}{
		{"", 6, false},
		{"6", 6, true},
		{"6", 7, false},
		{"all", 6, true},
		{"all", 27, true},
		{"all", 0, false},
	} {
		t.Setenv("INK_SCORE_CAMPAIGN_ID", tc.scope)
		if got := scoreEnabled(tc.campaign); got != tc.want {
			t.Errorf("scope %q campaign %d: got %t, want %t", tc.scope, tc.campaign, got, tc.want)
		}
	}
}
