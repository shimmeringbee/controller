package external

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/interface/http/auth"
	"net/http"
)

var _ auth.AuthenticationProvider = (*Authenticator)(nil)

type Authenticator struct {
	UserHeader string
}

const HttpUserHeader string = "HTTP_USER"

func (a Authenticator) AuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := r.Header.Get(a.UserHeader)
		if len(user) == 0 {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), auth.UserIdentityContextKey, user)))
	})
}

func (a Authenticator) AuthenticationRouter() http.Handler {
	return mux.NewRouter()
}

func (a Authenticator) AuthenticationType() any {
	return auth.AuthenticatorType{
		Type: "external",
	}
}
