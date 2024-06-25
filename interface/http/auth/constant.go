package auth

import "net/http"

const UserIdentityContextKey = "AuthenticatedUserIdentity"

type AuthenticationProvider interface {
	AuthenticationMiddleware(next http.Handler) http.Handler
	AuthenticationRouter() http.Handler
	AuthenticationType() any
}

type AuthenticatorType struct {
	Type string `json:"type"`
}
