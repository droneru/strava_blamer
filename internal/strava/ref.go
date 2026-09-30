package strava

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParseActivityRef accepts an activity id or a strava.com activity URL
// and returns the activity id.
func ParseActivityRef(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if id, err := parseID(s); err == nil {
		return id, nil
	}

	raw := s
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid activity reference %q: %w", s, err)
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if host != "strava.com" || len(parts) < 2 || parts[0] != "activities" {
		return 0, fmt.Errorf("invalid activity reference %q: want id or https://www.strava.com/activities/<id>", s)
	}
	id, err := parseID(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid activity reference %q: %w", s, err)
	}
	return id, nil
}

func parseID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid activity id %q", s)
	}
	return id, nil
}
