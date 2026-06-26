//go:build microsoft || microsoft_email

// Package common is a CONCERN-AGNOSTIC Azure-AD app-token primitive.
//
// Charter / delineation principle:
//
//   - This package MUST NOT hardcode a concern prefix. There is no
//     concern-specific env-var literal anywhere in here. Callers inject their
//     own {CONCERN}_{PROVIDER}_ prefix via FromEnv(prefix).
//   - This package MUST NOT hold package-global token state. There is no
//     sync.Once singleton and no package-level cached token. Each *Client
//     instance caches its OWN token + expiry behind its own mutex, so two
//     Microsoft concerns can hold DIFFERENT Azure apps simultaneously in one
//     process.
//
// Concrete concerns and the prefixes they inject:
//
//   - EMAIL (Graph):    common.NewClient(common.FromEnv("EMAIL_MICROSOFT_"))
//     lives at internal/email/, reads EMAIL_MICROSOFT_{TENANT_ID,CLIENT_ID,
//     CLIENT_SECRET,...}.
//   - STORAGE (SharePoint, future): common.NewClient(common.FromEnv("STORAGE_SHAREPOINT_"))
//     will live at internal/storage/sharepoint/, reads STORAGE_SHAREPOINT_{TENANT_ID,
//     CLIENT_ID,CLIENT_SECRET,SITE_URL,...}. It is an Azure app INDEPENDENT of
//     EMAIL_MICROSOFT_ — a different *Client instance with its own credentials
//     and its own token cache. The adapter does not exist yet; this note records
//     the shape so the delineation stays explicit. See SHAREPOINT.md in the
//     module root.
package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Config holds the Azure-AD app (client_credentials) configuration for a single
// concern. Populate it via FromEnv with a caller-supplied prefix, or construct
// it directly.
type Config struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	Timeout      time.Duration
}

// FromEnv builds a Config by reading {prefix}TENANT_ID, {prefix}CLIENT_ID,
// {prefix}CLIENT_SECRET, and {prefix}TIMEOUT from the environment. The prefix is
// INJECTED by the caller — e.g. FromEnv("EMAIL_MICROSOFT_") or
// FromEnv("STORAGE_SHAREPOINT_"). No concern prefix is hardcoded here.
func FromEnv(prefix string) Config {
	timeout := 30 * time.Second
	if timeoutStr := os.Getenv(prefix + "TIMEOUT"); timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}

	return Config{
		TenantID:     os.Getenv(prefix + "TENANT_ID"),
		ClientID:     os.Getenv(prefix + "CLIENT_ID"),
		ClientSecret: os.Getenv(prefix + "CLIENT_SECRET"),
		Timeout:      timeout,
	}
}

// tokenResponse represents the OAuth2 token response from Microsoft
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// Client is a per-instance Azure-AD app-token primitive. It owns its Config,
// its HTTP client, and its OWN cached token + expiry behind its own mutex.
// Construct one per concern via NewClient — there is no package-global singleton.
type Client struct {
	config     Config
	httpClient *http.Client

	mu          sync.Mutex
	cachedToken string
	tokenExpiry time.Time
}

// NewClient constructs a per-instance Azure-AD token client for one concern.
// The returned *Client holds its own credentials and token cache, so multiple
// concerns (e.g. EMAIL Graph and a future SharePoint STORAGE) can each hold a
// DIFFERENT Azure app in the same process.
func NewClient(config Config) (*Client, error) {
	if config.TenantID == "" || config.ClientID == "" || config.ClientSecret == "" {
		return nil, errors.New("tenant_id, client_id, and client_secret are required for Microsoft Graph")
	}

	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}

	log.Println("✅ Microsoft Graph client initialized successfully")

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}, nil
}

// getToken fetches a new access token using the Client Credentials flow, caching
// it per-instance. This is the simple approach - just POST to get token, like
// Google Apps Script.
func (c *Client) getToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return a still-valid cached token.
	if c.cachedToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.cachedToken, nil
	}

	log.Println("🔄 Fetching Microsoft access token (client_credentials)...")

	// Build token request - exactly like the Google Apps Script version
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.config.TenantID)

	data := url.Values{}
	data.Set("client_id", c.config.ClientID)
	data.Set("grant_type", "client_credentials")
	data.Set("scope", "https://graph.microsoft.com/.default")
	data.Set("client_secret", c.config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errBody)
		return "", fmt.Errorf("token request failed with status %d: %v", resp.StatusCode, errBody)
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	// Cache the token with expiry (subtract 60 seconds for safety margin)
	c.cachedToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	log.Printf("✅ Microsoft token fetched successfully (expires in %d seconds)", tokenResp.ExpiresIn)

	return c.cachedToken, nil
}

// GetAuthenticatedClient returns an HTTP client that adds this instance's auth
// header to requests.
func (c *Client) GetAuthenticatedClient(ctx context.Context) (*http.Client, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}

	// Return a client that wraps the base client with auth header
	return &http.Client{
		Timeout: c.config.Timeout,
		Transport: &authTransport{
			token: token,
			base:  http.DefaultTransport,
		},
	}, nil
}

// authTransport adds Authorization header to all requests
type authTransport struct {
	token string
	base  http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req)
}

// TestConnection tests the Microsoft Graph connection for this instance.
// For client_credentials flow, we just verify we can get a token - no API call needed.
func (c *Client) TestConnection(ctx context.Context) error {
	// Simply verify we can get a token - this proves the credentials work
	_, err := c.getToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	log.Println("✅ Microsoft Graph connection test successful (token verified)")
	return nil
}

// Close clears this instance's cached token.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cachedToken = ""
	c.tokenExpiry = time.Time{}
	log.Println("✅ Microsoft Graph client closed successfully")
	return nil
}
