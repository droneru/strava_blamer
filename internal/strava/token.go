package strava

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

const DefaultTokenURL = "https://www.strava.com/oauth/token"

type AuthConfig struct {
	ClientID     string
	ClientSecret string
	TokenURL     string // DefaultTokenURL if empty
}

// NewTokenSource returns a token source backed by the token file at path.
//
// If the file does not exist, it is created from initialRefreshToken.
// Strava rotates the refresh token on every refresh and invalidates the old
// one, so every new token is written to the file before it is used.
func NewTokenSource(ctx context.Context, cfg AuthConfig, path, initialRefreshToken string) (oauth2.TokenSource, error) {
	tok, err := loadToken(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if initialRefreshToken == "" {
			return nil, fmt.Errorf("token file %s not found and STRAVA_REFRESH_TOKEN is not set", path)
		}
		tok = &oauth2.Token{RefreshToken: initialRefreshToken}
		if err := saveToken(path, tok); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	}

	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = DefaultTokenURL
	}
	oc := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL:  tokenURL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
	return &persistingSource{src: oc.TokenSource(ctx, tok), path: path, last: tok}, nil
}

type persistingSource struct {
	src  oauth2.TokenSource
	path string

	mu   sync.Mutex
	last *oauth2.Token
}

func (s *persistingSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tok, err := s.src.Token()
	if err != nil {
		return nil, err
	}
	if tok.AccessToken != s.last.AccessToken || tok.RefreshToken != s.last.RefreshToken {
		if err := saveToken(s.path, tok); err != nil {
			return nil, err
		}
		s.last = tok
	}
	return tok, nil
}

type tokenFile struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token"`
	Expiry       int64  `json:"expires_at,omitempty"`
}

func loadToken(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read token file: %w", err)
	}
	var f tokenFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse token file %s: %w", path, err)
	}
	if f.RefreshToken == "" {
		return nil, fmt.Errorf("token file %s has no refresh_token", path)
	}
	tok := &oauth2.Token{AccessToken: f.AccessToken, RefreshToken: f.RefreshToken}
	if f.Expiry > 0 {
		tok.Expiry = time.Unix(f.Expiry, 0)
	}
	return tok, nil
}

// saveToken writes the token atomically so a crash never leaves
// a truncated file with the only valid refresh token.
func saveToken(path string, tok *oauth2.Token) error {
	f := tokenFile{AccessToken: tok.AccessToken, RefreshToken: tok.RefreshToken}
	if !tok.Expiry.IsZero() {
		f.Expiry = tok.Expiry.Unix()
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".token-*.json")
	if err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("save token: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("save token: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	return nil
}
