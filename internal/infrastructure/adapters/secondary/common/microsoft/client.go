//go:build microsoft

package microsoft

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

var (
	httpClient      *http.Client
	microsoftConfig *MicrosoftConfig
	cachedToken     string
	tokenExpiry     time.Time
	onceMicrosoft   sync.Once
	tokenMutex      sync.RWMutex
)

// MicrosoftConfig holds Microsoft common configuration
type MicrosoftConfig struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	Timeout      time.Duration
}

// tokenResponse represents the OAuth2 token response from Microsoft
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// DefaultMicrosoftConfig returns default Microsoft configuration from environment
func DefaultMicrosoftConfig() *MicrosoftConfig {
	timeout := 30 * time.Second
	if timeoutStr := os.Getenv("LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_TIMEOUT"); timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}

	return &MicrosoftConfig{
		TenantID:     os.Getenv("LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_TENANT_ID"),
		ClientID:     os.Getenv("LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_CLIENT_ID"),
		ClientSecret: os.Getenv("LEAPFOR_INTEGRATION_EMAIL_MICROSOFT_CLIENT_SECRET"),
		Timeout:      timeout,
	}
}

// InitializeMicrosoftClient initializes the Microsoft Graph client singleton
func InitializeMicrosoftClient(ctx context.Context, config *MicrosoftConfig) error {
	var initErr error

	onceMicrosoft.Do(func() {
		if config == nil {
			config = DefaultMicrosoftConfig()
		}

		// Validate required configuration
		if config.TenantID == "" || config.ClientID == "" || config.ClientSecret == "" {
			initErr = errors.New("tenant_id, client_id, and client_secret are required for Microsoft Graph")
			return
		}

		// Store config for token fetching
		microsoftConfig = config

		// Create HTTP client with timeout
		httpClient = &http.Client{
			Timeout: config.Timeout,
		}

		log.Println("✅ Microsoft Graph client initialized successfully")
	})

	return initErr
}

// getToken fetches a new access token using Client Credentials flow
// This is the simple approach - just POST to get token, like Google Apps Script
func getToken(ctx context.Context) (string, error) {
	if microsoftConfig == nil {
		return "", errors.New("Microsoft client not initialized")
	}

	// Check if we have a valid cached token
	tokenMutex.RLock()
	if cachedToken != "" && time.Now().Before(tokenExpiry) {
		token := cachedToken
		tokenMutex.RUnlock()
		return token, nil
	}
	tokenMutex.RUnlock()

	// Fetch new token using Client Credentials flow
	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	// Double-check after acquiring write lock
	if cachedToken != "" && time.Now().Before(tokenExpiry) {
		return cachedToken, nil
	}

	log.Println("🔄 Fetching Microsoft access token (client_credentials)...")

	// Build token request - exactly like the Google Apps Script version
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", microsoftConfig.TenantID)

	data := url.Values{}
	data.Set("client_id", microsoftConfig.ClientID)
	data.Set("grant_type", "client_credentials")
	data.Set("scope", "https://graph.microsoft.com/.default")
	data.Set("client_secret", microsoftConfig.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
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
	cachedToken = tokenResp.AccessToken
	tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	log.Printf("✅ Microsoft token fetched successfully (expires in %d seconds)", tokenResp.ExpiresIn)

	return cachedToken, nil
}

// GetAuthenticatedClient returns an HTTP client that adds auth header to requests
func GetAuthenticatedClient(ctx context.Context) (*http.Client, error) {
	token, err := getToken(ctx)
	if err != nil {
		return nil, err
	}

	// Return a client that wraps the base client with auth header
	return &http.Client{
		Timeout: microsoftConfig.Timeout,
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

// TestConnection tests the Microsoft Graph connection
// For client_credentials flow, we just verify we can get a token - no API call needed
func TestConnection(ctx context.Context) error {
	// Simply verify we can get a token - this proves the credentials work
	_, err := getToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	log.Println("✅ Microsoft Graph connection test successful (token verified)")
	return nil
}

// Close cleans up Microsoft client resources
func Close() error {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()
	cachedToken = ""
	tokenExpiry = time.Time{}
	log.Println("✅ Microsoft Graph client closed successfully")
	return nil
}
