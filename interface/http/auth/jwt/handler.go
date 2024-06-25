package jwt

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/interface/http/auth"
	"net/http"
	"strings"
	"time"
)

var clock = time.Now

var _ auth.AuthenticationProvider = (*Authenticator)(nil)

type Authenticator struct {
	SystemIdentifier string
	TTL              time.Duration

	KeyIdentifier string
	PrivateKey    *ecdsa.PrivateKey
}

func (a Authenticator) AuthenticationRouter() http.Handler {
	return mux.NewRouter()
}

func (a Authenticator) AuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader, found := r.Header["Authentication"]
		if !found || len(authHeader) != 1 {
			w.Header().Add("WWW-Authenticate", fmt.Sprintf("Bearer realm=\"%s\"", a.SystemIdentifier))
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		authParts := strings.SplitN(authHeader[0], " ", 2)
		if authParts[0] != "Bearer" || len(authParts) != 2 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			w.Header().Add("WWW-Authenticate", fmt.Sprintf("Bearer realm=\"%s\", error=\"invalid_request\", error=\"Incomplete or incompatible authentication provided.\"", a.SystemIdentifier))
			return
		}

		uid, err := a.Verify(authParts[1])
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			w.Header().Add("WWW-Authenticate", fmt.Sprintf("Bearer realm=\"%s\", error=\"invalid_token\", error=\"Invalid credential.\"", a.SystemIdentifier))
			return
		}

		nextR := r.WithContext(context.WithValue(r.Context(), auth.UserIdentityContextKey, uid))
		next.ServeHTTP(w, nextR)
	})
}

func (a Authenticator) AuthenticationType() any {
	return auth.AuthenticatorType{
		Type: "jwt",
	}
}

func (a Authenticator) Sign(uid string) (string, error) {
	id := uuid.New().String()

	iss := clock()
	exp := iss.Add(a.TTL)

	claims := jwt.StandardClaims{
		Id: id,

		Issuer:  a.SystemIdentifier,
		Subject: uid,

		IssuedAt:  iss.Unix(),
		ExpiresAt: exp.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = a.KeyIdentifier

	return token.SignedString(a.PrivateKey)
}

func (a Authenticator) Verify(jwtString string) (string, error) {
	token, err := jwt.ParseWithClaims(jwtString, &jwt.StandardClaims{}, a.keyLookup)
	if err != nil {
		return "", fmt.Errorf("failed to parse and verify signature in token: %w", err)
	}

	claims := token.Claims.(*jwt.StandardClaims)
	if !claims.VerifyIssuer(a.SystemIdentifier, true) {
		return "", fmt.Errorf("JWT is not for this system")
	}

	return claims.Subject, nil
}

func (a Authenticator) keyLookup(token *jwt.Token) (any, error) {
	if token.Header["alg"] != "ES256" {
		return nil, errors.New("unacceptable algorithm in JWT")
	}

	if kid, found := token.Header["kid"]; found && kid == a.KeyIdentifier {
		return a.PrivateKey.Public(), nil
	}

	return nil, errors.New("no public key found for token")
}
