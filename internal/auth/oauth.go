package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Scopes required by the sheets-mcp server.
var Scopes = []string{
	"https://www.googleapis.com/auth/spreadsheets",
	"https://www.googleapis.com/auth/drive.metadata.readonly",
}

// oauthFlowTimeout is how long the user has to complete the browser consent.
const oauthFlowTimeout = 2 * time.Minute

// oauthCallbackShutdownTimeout is how long Shutdown waits for the callback server to stop.
const oauthCallbackShutdownTimeout = 5 * time.Second

// revokeTimeout bounds the token revocation HTTP request.
const revokeTimeout = 30 * time.Second

const (
	callbackReadHeaderTimeout = 5 * time.Second
	callbackReadTimeout       = 10 * time.Second
	callbackWriteTimeout      = 10 * time.Second
)

// OAuthConfig builds an oauth2.Config from the application config and the
// given redirect URL.
func OAuthConfig(cfg *Config, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Scopes:       Scopes,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURL,
	}
}

// RunOAuthFlow performs the full browser-based OAuth consent flow:
//  1. Binds a local HTTP server on localhost:0 (OS-assigned port)
//  2. Builds the Google consent URL with the actual port in the redirect URI
//  3. Opens the system browser via xdg-open
//  4. Waits for the callback with a 2-minute timeout
//  5. Exchanges the authorization code for tokens
//  6. Saves the tokens to disk
//  7. Shuts down the local server
func RunOAuthFlow(cfg *Config) (*oauth2.Token, error) {
	// Bind listener on a random port.
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("bind local callback server: %w", err)
	}
	defer func() { _ = listener.Close() }()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURL := fmt.Sprintf("http://localhost:%d/callback", port)
	oauthCfg := OAuthConfig(cfg, redirectURL)

	// Generate random state for CSRF protection.
	state, err := randomState()
	if err != nil {
		return nil, fmt.Errorf("generate OAuth state: %w", err)
	}

	consentURL := oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	// Channel to receive the auth code or error from the callback handler.
	type callbackResult struct {
		code string
		err  error
	}
	resultCh := make(chan callbackResult, 1)
	trySend := func(result callbackResult) bool {
		select {
		case resultCh <- result:
			return true
		default:
			return false
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			if !trySend(callbackResult{err: fmt.Errorf("OAuth state mismatch")}) {
				http.Error(w, "Session already completed", http.StatusBadRequest)
				return
			}
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}

		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			if !trySend(callbackResult{err: fmt.Errorf("OAuth error from Google: %s", errMsg)}) {
				http.Error(w, "Session already completed", http.StatusBadRequest)
				return
			}
			_, _ = fmt.Fprintf(w, "Authentication failed: %s. You can close this tab.", errMsg)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			if !trySend(callbackResult{err: fmt.Errorf("no authorization code in callback")}) {
				http.Error(w, "Session already completed", http.StatusBadRequest)
				return
			}
			http.Error(w, "Missing code", http.StatusBadRequest)
			return
		}

		if !trySend(callbackResult{code: code}) {
			_, _ = fmt.Fprint(w, "This authentication session was already completed. You can close this tab.")
			return
		}
		_, _ = fmt.Fprint(w, "Authentication successful! You can close this tab.")
	})

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: callbackReadHeaderTimeout,
		ReadTimeout:       callbackReadTimeout,
		WriteTimeout:      callbackWriteTimeout,
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), oauthCallbackShutdownTimeout)
		defer cancel()
		_ = server.Shutdown(stopCtx)
	}()

	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			trySend(callbackResult{err: fmt.Errorf("callback server: %w", serveErr)})
		}
	}()

	// Open the browser.
	fmt.Printf("Opening browser for Google authentication...\n")
	fmt.Printf("If the browser doesn't open, visit this URL:\n%s\n\n", consentURL)

	if err := openBrowser(consentURL); err != nil {
		fmt.Printf("Warning: could not open browser automatically: %v\n", err)
	}

	// Wait for callback or timeout.
	select {
	case result := <-resultCh:
		if result.err != nil {
			return nil, result.err
		}

		// Exchange code for token.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		tok, err := oauthCfg.Exchange(ctx, result.code)
		if err != nil {
			return nil, fmt.Errorf("exchange authorization code: %w", err)
		}

		if err := SaveToken(tok); err != nil {
			return nil, err
		}

		return tok, nil

	case <-time.After(oauthFlowTimeout):
		return nil, fmt.Errorf("authentication timed out after %v. Run 'sheets-mcp auth' to try again", oauthFlowTimeout)
	}
}

// RevokeToken revokes the token via Google's revocation endpoint and deletes
// the token file.
func RevokeToken(tok *oauth2.Token) error {
	// Prefer revoking the refresh token; fall back to access token.
	revokeTok := tok.RefreshToken
	if revokeTok == "" {
		revokeTok = tok.AccessToken
	}
	if revokeTok == "" {
		// Nothing to revoke, just delete the file.
		return DeleteToken()
	}

	ctx, cancel := context.WithTimeout(context.Background(), revokeTimeout)
	defer cancel()

	form := url.Values{}
	form.Set("token", revokeTok)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/revoke", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("revoke token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: revokeTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("revoke token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	_, _ = io.Copy(io.Discard, resp.Body)

	// Google returns 200 on success. A 400 typically means the token is
	// already invalid, which is fine — we still delete the local file.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		return fmt.Errorf("revoke token: Google returned HTTP %d", resp.StatusCode)
	}

	return DeleteToken()
}

// openBrowser opens the given URL in the default system browser.
func openBrowser(url string) error {
	return exec.Command("xdg-open", url).Start()
}

// randomState generates a random hex string for OAuth state parameter.
func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
