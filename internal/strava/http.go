package strava

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const DefaultBaseURL = "https://www.strava.com/api/v3"

// APIError is a non-2xx response from the Strava API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("strava api: %d %s", e.StatusCode, e.Message)
}

// HTTPClient implements Client over the Strava REST API. The underlying
// http.Client must add authorization (see NewTokenSource).
type HTTPClient struct {
	http    *http.Client
	baseURL string
}

func NewHTTPClient(hc *http.Client, baseURL string) *HTTPClient {
	return &HTTPClient{http: hc, baseURL: strings.TrimRight(baseURL, "/")}
}

type activityJSON struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	SportType string    `json:"sport_type"`
	Distance  float64   `json:"distance"`
	StartDate time.Time `json:"start_date"`
}

type lapJSON struct {
	LapIndex   int     `json:"lap_index"`
	Distance   float64 `json:"distance"`
	MovingTime int     `json:"moving_time"`
}

func (c *HTTPClient) GetActivity(ctx context.Context, id int64) (Activity, error) {
	var a activityJSON
	if err := c.get(ctx, fmt.Sprintf("/activities/%d", id), &a); err != nil {
		return Activity{}, fmt.Errorf("get activity %d: %w", id, err)
	}
	return Activity{
		ID:        a.ID,
		Name:      a.Name,
		SportType: a.SportType,
		DistanceM: a.Distance,
		StartDate: a.StartDate,
	}, nil
}

func (c *HTTPClient) GetLaps(ctx context.Context, id int64) ([]Lap, error) {
	var raw []lapJSON
	if err := c.get(ctx, fmt.Sprintf("/activities/%d/laps", id), &raw); err != nil {
		return nil, fmt.Errorf("get laps of activity %d: %w", id, err)
	}
	laps := make([]Lap, len(raw))
	for i, l := range raw {
		laps[i] = Lap{
			Index:      l.LapIndex,
			DistanceM:  l.Distance,
			MovingTime: time.Duration(l.MovingTime) * time.Second,
		}
	}
	return laps, nil
}

func (c *HTTPClient) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return readAPIError(resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func readAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	var e struct {
		Message string `json:"message"`
		Errors  []struct {
			Resource string `json:"resource"`
			Field    string `json:"field"`
			Code     string `json:"code"`
		} `json:"errors"`
	}
	msg := strings.TrimSpace(string(body))
	if json.Unmarshal(body, &e) == nil && e.Message != "" {
		msg = e.Message
		// Details like "Application Status: Inactive" explain most 4xx responses.
		for _, d := range e.Errors {
			msg += fmt.Sprintf(" (%s %s: %s)", d.Resource, d.Field, d.Code)
		}
	}
	if msg == "" {
		msg = http.StatusText(resp.StatusCode)
	}
	return &APIError{StatusCode: resp.StatusCode, Message: msg}
}
