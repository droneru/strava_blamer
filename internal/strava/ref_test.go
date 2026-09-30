package strava

import "testing"

func TestParseActivityRef(t *testing.T) {
	tests := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{in: "15234567890", want: 15234567890},
		{in: " 42 ", want: 42},
		{in: "https://www.strava.com/activities/15234567890", want: 15234567890},
		{in: "https://strava.com/activities/15234567890/", want: 15234567890},
		{in: "http://www.strava.com/activities/1/overview", want: 1},
		{in: "https://www.strava.com/activities/7?share_sig=abc", want: 7},
		{in: "www.strava.com/activities/7", want: 7},
		{in: "", wantErr: true},
		{in: "0", wantErr: true},
		{in: "-5", wantErr: true},
		{in: "abc", wantErr: true},
		{in: "https://www.strava.com/athletes/123", wantErr: true},
		{in: "https://www.strava.com/activities/", wantErr: true},
		{in: "https://www.strava.com/activities/abc", wantErr: true},
		{in: "https://example.com/activities/123", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseActivityRef(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("got %d, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}
