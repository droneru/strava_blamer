package strava

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPClientGetActivity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activities/42" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Write([]byte(`{"id":42,"name":"Morning Swim","sport_type":"Swim","type":"Swim",
			"distance":2400.5,"start_date":"2026-09-01T05:30:00Z","moving_time":3000}`))
	}))
	defer srv.Close()

	got, err := NewHTTPClient(srv.Client(), srv.URL).GetActivity(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	want := Activity{
		ID:        42,
		Name:      "Morning Swim",
		SportType: "Swim",
		DistanceM: 2400.5,
		StartDate: time.Date(2026, 9, 1, 5, 30, 0, 0, time.UTC),
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestHTTPClientGetLaps(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activities/42/laps" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Write([]byte(`[
			{"lap_index":1,"distance":400,"moving_time":420},
			{"lap_index":2,"distance":0,"moving_time":0},
			{"lap_index":3,"distance":800,"moving_time":900}]`))
	}))
	defer srv.Close()

	got, err := NewHTTPClient(srv.Client(), srv.URL).GetLaps(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	want := []Lap{
		{Index: 1, DistanceM: 400, MovingTime: 7 * time.Minute},
		{Index: 2},
		{Index: 3, DistanceM: 800, MovingTime: 15 * time.Minute},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d laps, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("lap %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestHTTPClientAPIError(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantMsg string
	}{
		{"strava json", 404, `{"message":"Record Not Found","errors":[]}`, "Record Not Found"},
		{"strava json with details", 403,
			`{"message":"Forbidden","errors":[{"resource":"Application","field":"Status","code":"Inactive"}]}`,
			"Forbidden (Application Status: Inactive)"},
		{"plain text", 429, `Rate Limit Exceeded`, "Rate Limit Exceeded"},
		{"empty body", 401, ``, "Unauthorized"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			_, err := NewHTTPClient(srv.Client(), srv.URL).GetActivity(context.Background(), 1)
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("got %v, want *APIError", err)
			}
			if apiErr.StatusCode != tt.status || apiErr.Message != tt.wantMsg {
				t.Fatalf("got %d %q, want %d %q", apiErr.StatusCode, apiErr.Message, tt.status, tt.wantMsg)
			}
		})
	}
}
