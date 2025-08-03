package providers

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestProviderGoogle_Returns_GoogleAuthResult(t *testing.T) {
	ctx := context.Background()
	authCode := "auth_code"

	keyGen := TestKeyPairGenerator{}
	keyGen.GenerateRSAKeys()

	mux := http.NewServeMux()

	mux.HandleFunc("/authCode", googleAuthURIHandler(10, keyGen.PrivateKey))
	mux.HandleFunc("/certs", googleCertsURLHandler(keyGen.PublicKeyStr))

	ts := httptest.NewServer(mux)
	defer ts.Close()

	credentials := GoogleCredentials{
		AuthURI:               ts.URL + "/authCode",
		CertsURL:              ts.URL + "/certs",
		ClientID:              "google_client_id",
		ClientSecret:          "google_client_secret",
		IDTokenExpectedAud:    testExpectedAudience,
		IDTokenExpectedIssuer: testExpectedIssuer,
		PrivateKey:            keyGen.PrivateKeyStr,
	}

	p := NewGoogleProvider(credentials, WithTimeout(1*time.Second))
	res, err := p.Authenticate(ctx, map[string]string{GoogleAuthCodeFieldName: authCode})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, res.GetID(), testSubject)
}

func generateGoogleIDToken(secs int, privateKey *rsa.PrivateKey) string {
	claims := jwt.MapClaims{
		"sub":   testSubject,
		"exp":   time.Now().Add(time.Second * time.Duration(secs)).Unix(),
		"email": "player01@example.com",
		"aud":   testExpectedAudience,
		"iss":   testExpectedIssuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = testKeyID

	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		panic(err)
	}
	return signedToken
}

// Helper functions to generate test http handlers data

func googleAuthURIHandler(secs int, privateKey *rsa.PrivateKey) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := tokenResponse{
			AccessToken:  "access_token",
			ExpiresIn:    time.Now().Add(time.Duration(secs) * time.Second).Unix(),
			RefreshToken: "refresh_token",
			Scope:        "scope",
			TokenType:    "token_type",
			IDToken:      generateGoogleIDToken(10, privateKey),
		}

		b, _ := json.Marshal(t)

		_, _ = w.Write(b)
	}
}

func googleCertsURLHandler(pub string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expires := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC1123)

		response := map[string]any{
			testKeyID: pub,
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("expires", expires)
		w.WriteHeader(http.StatusOK)

		_ = json.NewEncoder(w).Encode(response)
	}
}
