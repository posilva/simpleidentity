package providers

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"
)

func TestProviderApple_Returns_AppleAuthResult(t *testing.T) {
	cts := context.Background()

	keyGen := TestKeyPairGenerator{}
	keyGen.GenerateRSAKeys()
	mux := http.NewServeMux()
	mux.HandleFunc("/authCode", appleAuthURIHandler(10, keyGen.PrivateKey, true, 1, true))
	mux.HandleFunc("/certs", appleCertsURLHandler(keyGen.PublicKey))

	ts := httptest.NewServer(mux)
	defer ts.Close()

	credentials := AppleCredentials{
		AuthTokensURL:           ts.URL + "/authCode",
		CertsURL:                ts.URL + "/certs",
		ClientID:                "apple_client_id",
		ClientSecret:            "apple_client_secret",
		IDTokenExpectedAudience: testExpectedAudience,
		IDTokenExpectedIssuer:   testExpectedIssuer,
	}

	p := NewAppleProvider(credentials)
	res, err := p.Authenticate(cts, map[string]string{
		AppleIdentityTokenFieldName:     generateAppleIDToken(10, keyGen.PrivateKey, true, 1, true),
		AppleAuthorizationCodeFieldName: "auth_code",
		AppleNonceFieldName:             testExpectedNonce,
		AppleUserIDFieldName:            testSubject,
		AppleEmailFieldName:             testEmail,
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, res.GetID(), testSubject)
}

func TestProviderApple_Returns_Error(t *testing.T) {
	// TODO: create a table test to cover all the errors
	cts := context.Background()

	keyGen := TestKeyPairGenerator{}
	keyGen.GenerateRSAKeys()
	mux := http.NewServeMux()
	mux.HandleFunc("/authCode", appleAuthURIHandler(10, keyGen.PrivateKey, true, 1, true))
	mux.HandleFunc("/certs", appleCertsURLHandler(keyGen.PublicKey))

	ts := httptest.NewServer(mux)
	defer ts.Close()

	credentials := AppleCredentials{
		AuthTokensURL:           ts.URL + "/authCode",
		CertsURL:                ts.URL + "/certs",
		ClientID:                "apple_client_id",
		ClientSecret:            "apple_client_secret",
		IDTokenExpectedAudience: testExpectedAudience,
		IDTokenExpectedIssuer:   testExpectedIssuer,
	}

	p := NewAppleProvider(credentials)
	res, err := p.Authenticate(cts, map[string]string{
		AppleIdentityTokenFieldName:     generateAppleIDToken(10, keyGen.PrivateKey, true, 1, true),
		AppleAuthorizationCodeFieldName: "auth_code",
		AppleNonceFieldName:             "unexpected_nonce",
		AppleUserIDFieldName:            testSubject,
		AppleEmailFieldName:             testEmail,
	})
	require.Error(t, err)
	require.Nil(t, res)
}

func generateAppleIDToken(secs int, privateKey *rsa.PrivateKey, isPrivateEmail bool, realUserStatus int, useNounce bool) string {
	// https://developer.apple.com/documentation/signinwithapple/authenticating-users-with-sign-in-with-apple#Retrieve-the-users-information-from-Apple-ID-servers
	claims := jwt.MapClaims{
		"iss":              testExpectedIssuer,
		"sub":              testSubject,
		"aud":              testExpectedAudience,
		"iat":              time.Now().Add(-1 * time.Hour).Unix(),
		"exp":              time.Now().Add(time.Second * time.Duration(secs)).Unix(),
		"nonce":            testExpectedNonce,
		"nonce_supported":  useNounce,
		"email":            testEmail,
		"email_verified":   true,
		"is_private_email": isPrivateEmail,
		"real_user_status": realUserStatus,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = testKeyID

	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		panic(err)
	}
	return signedToken
}

func appleAuthURIHandler(secs int, privateKey *rsa.PrivateKey, isPrivateEmail bool, realUserStatus int, useNounce bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/authCode" {
			http.NotFound(w, r)
			return
		}

		data := exchangeTokenResponse{
			AccessToken:  "access_token",
			TokenType:    "token_type",
			ExpiresIn:    time.Now().Add(time.Duration(secs) * time.Second).Unix(),
			RefreshToken: "refresh_token",
			IDToken:      generateAppleIDToken(secs, privateKey, isPrivateEmail, realUserStatus, useNounce),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			panic(fmt.Errorf("failed to encode response in json: %w", err))
		}
	}
}

func appleCertsURLHandler(pubKey *rsa.PublicKey) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/certs" {
			http.NotFound(w, r)
			return
		}
		jwk := jose.JSONWebKey{
			Key:       pubKey,
			KeyID:     testKeyID,
			Use:       "sig",
			Algorithm: string(jose.RS256),
		}

		// 3. Marshal the JWK to JSON
		jwkJSON, err := json.MarshalIndent(map[string][]jose.JSONWebKey{"keys": {jwk}}, "", "  ")
		if err != nil {
			panic(fmt.Errorf("failed to marshal JWK: %w", err))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(jwkJSON)
	}
}
