package v1

import (
	"context"
	"github.com/shimmeringbee/controller/interface/http/auth"
	"github.com/shimmeringbee/controller/interface/http/auth/null"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_authenticationCheck(t *testing.T) {
	t.Run("returns a not authenticated value if there is no user identity", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/auth/check", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		authenticationCheck(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "{\"authenticated\":false}", rr.Body.String())
	})

	t.Run("returns a authenticated value if there is a user identity", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/auth/check", nil)
		if err != nil {
			t.Fatal(err)
		}

		req = req.WithContext(context.WithValue(req.Context(), auth.UserIdentityContextKey, "username"))
		rr := httptest.NewRecorder()

		authenticationCheck(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "{\"authenticated\":true,\"identity\":\"username\"}", rr.Body.String())
	})
}

func Test_authenticationType(t *testing.T) {
	t.Run("returns the authentication type data marshalled as JSON", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/auth/type", nil)
		if err != nil {
			t.Fatal(err)
		}

		req = req.WithContext(context.WithValue(req.Context(), auth.UserIdentityContextKey, "username"))
		rr := httptest.NewRecorder()

		authenticationType(null.Authenticator{})(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "{\"type\":\"null\"}", rr.Body.String())
	})
}
