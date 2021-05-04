package null

import (
	"github.com/shimmeringbee/controller/interface/http/auth"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthenticator_AuthenticationMiddleware(t *testing.T) {
	t.Run("verifies that the null authenticator sets a user identity", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		a := Authenticator{}

		handler := a.AuthenticationMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "NullAuthentication", request.Context().Value(auth.UserIdentityContextKey))
			writer.WriteHeader(200)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
