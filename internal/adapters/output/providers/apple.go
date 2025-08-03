package providers

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/posilva/account-service/internal/adapters/output/providers/certs"
	"github.com/posilva/account-service/internal/core/ports"
)

// References:
// https://developer.apple.com/documentation/signinwithapple/authenticating-users-with-sign-in-with-apple
// https://developer.apple.com/documentation/signinwithapplerestapi/generate-and-validate-tokens
// https://developer.apple.com/documentation/signinwithapple/verifying-a-user

const (
	// identity token, authorization code, and user identifier to your app.
	AppleIdentityTokenFieldName     = "identityToken"
	AppleAuthorizationCodeFieldName = "authorizationCode"
	AppleUserIDFieldName            = "userID"
	AppleNonceFieldName             = "nonce"
	AppleEmailFieldName             = "email"
)

type AppleCredentials struct {
	ClientID                string
	ClientSecret            string
	TeamID                  string
	KeyID                   string
	CertsURL                string
	AuthTokensURL           string
	IDTokenExpectedAudience string
	IDTokenExpectedIssuer   string
}

type appleProvider struct {
	credentials  AppleCredentials
	cacheManager certs.CacheManager
}

type appleAuthResult struct {
	ID string
}
type appleIDTokenClaims struct {
	Issuer         string `json:"iss"`
	Subject        string `json:"sub"`
	Audience       string `json:"aud"`
	IssuedAt       int64  `json:"iat"`
	Email          string `json:"email"`
	Expiry         int64  `json:"exp"`
	Nonce          string `json:"nonce"`
	NonceSupported bool   `json:"nonce_supported"`
	EmailVerified  bool   `json:"email_verified"`
	IsPrivateEmail bool   `json:"is_private_email"`
	RealUserStatus int    `json:"real_user_status"`
	jwt.RegisteredClaims
}

type exchangeTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
}

type exchangeTokenResponseError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type appleJWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type appleJWKS struct {
	Keys []appleJWK `json:"keys"`
}

// TODO: implement the options pattern if it makes sense like we have done for google provider

// NewAppleProvider creates a new Apple provider
func NewAppleProvider(cp AppleCredentials) ports.AuthProvider {
	return &appleProvider{
		credentials:  cp,
		cacheManager: certs.NewSimpleCacheManager(),
	}
}

func (r *appleAuthResult) GetID() string {
	return r.ID
}

func (p *appleProvider) Authenticate(ctx context.Context, data map[string]string) (ports.AuthResult, error) {
	_, ok := data[AppleIdentityTokenFieldName]
	if !ok {
		return nil, fmt.Errorf("missing required field: %s", AppleIdentityTokenFieldName)
	}
	authCode, ok := data[AppleAuthorizationCodeFieldName]
	if !ok {
		return nil, fmt.Errorf("missing required field: %s", AppleAuthorizationCodeFieldName)
	}
	userID, ok := data[AppleUserIDFieldName]
	if !ok {
		return nil, fmt.Errorf("missing required field: %s", AppleUserIDFieldName)
	}
	nonce, ok := data[AppleNonceFieldName]
	if !ok {
		return nil, fmt.Errorf("missing required field: %s", AppleNonceFieldName)
	}
	email, ok := data[AppleEmailFieldName]
	if !ok {
		return nil, fmt.Errorf("missing required field: %s", AppleEmailFieldName)
	}
	/*
		  * TODO: this must be enough to authenticate a user
			claims, err := p.verifyIDToken(idToken, nonce, email)
			if err != nil {
				return nil, fmt.Errorf("failed to verify direct id token: %w", err)
			}
	*/
	exchangeResponse, err := p.exchangeAuthCodeByRefreshToken(authCode)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange auth code: %w", err)
	}

	claims, err := p.verifyIDToken(exchangeResponse.IDToken, nonce, email)
	if err != nil {
		return nil, fmt.Errorf("failed to verify id token: %w", err)
	}
	if userID != claims.Subject {
		return nil, fmt.Errorf("userID mismatch")
	}
	return &appleAuthResult{ID: claims.Subject}, nil
}

func (p *appleProvider) exchangeAuthCodeByRefreshToken(authCode string) (*exchangeTokenResponse, error) {
	// send a form encoded data
	form := url.Values{}
	form.Add("code", authCode)
	form.Add("client_id", p.credentials.ClientID)
	form.Add("client_secret", p.credentials.ClientSecret)
	form.Add("redirect_uri", "")
	form.Add("grant_type", "authorization_code")

	resp, err := http.PostForm(p.credentials.AuthTokensURL, form)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange auth code: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// handler error state in case it happens
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read body from error response: %w", err)
		}

		var errorBody exchangeTokenResponseError
		err = json.Unmarshal(body, &errorBody)
		if err != nil {
			err = fmt.Errorf("error: %s: %s", errorBody.Error, errorBody.ErrorDescription)
			return nil, fmt.Errorf("failed to unmarshal error body: %w", err)
		}
		return nil, fmt.Errorf("failed to exchange auth code with wrong status code %d: %w", resp.StatusCode, err)
	}

	// or handle the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body from exchange response: %w", err)
	}

	var exchangeTokenResponse exchangeTokenResponse
	err = json.Unmarshal(body, &exchangeTokenResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body from json: %w", err)
	}
	return &exchangeTokenResponse, nil
}

func (p *appleProvider) verifyIDToken(idToken string, nonce string, email string) (*appleIDTokenClaims, error) {
	token, err := jwt.ParseWithClaims(idToken, &appleIDTokenClaims{}, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("no kid found in token header")
		}

		pubKey, err := p.fetchPublicKeyByID(kid)
		if err != nil {
			return nil, fmt.Errorf("failed to get public keys: %w", err)
		}
		return pubKey, nil
	},
		jwt.WithLeeway(30*time.Second))
	if err != nil {
		return nil, fmt.Errorf("token parser error: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*appleIDTokenClaims)

	if !ok {
		return nil, errors.New("invalid claims format")
	}

	if claims.Issuer != p.credentials.IDTokenExpectedIssuer {
		return nil, errors.New("invalid issuer")
	}
	if claims.Audience != p.credentials.IDTokenExpectedAudience {
		return nil, errors.New("invalid audience")
	}

	if claims.Nonce != nonce {
		return nil, errors.New("invalid nonce")
	}

	if email != "" && email != claims.Email {
		return nil, errors.New("invalid email")
	}
	return claims, nil
}

// TODO: this method is similar to the one on google provider so maybe we can
// factorise this in a single one
func (p *appleProvider) fetchPublicKeyByID(id string) (*rsa.PublicKey, error) {
	key := p.cacheManager.Get(id)
	if key == nil {
		resp, err := http.Get(p.credentials.CertsURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch public keys from certs url: %w", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read body from apple keys endpoint: %w", err)
		}
		var jwks appleJWKS
		if err := json.Unmarshal(body, &jwks); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		for _, jwk := range jwks.Keys {
			k, err := createPublicKeyFromJWK(jwk)
			if err != nil {
				return nil, fmt.Errorf("failed to create public key from JWK key id %s: %w", jwk.Kid, err)
			}
			expireAt := time.Now().Add(1 * time.Hour)
			_ = p.cacheManager.Add(jwk.Kid, k, expireAt)
		}

		key = p.cacheManager.Get(id)
		if key == nil {
			return nil, fmt.Errorf("public key id '%s' not found", id)
		}
	}
	return key, nil
}

// createPublicKeyFromJWK takes a JSON string containing Apple's JWK data
// and returns an RSA public key that can be used to verify JWT tokens
func createPublicKeyFromJWK(jwk appleJWK) (*rsa.PublicKey, error) {
	if jwk.Kty != "RSA" {
		return nil, fmt.Errorf("expected RSA key type, got: %s", jwk.Kty)
	}

	nBytes, err := base64URLDecode(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	eBytes, err := base64URLDecode(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	publicKey := &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}

	return publicKey, nil
}

func base64URLDecode(data string) ([]byte, error) {
	// Go's base64.URLEncoding handles the URL-safe characters automatically
	// but we need to add padding if it's missing
	switch len(data) % 4 {
	case 2:
		data += "=="
	case 3:
		data += "="
	}

	return base64.URLEncoding.DecodeString(data)
}
