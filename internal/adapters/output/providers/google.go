package providers

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/posilva/simpleidentity/internal/adapters/output/providers/certs"
	"github.com/posilva/simpleidentity/internal/core/domain"
	"github.com/posilva/simpleidentity/internal/core/ports"
)

// References:
// - https://pkg.go.dev/google.golang.org/api/androidpublisher/v3
// - https://developers.google.com/identity/sign-in/android/backend-auth
// - https://developer.android.com/games/pgs/sign-in

// TODO: Consider to implement a retry policy
// TODO: Implement metrics, traces specially with the new client inject context with metadata

const (
	defaultTimeout = 2 * time.Second
)

const (
	GoogleAuthCodeFieldName = "token"
)

type googleIDTokenClaims struct {
	Issuer   string `json:"iss"`
	Subject  string `json:"sub"`
	Audience string `json:"aud"`
	Email    string `json:"email"`
	Expiry   int64  `json:"exp"`
	jwt.RegisteredClaims
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
	IDToken      string `json:"id_token"`
}

// GoogleCredentials defines the interface to get the needed Google Credentials
type GoogleCredentials struct {
	ClientID              string
	ClientSecret          string
	PrivateKey            string
	AuthURI               string
	CertsURL              string
	IDTokenExpectedIssuer string
	IDTokenExpectedAud    string
}

type googleProvider struct {
	requestTimeout time.Duration
	credentials    GoogleCredentials
	cacheManager   certs.CacheManager
}

type googleAuthResult struct {
	ID string
}

type GoogleProviderOption func(*googleProvider)

func WithTimeout(timeout time.Duration) GoogleProviderOption {
	return func(p *googleProvider) {
		p.requestTimeout = timeout
	}
}

func WithCertificatesCacheManager(cm certs.CacheManager) GoogleProviderOption {
	return func(p *googleProvider) {
		p.cacheManager = cm
	}
}

func (r *googleAuthResult) GetID() string {
	return r.ID
}

// NewGoogleProvider creates a new GoogleProvider
// serviceAccount is a placeholder for the Google service account credentials in json format.
func NewGoogleProvider(credentials GoogleCredentials, opts ...GoogleProviderOption) ports.AuthProvider {
	svc := &googleProvider{
		requestTimeout: defaultTimeout,
		credentials:    credentials,
		cacheManager:   certs.NewSimpleCacheManager(),
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// Authenticate executes authentication with Google and returns an authresult.
func (p *googleProvider) Authenticate(ctx context.Context, data map[string]string) (ports.AuthResult, error) {
	authToken, ok := data[GoogleAuthCodeFieldName]
	if !ok {
		return nil, domain.ErrMissingRequiredProviderAuthData
	}
	resp, err := p.exchangeAuthCode(authToken)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange auth code: %w", err)
	}

	claims, err := p.verifyIDToken(resp.IDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify id token: %w", err)
	}

	return &googleAuthResult{ID: claims.Subject}, nil
}

func (p *googleProvider) exchangeAuthCode(authCode string) (*tokenResponse, error) {
	form := url.Values{}
	form.Add("code", authCode)
	form.Add("client_id", p.credentials.ClientID)
	form.Add("client_secret", p.credentials.ClientSecret)
	form.Add("redirect_uri", "") // this is mobile we can keep empty
	form.Add("grant_type", "authorization_code")

	// TODO: PMS replace by a more robust http client setup using the configured timeout
	resp, err := http.PostForm(p.credentials.AuthURI, form)
	if err != nil {
		return nil, fmt.Errorf("failed to post to token endpoint: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		var body bytes.Buffer
		_, _ = body.ReadFrom(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s", body.String())
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}

// fetchPublicKeyById fetches Google's public certs (PEM format)
func (p *googleProvider) fetchPublicKeyByID(id string) (*rsa.PublicKey, error) {
	key := p.cacheManager.Get(id)
	if key == nil {
		resp, err := http.Get(p.credentials.CertsURL)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		expiresHeader := resp.Header.Get("Expires")
		expiresAt, err := time.Parse(time.RFC1123, expiresHeader)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expires header: %w", err)
		}

		certs := map[string]string{}
		if err := json.NewDecoder(resp.Body).Decode(&certs); err != nil {
			return nil, err
		}

		keys := map[string]*rsa.PublicKey{}
		for kid, certPEM := range certs {
			block, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(certPEM))
			keys[kid] = block
		}

		for i, k := range keys {
			_ = p.cacheManager.Add(i, k, expiresAt)
		}

		key = p.cacheManager.Get(id)
		if key == nil {
			return nil, fmt.Errorf("public key id '%s' not found", id)
		}
	}
	return key, nil
}

func (p *googleProvider) verifyIDToken(idToken string) (*googleIDTokenClaims, error) {
	token, err := jwt.ParseWithClaims(idToken, &googleIDTokenClaims{}, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("no kid found in token header")
		}

		pubKey, err := p.fetchPublicKeyByID(kid)
		if err != nil {
			return nil, fmt.Errorf("failed to get public keys: %w", err)
		}

		return pubKey, nil
	}, jwt.WithLeeway(30*time.Second))
	if err != nil {
		return nil, fmt.Errorf("token parse error %s: %w", idToken, err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*googleIDTokenClaims)
	if !ok {
		return nil, errors.New("invalid claims format")
	}

	if claims.Issuer != p.credentials.IDTokenExpectedIssuer {
		return nil, errors.New("invalid issuer")
	}
	if claims.Audience != p.credentials.IDTokenExpectedAud {
		return nil, errors.New("invalid audience")
	}

	return claims, nil
}
