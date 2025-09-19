package ebay

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// OAuthConfig holds eBay OAuth configuration
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
	Sandbox      bool
}

// OAuthToken represents an eBay OAuth token
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"-"`

	// Additional fields from eBay OAuth response
	EBayUserID string `json:"ebay_user_id,omitempty"` // eBay user identifier
	UserID     string `json:"user_id,omitempty"`      // Numeric user ID
}

// OAuthManager handles eBay OAuth authentication
type OAuthManager struct {
	config         OAuthConfig
	httpClient     *http.Client
	sessionManager *SessionManager

	// Token storage (in production, use secure storage)
	mu     sync.RWMutex
	tokens map[string]*OAuthToken // ebayUserID -> token
}

// NewOAuthManager creates a new OAuth manager
func NewOAuthManager(config OAuthConfig) *OAuthManager {
	return &OAuthManager{
		config:         config,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		sessionManager: NewSessionManager(),
		tokens:         make(map[string]*OAuthToken),
	}
}

// GetAuthorizationURL generates the OAuth authorization URL
func (m *OAuthManager) GetAuthorizationURL(state string) string {
	baseURL := "https://auth.ebay.com/oauth2/authorize"
	if m.config.Sandbox {
		baseURL = "https://auth.sandbox.ebay.com/oauth2/authorize"
	}

	params := url.Values{}
	params.Set("client_id", m.config.ClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", m.config.RedirectURI)
	params.Set("scope", strings.Join(m.config.Scopes, " "))
	params.Set("state", state)

	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

// ExchangeCodeForToken exchanges authorization code for access token
func (m *OAuthManager) ExchangeCodeForToken(code string) (*OAuthToken, error) {
	tokenURL := "https://api.ebay.com/identity/v1/oauth2/token"
	if m.config.Sandbox {
		tokenURL = "https://api.sandbox.ebay.com/identity/v1/oauth2/token"
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", m.config.RedirectURI)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set headers
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", m.config.ClientID, m.config.ClientSecret)))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var token OAuthToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}

	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	// Get user info from eBay to populate user IDs
	if err := m.populateUserInfo(&token); err != nil {
		// Log error but don't fail - we can still use the token
		// In production, log this properly
	}

	return &token, nil
}

// RefreshAccessToken refreshes an expired access token
func (m *OAuthManager) RefreshAccessToken(refreshToken string) (*OAuthToken, error) {
	tokenURL := "https://api.ebay.com/identity/v1/oauth2/token"
	if m.config.Sandbox {
		tokenURL = "https://api.sandbox.ebay.com/identity/v1/oauth2/token"
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("scope", strings.Join(m.config.Scopes, " "))

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set headers
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", m.config.ClientID, m.config.ClientSecret)))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed: %s", string(body))
	}

	var token OAuthToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}

	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	return &token, nil
}

// GetValidToken returns a valid access token, refreshing if necessary
func (m *OAuthManager) GetValidToken(ebayUserID string) (*OAuthToken, error) {
	m.mu.RLock()
	token, exists := m.tokens[ebayUserID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no token found for eBay user %s", ebayUserID)
	}

	// Check if token is expired or about to expire (5 minute buffer)
	if time.Now().Add(5 * time.Minute).After(token.ExpiresAt) {
		// Need to refresh
		newToken, err := m.RefreshAccessToken(token.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("refreshing token: %w", err)
		}

		// Copy user IDs from old token
		newToken.EBayUserID = token.EBayUserID
		newToken.UserID = token.UserID

		// Update stored token
		m.mu.Lock()
		m.tokens[ebayUserID] = newToken
		m.mu.Unlock()

		return newToken, nil
	}

	return token, nil
}

// StoreToken stores a token for a user and creates a session
func (m *OAuthManager) StoreToken(ebayUserID string, token *OAuthToken, ipAddress string) (*Session, error) {
	// Store token
	m.mu.Lock()
	m.tokens[ebayUserID] = token
	m.mu.Unlock()

	// Create session
	session, err := m.sessionManager.CreateSession(ebayUserID, ipAddress)
	if err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}

	return session, nil
}

// GetSession returns a session by ID
func (m *OAuthManager) GetSession(sessionID string) (*Session, bool) {
	return m.sessionManager.GetSession(sessionID)
}

// GetUserBySession returns the eBay user ID for a session
func (m *OAuthManager) GetUserBySession(sessionID string) (string, bool) {
	session, exists := m.sessionManager.GetSession(sessionID)
	if !exists {
		return "", false
	}
	return session.EBayUserID, true
}

// DeleteSession removes a session
func (m *OAuthManager) DeleteSession(sessionID string) {
	m.sessionManager.DeleteSession(sessionID)
}

// populateUserInfo fetches user information from eBay API
func (m *OAuthManager) populateUserInfo(token *OAuthToken) error {
	// Use eBay Account API to get user info
	endpoint := "https://apiz.ebay.com/commerce/identity/v1/user/"
	if m.config.Sandbox {
		endpoint = "https://apiz.sandbox.ebay.com/commerce/identity/v1/user/"
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("user info request failed with status %d", resp.StatusCode)
	}

	var userInfo struct {
		UserID   string `json:"userId"`
		Username string `json:"username"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return fmt.Errorf("parsing user info: %w", err)
	}

	// Populate token with user info
	token.UserID = userInfo.UserID
	token.EBayUserID = userInfo.Username

	// If username is empty, use UserID as fallback
	if token.EBayUserID == "" {
		token.EBayUserID = userInfo.UserID
	}

	return nil
}

// GenerateState generates a random state parameter for OAuth
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
