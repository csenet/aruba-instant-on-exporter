package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/csenet/instanton-exporter/models"
)

type Client struct {
	httpClient      *http.Client
	settings        *models.Settings
	username        string
	password        string
	token           *models.AuthToken
	tokenObtainedAt time.Time
	sessionToken    string
	pkceChallenge   *PKCEChallenge
}

func NewClient(username, password string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		username: username,
		password: password,
	}
}

func (c *Client) FetchSettings() error {
	resp, err := c.httpClient.Get("https://portal.arubainstanton.com/settings.json")
	if err != nil {
		return fmt.Errorf("failed to fetch settings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to fetch settings: status %d, body: %s", resp.StatusCode, string(body))
	}

	var settings models.Settings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return fmt.Errorf("failed to decode settings: %w", err)
	}


	c.settings = &settings
	return nil
}

func (c *Client) GetTemporaryAccessToken() error {
	if c.settings == nil {
		if err := c.FetchSettings(); err != nil {
			return err
		}
	}

	// Step 1: Get temporary access token via MFA validation (this becomes our sessionToken)
	data := url.Values{}
	data.Set("username", c.username)
	data.Set("password", c.password)

	mfaURL := "https://sso.arubainstanton.com/aio/api/v1/mfa/validate/full"


	req, err := http.NewRequest("POST", mfaURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create MFA request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send MFA request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read MFA response: %w", err)
	}


	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MFA validation failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var mfaResp models.AuthToken
	if err := json.Unmarshal(body, &mfaResp); err != nil {
		return fmt.Errorf("failed to decode MFA response: %w", err)
	}

	if !mfaResp.Success {
		return fmt.Errorf("MFA validation was not successful")
	}

	if mfaResp.AccessToken == "" {
		return fmt.Errorf("no access token received from MFA validation")
	}

	c.sessionToken = mfaResp.AccessToken
	fmt.Printf("[INFO] Session token obtained (expires in %d seconds)\n", mfaResp.ExpiresIn)

	return nil
}

func (c *Client) GetAuthorizationCode() (string, error) {
	if c.sessionToken == "" {
		if err := c.GetTemporaryAccessToken(); err != nil {
			return "", err
		}
	}

	// Generate PKCE challenge
	pkce, err := GeneratePKCE()
	if err != nil {
		return "", fmt.Errorf("failed to generate PKCE: %w", err)
	}
	c.pkceChallenge = pkce

	// Build authorization URL
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", c.settings.SSOClientIDAuthZ)
	params.Set("scope", "profile openid")
	params.Set("redirect_uri", "https://portal.arubainstanton.com")
	params.Set("code_challenge", pkce.Challenge)
	params.Set("code_challenge_method", "S256")
	params.Set("sessionToken", c.sessionToken)

	authURL := fmt.Sprintf("%s%s?%s", c.settings.SSOBaseURL, c.settings.SSOEndpointAuthZ, params.Encode())


	// Make GET request with redirect disabled to capture the authorization code
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(authURL)
	if err != nil {
		return "", fmt.Errorf("failed to get authorization code: %w", err)
	}
	defer resp.Body.Close()

	// Get the redirect location
	location := resp.Header.Get("Location")

	if location == "" {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("no redirect location found, status: %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse the authorization code from redirect URL
	redirectURL, err := url.Parse(location)
	if err != nil {
		return "", fmt.Errorf("failed to parse redirect URL: %w", err)
	}

	code := redirectURL.Query().Get("code")
	if code == "" {
		return "", fmt.Errorf("authorization code not found in redirect URL: %s", location)
	}

	fmt.Printf("[INFO] Authorization code obtained\n")
	return code, nil
}

func (c *Client) GetAccessToken() error {
	if c.settings == nil {
		if err := c.FetchSettings(); err != nil {
			return err
		}
	}

	// Get authorization code
	code, err := c.GetAuthorizationCode()
	if err != nil {
		return fmt.Errorf("failed to get authorization code: %w", err)
	}

	// Exchange code for access token using PKCE
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", c.settings.SSOClientIDAuthZ)
	data.Set("redirect_uri", "https://portal.arubainstanton.com")
	data.Set("code", code)
	data.Set("code_verifier", c.pkceChallenge.Verifier)

	tokenURL := fmt.Sprintf("%s%s", c.settings.SSOBaseURL, c.settings.SSOEndpointTokens)


	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read token response: %w", err)
	}


	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token exchange failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp models.AuthToken
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	c.token = &tokenResp
	c.tokenObtainedAt = time.Now()
	fmt.Printf("[INFO] Access token obtained (expires in %d seconds)\n", tokenResp.ExpiresIn)

	return nil
}

// IsTokenExpired checks if the current access token is expired or will expire soon
// Returns true if token needs refresh (expired or will expire in next 5 minutes)
func (c *Client) IsTokenExpired() bool {
	if c.token == nil || c.token.AccessToken == "" {
		return true
	}

	// Add a 5-minute buffer to refresh before actual expiration
	expirationBuffer := 5 * time.Minute
	expiresAt := c.tokenObtainedAt.Add(time.Duration(c.token.ExpiresIn) * time.Second)

	return time.Now().Add(expirationBuffer).After(expiresAt)
}

func (c *Client) GetToken() (string, error) {
	if c.IsTokenExpired() {
		fmt.Printf("[INFO] Access token expired or expiring soon, refreshing...\n")
		if err := c.GetAccessToken(); err != nil {
			return "", err
		}
	}
	return c.token.AccessToken, nil
}