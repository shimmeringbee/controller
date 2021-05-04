package auth

import "net/http"

const UserIdentityContextKey = "AuthenticatedUserIdentity"

type AuthenticationProvider interface {
	AuthenticationMiddleware(next http.Handler) http.Handler
	AuthenticationRouter() http.Handler
	AuthenticationType() interface{}
}

type AuthenticatorType struct {
	Type string `json:"type"`
}
