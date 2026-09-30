package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/droneru/strava_blamer/internal/strava"
)

type fakeClient struct {
	activities map[int64]strava.Activity
	laps       map[int64][]strava.Lap
	lapCalls   int
}

func (f *fakeClient) GetActivity(_ context.Context, id int64) (strava.Activity, error) {
	a, ok := f.activities[id]
	if !ok {
		return strava.Activity{}, &strava.APIError{StatusCode: 404, Message: "Record Not Found"}
	}
	return a, nil
}

func (f *fakeClient) GetLaps(_ context.Context, id int64) ([]strava.Lap, error) {
	f.lapCalls++
	return f.laps[id], nil
}

func run(t *testing.T, client strava.Client, args ...string) (string, error) {
	t.Helper()
	factory := func(context.Context, string) (strava.Client, error) { return client, nil }
	root := newRootCmd(factory)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func TestGet(t *testing.T) {
	start := time.Date(2026, 9, 1, 5, 30, 0, 0, time.UTC)
	client := &fakeClient{
		activities: map[int64]strava.Activity{
			1: {ID: 1, Name: "Morning Run", SportType: "Run", DistanceM: 16500, StartDate: start},
			2: {ID: 2, Name: "Утренний заплыв", SportType: "Swim", DistanceM: 1200, StartDate: start},
			3: {ID: 3, Name: "Open water", SportType: "Swim", DistanceM: 1500, StartDate: start},
		},
		laps: map[int64][]strava.Lap{
			2: {
				{Index: 1, DistanceM: 400, MovingTime: 7 * time.Minute},
				{Index: 2, DistanceM: 0},
				{Index: 3, DistanceM: 800, MovingTime: 15 * time.Minute},
			},
		},
	}

	tests := []struct {
		name     string
		arg      string
		want     []string
		dontWant []string
	}{
		{
			name:     "run by id",
			arg:      "1",
			want:     []string{"ID:        1", "Название:  Morning Run", "Тип:       Run", "Дистанция: 16500 m", "2026-09-01T05:30:00Z"},
			dontWant: []string{"Laps"},
		},
		{
			name: "swim by url with laps",
			arg:  "https://www.strava.com/activities/2",
			want: []string{"Название:  Утренний заплыв", "Тип:       Swim", "Laps:", "1     400 m  7m0s", "2       0 m", "3     800 m  15m0s"},
		},
		{
			name: "swim without laps",
			arg:  "3",
			want: []string{"Название:  Open water", "Laps:      нет"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := run(t, client, "get", tt.arg)
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range tt.want {
				if !strings.Contains(out, s) {
					t.Errorf("output missing %q:\n%s", s, out)
				}
			}
			for _, s := range tt.dontWant {
				if strings.Contains(out, s) {
					t.Errorf("output contains %q:\n%s", s, out)
				}
			}
		})
	}
}

func TestGetDoesNotFetchLapsForRun(t *testing.T) {
	client := &fakeClient{activities: map[int64]strava.Activity{1: {ID: 1, SportType: "Run"}}}
	if _, err := run(t, client, "get", "1"); err != nil {
		t.Fatal(err)
	}
	if client.lapCalls != 0 {
		t.Fatalf("GetLaps called %d times for a run", client.lapCalls)
	}
}

func TestGetErrors(t *testing.T) {
	client := &fakeClient{}

	if _, err := run(t, client, "get", "not-an-id"); err == nil {
		t.Fatal("invalid ref: want error")
	}

	_, err := run(t, client, "get", "404")
	var apiErr *strava.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 404 {
		t.Fatalf("missing activity: got %v, want 404 APIError", err)
	}

	factoryErr := errors.New("no credentials")
	root := newRootCmd(func(context.Context, string) (strava.Client, error) { return nil, factoryErr })
	root.SetArgs([]string{"get", "1"})
	if err := root.Execute(); !errors.Is(err, factoryErr) {
		t.Fatalf("client factory error: got %v", err)
	}
}

func TestCommandsNotImplemented(t *testing.T) {
	tests := [][]string{
		{"blame", "123"},
		{"sweep"},
		{"watch"},
	}
	for _, args := range tests {
		t.Run(args[0], func(t *testing.T) {
			_, err := run(t, &fakeClient{}, args...)
			if !errors.Is(err, errNotImplemented) {
				t.Fatalf("got %v, want %v", err, errNotImplemented)
			}
		})
	}
}
