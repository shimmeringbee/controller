package null

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/interface/http/auth"
	"net/http"
)

var _ auth.AuthenticationProvider = (*Authenticator)(nil)

type Authenticator struct{}

func (a Authenticator) AuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), auth.UserIdentityContextKey, "NullAuthentication")))
	})
}

func (a Authenticator) AuthenticationRouter() http.Handler {
	return mux.NewRouter()
}
