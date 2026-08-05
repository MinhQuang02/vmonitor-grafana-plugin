package vmonitor

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	DefaultTokenURL   = "https://iamapis.vngcloud.vn/accounts-api/v2/auth/token"
	expirySkew        = 30 * time.Second
	fallbackTokenTTL  = 1 * time.Minute
	tokenFetchTimeout = 30 * time.Second
)

type TokenSource struct {
	tokenURL     string
	clientID     string
	clientSecret string
	http         *http.Client

	mu     sync.Mutex
	token  string
	expiry time.Time
}

func NewTokenSource(tokenURL, clientID, clientSecret string) *TokenSource {
	if tokenURL == "" {
		tokenURL = DefaultTokenURL
	}
	return &TokenSource{
		tokenURL:     tokenURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		http:         &http.Client{Timeout: tokenFetchTimeout},
	}
}

func (ts *TokenSource) Token(ctx context.Context) (string, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.token != "" && time.Now().Before(ts.expiry) {
		return ts.token, nil
	}
	if err := ts.fetch(ctx); err != nil {
		return "", err
	}
	return ts.token, nil
}

func (ts *TokenSource) Invalidate(failed string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.token == failed {
		ts.token = ""
		ts.expiry = time.Time{}
	}
}

func (ts *TokenSource) fetch(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), tokenFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.tokenURL,
		strings.NewReader(`{"grant_type":"client_credentials"}`))
	if err != nil {
		return fmt.Errorf("build token request: %w", err)
	}
	basic := base64.StdEncoding.EncodeToString([]byte(ts.clientID + ":" + ts.clientSecret))
	req.Header.Set("Authorization", "Basic "+basic)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := ts.http.Do(req)
	if err != nil {
		return fmt.Errorf("request IAM token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseAPIError(resp.StatusCode, body)
	}

	var tr struct {
		AccessToken  string `json:"access_token"`
		AccessTokenC string `json:"accessToken"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return fmt.Errorf("decode token response: %w", err)
	}
	token := tr.AccessToken
	if token == "" {
		token = tr.AccessTokenC
	}
	if token == "" {
		return fmt.Errorf("token response contained no access token")
	}

	ts.token = token
	ts.expiry = tokenExpiry(token, tr.ExpiresIn)
	return nil
}

func tokenExpiry(jwt string, expiresIn int64) time.Time {
	now := time.Now()
	if expiresIn > 0 {
		return now.Add(lifetimeWithSkew(time.Duration(expiresIn) * time.Second))
	}
	if exp, ok := jwtExpiry(jwt); ok {
		return exp.Add(-expirySkew)
	}
	return now.Add(fallbackTokenTTL)
}

func lifetimeWithSkew(lifetime time.Duration) time.Duration {
	if lifetime <= expirySkew {
		return lifetime / 2
	}
	return lifetime - expirySkew
}

func jwtExpiry(jwt string) (time.Time, bool) {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp <= 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.Exp, 0), true
}
