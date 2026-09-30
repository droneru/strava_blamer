package strava

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// fakeTokenServer mimics Strava: each refresh returns a new refresh token
// and only accepts the latest one.
type fakeTokenServer struct {
	t       *testing.T
	current string
	calls   int
}

func (f *fakeTokenServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.calls++
	if err := r.ParseForm(); err != nil {
		f.t.Fatal(err)
	}
	if got := r.PostForm.Get("client_id"); got != "id" {
		f.t.Errorf("client_id = %q", got)
	}
	if got := r.PostForm.Get("client_secret"); got != "secret" {
		f.t.Errorf("client_secret = %q", got)
	}
	if got := r.PostForm.Get("refresh_token"); got != f.current {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"Bad Request","errors":[{"field":"refresh_token","code":"invalid"}]}`))
		return
	}
	f.current = fmt.Sprintf("refresh-%d", f.calls)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token_type":    "Bearer",
		"access_token":  fmt.Sprintf("access-%d", f.calls),
		"refresh_token": f.current,
		"expires_in":    21600,
		"expires_at":    time.Now().Add(6 * time.Hour).Unix(),
	})
}

func newTokenSource(t *testing.T, srv *httptest.Server, path, initial string) *persistingSource {
	t.Helper()
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, srv.Client())
	ts, err := NewTokenSource(ctx, AuthConfig{ClientID: "id", ClientSecret: "secret", TokenURL: srv.URL}, path, initial)
	if err != nil {
		t.Fatal(err)
	}
	return ts.(*persistingSource)
}

func readTokenFile(t *testing.T, path string) tokenFile {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f tokenFile
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestTokenSourceBootstrapsFromEnvAndPersistsRotation(t *testing.T) {
	fake := &fakeTokenServer{t: t, current: "initial"}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "token.json")

	ts := newTokenSource(t, srv, path, "initial")
	if got := readTokenFile(t, path).RefreshToken; got != "initial" {
		t.Fatalf("file created with refresh token %q, want %q", got, "initial")
	}

	tok, err := ts.Token()
	if err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken != "access-1" {
		t.Fatalf("access token = %q", tok.AccessToken)
	}
	f := readTokenFile(t, path)
	if f.RefreshToken != "refresh-1" || f.AccessToken != "access-1" || f.Expiry == 0 {
		t.Fatalf("token file not updated after refresh: %+v", f)
	}

	// A valid access token is reused without hitting the server.
	if _, err := ts.Token(); err != nil {
		t.Fatal(err)
	}
	if fake.calls != 1 {
		t.Fatalf("token endpoint called %d times, want 1", fake.calls)
	}
}

func TestTokenSourcePrefersFileOverEnv(t *testing.T) {
	fake := &fakeTokenServer{t: t, current: "from-file"}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "token.json")
	if err := os.WriteFile(path, []byte(`{"refresh_token":"from-file"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	ts := newTokenSource(t, srv, path, "stale-env-token")
	if _, err := ts.Token(); err != nil {
		t.Fatal(err)
	}
	if got := readTokenFile(t, path).RefreshToken; got != "refresh-1" {
		t.Fatalf("refresh token in file = %q, want refresh-1", got)
	}
}

func TestTokenSourceReusesUnexpiredAccessTokenFromFile(t *testing.T) {
	fake := &fakeTokenServer{t: t, current: "r"}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "token.json")
	exp := time.Now().Add(time.Hour).Unix()
	body := fmt.Sprintf(`{"access_token":"cached","refresh_token":"r","expires_at":%d}`, exp)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	tok, err := newTokenSource(t, srv, path, "").Token()
	if err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken != "cached" || fake.calls != 0 {
		t.Fatalf("got access token %q after %d refreshes, want cached token and no refresh", tok.AccessToken, fake.calls)
	}
}

func TestNewTokenSourceErrors(t *testing.T) {
	dir := t.TempDir()
	cfg := AuthConfig{ClientID: "id", ClientSecret: "secret"}

	_, err := NewTokenSource(context.Background(), cfg, filepath.Join(dir, "missing.json"), "")
	if err == nil || !strings.Contains(err.Error(), "STRAVA_REFRESH_TOKEN") {
		t.Fatalf("missing file and env: got %v", err)
	}

	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte(`{"access_token":"x"}`), 0o600)
	if _, err := NewTokenSource(context.Background(), cfg, bad, "env"); err == nil {
		t.Fatal("file without refresh_token: want error")
	}
}
