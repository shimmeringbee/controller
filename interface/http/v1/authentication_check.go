package v1

import (
	"encoding/json"
	"github.com/shimmeringbee/controller/interface/http/auth"
	"net/http"
)

type AuthenticationCheckPayload struct {
	Authenticated bool   `json:"authenticated"`
	Identity      string `json:"identity,omitempty"`
}

func authenticationCheck(w http.ResponseWriter, r *http.Request) {
	var identity string
	var ok bool

	identityRaw := r.Context().Value(auth.UserIdentityContextKey)
	if identityRaw != nil {
		identity, ok = identityRaw.(string)
		if !ok {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	authenticated := len(identity) > 0

	payload := AuthenticationCheckPayload{
		Authenticated: authenticated,
		Identity:      identity,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
