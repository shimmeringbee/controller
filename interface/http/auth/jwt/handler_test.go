package jwt

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/golang-jwt/jwt"
	"github.com/shimmeringbee/controller/interface/http/auth"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var testPrivateKey = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIKibFA7Z1Qt18ANQVLseQcKYjjPLC0IDJFBiwOKyXZ/aoAoGCCqGSM49
AwEHoUQDQgAE5P+Q+WlIAyxnElejiN4vwQRPv8HfdKQg1wDzJncSJA+byHhg6cCZ
8dbv6iSlFL1B8yMliWBZmEhIQ/hzxPACGA==
-----END EC PRIVATE KEY-----`)

func TestAuthenticator_SignAndVerify(t *testing.T) {
	t.Run("signs a new JWT for the uid provided", func(t *testing.T) {
		pemBlock, _ := pem.Decode(testPrivateKey)
		privateKey, _ := x509.ParseECPrivateKey(pemBlock.Bytes)

		a := Authenticator{
			SystemIdentifier: "fixedIdentity",
			TTL:              30 * time.Second,

			KeyIdentifier: "kid",
			PrivateKey:    privateKey,
		}

		expectedUid := "uid"

		generatedToken, err := a.Sign(expectedUid)
		assert.NoError(t, err)

		actualUid, err := a.Verify(generatedToken)
		assert.NoError(t, err)
		assert.Equal(t, expectedUid, actualUid)
	})

	t.Run("verify fails if a JWT is provided with a None alg", func(t *testing.T) {
		jwtWithAlgNone := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIiwgImtpZCI6ImtpZCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImp0aSI6ImRmNGNkNjNmLWU2ODAtNDFhMS05NGEyLTA0MDAxOTk2MmNmZiIsImlhdCI6MTYxOTgwMTIwMywiZXhwIjoxNjE5ODA0ODE1fQ.zEMrLBs5f07fbI3z6IUWO-db9xZqWaRerXn0dsbPfSA"

		pemBlock, _ := pem.Decode(testPrivateKey)
		privateKey, _ := x509.ParseECPrivateKey(pemBlock.Bytes)

		a := Authenticator{
			SystemIdentifier: "fixedIdentity",
			TTL:              30 * time.Second,

			KeyIdentifier: "kid",
			PrivateKey:    privateKey,
		}

		actualUid, err := a.Verify(jwtWithAlgNone)
		assert.Error(t, err)
		assert.Empty(t, actualUid)
	})

	t.Run("verify fails if a JWT is provided with an unknown kid", func(t *testing.T) {
		pemBlock, _ := pem.Decode(testPrivateKey)
		privateKey, _ := x509.ParseECPrivateKey(pemBlock.Bytes)

		a := Authenticator{
			SystemIdentifier: "fixedIdentity",
			TTL:              30 * time.Second,

			KeyIdentifier: "kid",
			PrivateKey:    privateKey,
		}

		expectedUid := "uid"

		generatedToken, err := a.Sign(expectedUid)
		assert.NoError(t, err)

		a.KeyIdentifier = "otherkid"

		actualUid, err := a.Verify(generatedToken)
		assert.Error(t, err)
		assert.Empty(t, actualUid)
	})

	t.Run("verify fails if ticket has expired", func(t *testing.T) {
		jwt.TimeFunc = time.Now
		clock = func() time.Time { return time.Date(2021, time.April, 30, 9, 30, 0, 0, time.UTC) }

		pemBlock, _ := pem.Decode(testPrivateKey)
		privateKey, _ := x509.ParseECPrivateKey(pemBlock.Bytes)

		a := Authenticator{
			SystemIdentifier: "fixedIdentity",
			TTL:              30 * time.Second,

			KeyIdentifier: "kid",
			PrivateKey:    privateKey,
		}

		expectedUid := "uid"

		generatedToken, err := a.Sign(expectedUid)
		assert.NoError(t, err)

		actualUid, err := a.Verify(generatedToken)
		assert.Error(t, err)
		assert.Empty(t, actualUid)
	})

	t.Run("verify fails if ticket has used before it was issues", func(t *testing.T) {
		jwt.TimeFunc = func() time.Time { return time.Date(2021, time.April, 30, 9, 30, 0, 0, time.UTC) }
		clock = time.Now

		pemBlock, _ := pem.Decode(testPrivateKey)
		privateKey, _ := x509.ParseECPrivateKey(pemBlock.Bytes)

		a := Authenticator{
			SystemIdentifier: "fixedIdentity",
			TTL:              30 * time.Second,

			KeyIdentifier: "kid",
			PrivateKey:    privateKey,
		}

		expectedUid := "uid"

		generatedToken, err := a.Sign(expectedUid)
		assert.NoError(t, err)

		actualUid, err := a.Verify(generatedToken)
		assert.Error(t, err)
		assert.Empty(t, actualUid)
	})

	t.Run("verify fails if the issuer is not the system identity", func(t *testing.T) {
		jwt.TimeFunc = func() time.Time { return time.Date(2021, time.April, 30, 9, 30, 0, 0, time.UTC) }
		clock = jwt.TimeFunc

		pemBlock, _ := pem.Decode(testPrivateKey)
		privateKey, _ := x509.ParseECPrivateKey(pemBlock.Bytes)

		a := Authenticator{
			SystemIdentifier: "fixedIdentity",
			TTL:              30 * time.Second,

			KeyIdentifier: "kid",
			PrivateKey:    privateKey,
		}

		expectedUid := "uid"

		generatedToken, err := a.Sign(expectedUid)
		assert.NoError(t, err)

		a.SystemIdentifier = "otherSystemIdentity"

		actualUid, err := a.Verify(generatedToken)
		assert.Error(t, err)
		assert.Empty(t, actualUid)
	})
}

func failTestHandler(t *testing.T) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		t.Fatal("Downstream handler called, and should not have been.")
	}
}

func TestAuthenticator_AuthenticationMiddleware(t *testing.T) {
	t.Run("verifies that a missing Authentication Bearer results in a 401, and does not call next handler", func(t *testing.T) {
		pemBlock, _ := pem.Decode(testPrivateKey)
		privateKey, _ := x509.ParseECPrivateKey(pemBlock.Bytes)

		a := Authenticator{
			SystemIdentifier: "fixedIdentity",
			TTL:              30 * time.Second,
			KeyIdentifier:    "kid",
			PrivateKey:       privateKey,
		}

		handler := a.AuthenticationMiddleware(failTestHandler(t))

		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Equal(t, "Bearer realm=\"fixedIdentity\"", rr.Header().Get("WWW-Authenticate"))
	})

	t.Run("verifies that a http auth request results in a 401, and does not call next handler", func(t *testing.T) {
		pemBlock, _ := pem.Decode(testPrivateKey)
		privateKey, _ := x509.ParseECPrivateKey(pemBlock.Bytes)

		a := Authenticator{
			SystemIdentifier: "fixedIdentity",
			TTL:              30 * time.Second,
			KeyIdentifier:    "kid",
			PrivateKey:       privateKey,
		}

		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header["Authentication"] = []string{fmt.Sprintf("Basic d2FsbGFjZTpncm9tbWl0")}

		handler := a.AuthenticationMiddleware(failTestHandler(t))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Equal(t, "Bearer realm=\"fixedIdentity\", error=\"invalid_request\", error=\"Incomplete or incompatible authentication provided.\"", rr.Header().Get("WWW-Authenticate"))
	})

	t.Run("verifies that a http auth request results in a 401 with only the word Bearer, and does not call next handler", func(t *testing.T) {
		pemBlock, _ := pem.Decode(testPrivateKey)
		privateKey, _ := x509.ParseECPrivateKey(pemBlock.Bytes)

		a := Authenticator{
			SystemIdentifier: "fixedIdentity",
			TTL:              30 * time.Second,
			KeyIdentifier:    "kid",
			PrivateKey:       privateKey,
		}

		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header["Authentication"] = []string{fmt.Sprintf("Bearer")}

		handler := a.AuthenticationMiddleware(failTestHandler(t))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Equal(t, "Bearer realm=\"fixedIdentity\", error=\"invalid_request\", error=\"Incomplete or incompatible authentication provided.\"", rr.Header().Get("WWW-Authenticate"))
	})

	t.Run("verifies that an invalid Authentication Bearer results in a 401, and does not call next handler", func(t *testing.T) {
		jwt.TimeFunc = time.Now
		clock = time.Now

		pemBlock, _ := pem.Decode(testPrivateKey)
		privateKey, _ := x509.ParseECPrivateKey(pemBlock.Bytes)

		a := Authenticator{
			SystemIdentifier: "fixedIdentity",
			TTL:              30 * time.Second,
			KeyIdentifier:    "kid",
			PrivateKey:       privateKey,
		}

		futureJWT, _ := a.Sign("uid")

		jwt.TimeFunc = func() time.Time { return time.Date(2021, time.April, 30, 9, 30, 0, 0, time.UTC) }
		clock = jwt.TimeFunc

		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header["Authentication"] = []string{fmt.Sprintf("Bearer %s", futureJWT)}

		handler := a.AuthenticationMiddleware(failTestHandler(t))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Equal(t, "Bearer realm=\"fixedIdentity\", error=\"invalid_token\", error=\"Invalid credential.\"", rr.Header().Get("WWW-Authenticate"))
	})

	t.Run("verifies that an valid Authentication Bearer results in the next caller being returned", func(t *testing.T) {
		pemBlock, _ := pem.Decode(testPrivateKey)
		privateKey, _ := x509.ParseECPrivateKey(pemBlock.Bytes)

		a := Authenticator{
			SystemIdentifier: "fixedIdentity",
			TTL:              30 * time.Second,
			KeyIdentifier:    "kid",
			PrivateKey:       privateKey,
		}

		userId := "user id"

		futureJWT, _ := a.Sign(userId)

		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header["Authentication"] = []string{fmt.Sprintf("Bearer %s", futureJWT)}

		handler := a.AuthenticationMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, userId, request.Context().Value(auth.UserIdentityContextKey))
			writer.WriteHeader(200)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
