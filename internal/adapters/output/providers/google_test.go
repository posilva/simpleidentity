package providers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/posilva/account-service/internal/adapters/output/providers/certs"
	"github.com/stretchr/testify/require"
)

const (
	testExpectedIssuer   = "google_issuer"
	testExpectedAudience = "myapp_audience"
	testSubject          = "1234567890"
	testKeyID            = "test_key_id"
)

var (
	privateKeyString string
	publicKeyString  string
	privateKey       *rsa.PrivateKey
)

func TestMain(t *testing.T) {
	privateKeyString, publicKeyString, privateKey = createRSAKeys()
}

func TestProviderGoogle_Returns_GoogleAuthResult(t *testing.T) {
	ctx := context.Background()
	token := "auth_code"

	ctrl := mock.NewMockController(t)

	mux := http.NewServeMux()

	mux.HandleFunc("/authCode", googleAuthURIHandler(10))
	mux.HandleFunc("/certs", googleCertsURLHandler(publicKeyString))

	ts := httptest.NewServer(mux)
	defer ts.Close()

	credentialProviderMock := createCredentialProviderMock(ctrl, ts)

	p := NewGoogleProvider(credentialProviderMock,
		WithTimeout(1*time.Second), WithCertificatesCacheManager(certs.NewGoogleCacheManager()))
	res, err := p.Authenticate(ctx, map[string]string{GoogleAuthCodeFieldName: token})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, res.GetID(), testSubject)
}

// ################# auxiliary function to make tests less verbose
func createCredentialProviderMock(ctrl *matchers.MockController, ts *httptest.Server) GoogleCredentialsProvider {
	credentialProviderMock := mock.Mock[GoogleCredentialsProvider](ctrl)
	mock.WhenSingle(credentialProviderMock.GetAuthURI()).ThenReturn(ts.URL + "/authCode")
	mock.WhenSingle(credentialProviderMock.GetCertsURL()).ThenReturn(ts.URL + "/certs")
	mock.WhenSingle(credentialProviderMock.GetClientID()).ThenReturn("google_client_id")
	mock.WhenSingle(credentialProviderMock.GetClientSecret()).ThenReturn("google_client_secret")
	mock.WhenSingle(credentialProviderMock.GetIDExpectedAud()).ThenReturn(testExpectedAudience)
	mock.WhenSingle(credentialProviderMock.GetIDExpectedIssuer()).ThenReturn(testExpectedIssuer)
	mock.WhenSingle(credentialProviderMock.GetPrivateKey()).ThenReturn(privateKeyString)
	return credentialProviderMock
}

// to reduce verbosity adding named returns
func createRSAKeys() (priv string, pub string, privateKey *rsa.PrivateKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	publicKey := &privateKey.PublicKey

	privDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privDER,
	}
	privateKeyPEM := pem.EncodeToMemory(privBlock)
	priv = string(privateKeyPEM)

	pubDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		panic(err)
	}
	pubBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubDER,
	}
	publicKeyPEM := pem.EncodeToMemory(pubBlock)
	pub = string(publicKeyPEM)
	return
}

func generateIDToken(secs int) string {
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

func googleAuthURIHandler(secs int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := tokenResponse{
			AccessToken:  "access_token",
			ExpiresIn:    time.Now().Add(time.Duration(secs) * time.Second).Unix(),
			RefreshToken: "refresh_token",
			Scope:        "scope",
			TokenType:    "token_type",
			IDToken:      generateIDToken(10),
		}

		b, _ := json.Marshal(t)

		_, _ = w.Write(b)
	}
}

func googleCertsURLHandler(pub string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expires := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC1123)

		response := map[string]any{
			testKeyID: publicKeyString,
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("expires", expires)
		w.WriteHeader(http.StatusOK)

		_ = json.NewEncoder(w).Encode(response)
	}
}
