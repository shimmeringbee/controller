package external

import (
	"github.com/shimmeringbee/controller/interface/http/auth"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthenticator_AuthenticationMiddleware(t *testing.T) {
	t.Run("verifies that the external authenticator sets a user identity when HTTP_USER is set", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		expectedUser := "doctor"
		req.Header.Add("HTTP_USER", expectedUser)

		a := Authenticator{UserHeader: HttpUserHeader}

		handler := a.AuthenticationMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, expectedUser, request.Context().Value(auth.UserIdentityContextKey))
			writer.WriteHeader(200)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("verifies that 401 is returned when HTTP_USER is not set", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		a := Authenticator{UserHeader: HttpUserHeader}

		handler := a.AuthenticationMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			t.Fatal("Downstream handler called, and should not have been.")
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}
